package observer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/leader"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
)

var log = logf.Log.WithName("log_observer")

type LogObserver interface {
	Start(ctx context.Context) error
}

type logObserver struct {
	client     client.Client
	kubeClient kubernetes.Interface
	stopCh     chan struct{}
	isLeader   bool
}

type LogEntry struct {
	Dataset string `json:"dataset"`
	Type    string `json:"type"`
	Msg     string `json:"msg"`
	Size    int64  `json:"size"`
	Total   int64  `json:"total"`
	Unit    string `json:"unit"`
	Hash    string `json:"hash,omitempty"`
	Level   string `json:"level"`
}

func NewLogObserver(c client.Client, kubeClient kubernetes.Interface) LogObserver {
	return &logObserver{
		client:     c,
		kubeClient: kubeClient,
		stopCh:     make(chan struct{}),
	}
}

func (o *logObserver) Start(ctx context.Context) error {
	log.Info("Starting log observer - waiting for leadership")

	go func() {
		err := leader.Become(ctx, "log-observer-lock")
		if err != nil {
			log.Error(err, "Failed to acquire leadership lock")
			return
		}

		log.Info("Acquired leadership - starting log observer")
		o.isLeader = true
		o.run(ctx)
	}()

	return nil
}

func (o *logObserver) Stop() {
	log.Info("Stopping log observer")
	close(o.stopCh)
}

func (o *logObserver) run(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !o.isLeader {
				log.V(4).Info("Not the leader, skipping log observation")
				continue
			}

			if err := o.updateProteinDatabaseStatus(ctx); err != nil {
				log.Error(err, "Error updating ProteinDatabase status")
			}
		case <-o.stopCh:
			log.Info("Log observer stopped")
			return
		case <-ctx.Done():
			log.Info("Context done, stopping log observer")
			return
		}
	}
}

func (o *logObserver) updateProteinDatabaseStatus(ctx context.Context) error {
	proteinDBList := &datav1.ProteinDatabaseList{}
	if err := o.client.List(ctx, proteinDBList); err != nil {
		return fmt.Errorf("failed to list ProteinDatabase resources: %w", err)
	}

	for i := range proteinDBList.Items {
		proteinDB := &proteinDBList.Items[i]
		if err := o.updateSingleProteinDBStatus(ctx, proteinDB); err != nil {
			log.Error(err, "Failed to update ProteinDatabase status",
				"name", proteinDB.Name, "namespace", proteinDB.Namespace)
		}
	}

	return nil
}

func (o *logObserver) updateSingleProteinDBStatus(ctx context.Context, proteinDB *datav1.ProteinDatabase) error {
	jobList := &v1.JobList{}
	labelSelector := client.MatchingLabels{
		"app.kubernetes.io/instance":   proteinDB.Name,
		"app.kubernetes.io/managed-by": "kubefold-operator",
	}
	if err := o.client.List(ctx, jobList, labelSelector, client.InNamespace(proteinDB.Namespace)); err != nil {
		return fmt.Errorf("failed to list jobs for ProteinDatabase %s: %w", proteinDB.Name, err)
	}

	var totalSize, totalBytes int64
	allCompleted := true
	datasetCount := 0

	for i := range jobList.Items {
		job := &jobList.Items[i]

		if job.Status.CompletionTime == nil {
			allCompleted = false
		}

		podList := &corev1.PodList{}
		podLabelSelector := client.MatchingLabels{
			"job-name": job.Name,
		}
		if err := o.client.List(ctx, podList, podLabelSelector, client.InNamespace(job.Namespace)); err != nil {
			log.Error(err, "Failed to list pods for job", "job", job.Name)
			continue
		}

		for j := range podList.Items {
			pod := &podList.Items[j]
			if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
				continue
			}

			req := o.kubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
				Container: "downloader",
				Follow:    false,
				TailLines: int64Ptr(100),
			})

			podLogs, err := req.Stream(ctx)
			if err != nil {
				log.Error(err, "Failed to stream logs for pod", "pod", pod.Name)
				continue
			}

			jobSize, jobTotal, err := o.processLogStreamForTotals(podLogs)
			podLogs.Close()

			if err != nil {
				log.Error(err, "Error processing log stream")
				continue
			}

			if jobTotal > 0 {
				totalSize += jobSize
				totalBytes += jobTotal
				datasetCount++
			}
		}
	}

	if datasetCount > 0 {
		proteinDBCopy := proteinDB.DeepCopy()

		var progressStr string
		if totalBytes > 0 {
			progressPercent := float64(totalSize) / float64(totalBytes) * 100
			progressStr = fmt.Sprintf("%.1f%%", progressPercent)
		} else {
			progressStr = "0%"
		}

		sizeStr := humanReadableSize(totalBytes)

		if allCompleted && datasetCount > 0 {
			proteinDBCopy.Status.DownloadStatus = datav1.ProteinDatabaseDownloadStatusCompleted
		} else if totalSize > 0 {
			proteinDBCopy.Status.DownloadStatus = datav1.ProteinDatabaseDownloadStatusDownloading
		} else {
			proteinDBCopy.Status.DownloadStatus = datav1.ProteinDatabaseDownloadStatusNotStarted
		}

		proteinDBCopy.Status.Progress = progressStr
		proteinDBCopy.Status.Size = sizeStr

		if proteinDBCopy.Status.Progress != proteinDB.Status.Progress ||
			proteinDBCopy.Status.Size != proteinDB.Status.Size ||
			proteinDBCopy.Status.DownloadStatus != proteinDB.Status.DownloadStatus {

			if err := o.client.Status().Update(ctx, proteinDBCopy); err != nil {
				if errors.IsConflict(err) {
					log.V(1).Info("Conflict updating ProteinDatabase status, will retry",
						"name", proteinDB.Name, "namespace", proteinDB.Namespace)
					return nil
				}
				return fmt.Errorf("failed to update ProteinDatabase status: %w", err)
			}

			log.Info("Updated ProteinDatabase status",
				"name", proteinDB.Name,
				"namespace", proteinDB.Namespace,
				"progress", progressStr,
				"size", sizeStr)
		}
	}

	return nil
}

func (o *logObserver) processLogStreamForTotals(logStream io.ReadCloser) (size int64, total int64, err error) {
	scanner := bufio.NewScanner(logStream)

	var latestSize, latestTotal int64
	foundEntry := false

	for scanner.Scan() {
		line := scanner.Text()
		var logEntry LogEntry
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			continue
		}

		if logEntry.Type != "download" {
			continue
		}

		latestSize = logEntry.Size
		latestTotal = logEntry.Total
		foundEntry = true
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("error reading log stream: %w", err)
	}

	if !foundEntry {
		return 0, 0, nil
	}

	return latestSize, latestTotal, nil
}

func int64Ptr(i int64) *int64 {
	return &i
}

func humanReadableSize(bytes int64) string {
	const (
		_        = iota
		KB int64 = 1 << (10 * iota)
		MB
		GB
		TB
	)

	unit := ""
	value := float64(bytes)

	switch {
	case bytes >= TB:
		unit = "TB"
		value = float64(bytes) / float64(TB)
	case bytes >= GB:
		unit = "GB"
		value = float64(bytes) / float64(GB)
	case bytes >= MB:
		unit = "MB"
		value = float64(bytes) / float64(MB)
	case bytes >= KB:
		unit = "KB"
		value = float64(bytes) / float64(KB)
	default:
		unit = "B"
	}

	return fmt.Sprintf("%.2f %s", value, unit)
}
