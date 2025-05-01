package observer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kubefold/operator/internal/util"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	downloaderTypes "github.com/kubefold/downloader/pkg/types"
	datav1 "github.com/kubefold/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("proteindatabase_log_observer")

type LogObserver interface {
	Start(ctx context.Context) error
}

type logObserver struct {
	client     client.Client
	kubeClient kubernetes.Interface
	stopCh     chan struct{}
}

type LogEntry struct {
	DatasetName string    `json:"dataset"`
	Type        string    `json:"type"`
	Msg         string    `json:"msg"`
	Size        int64     `json:"size"`
	Total       int64     `json:"total"`
	Unit        string    `json:"unit"`
	Hash        string    `json:"hash,omitempty"`
	Level       string    `json:"level"`
	Time        time.Time `json:"time,omitempty"`
}

func (l *LogEntry) Dataset() downloaderTypes.Dataset {
	return downloaderTypes.Dataset(l.DatasetName)
}

func NewLogObserver(c client.Client, kubeClient kubernetes.Interface) LogObserver {
	return &logObserver{
		client:     c,
		kubeClient: kubeClient,
		stopCh:     make(chan struct{}),
	}
}

func (o *logObserver) Start(ctx context.Context) error {
	go o.run(ctx)
	return nil
}

func (o *logObserver) Stop() {
	close(o.stopCh)
}

func (o *logObserver) run(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
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
	var proteinDatabases datav1.ProteinDatabaseList
	if err := o.client.List(ctx, &proteinDatabases); err != nil {
		return fmt.Errorf("failed to list protein databases: %w", err)
	}

	for i := range proteinDatabases.Items {
		pd := &proteinDatabases.Items[i]
		if err := o.processProteinDatabase(ctx, pd); err != nil {
			log.Error(err, "Error processing ProteinDatabase", "name", pd.Name, "namespace", pd.Namespace)
			continue
		}
	}

	return nil
}

func (o *logObserver) processProteinDatabase(ctx context.Context, pd *datav1.ProteinDatabase) error {
	pods, err := o.findDownloadPods(ctx, pd)
	if err != nil {
		return fmt.Errorf("failed to find download pods: %w", err)
	}
	if len(pods) == 0 {
		return nil
	}

	proteinDatabaseStatus := pd.Status.DeepCopy()
	for _, pod := range pods {
		if err := o.processPodsLogs(ctx, pod, proteinDatabaseStatus); err != nil {
			log.Error(err, "Error processing pod logs", "pod", pod.Name, "namespace", pod.Namespace)
			continue
		}
	}

	return o.updateStatus(ctx, pd, proteinDatabaseStatus)
}

func (o *logObserver) findDownloadPods(ctx context.Context, pd *datav1.ProteinDatabase) ([]corev1.Pod, error) {
	var podList corev1.PodList
	err := o.client.List(ctx, &podList,
		&client.ListOptions{
			Namespace: pd.Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"data.kubefold.io/database": pd.Name,
			}),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	return podList.Items, nil
}

func (o *logObserver) processPodsLogs(ctx context.Context, pod corev1.Pod, proteinDatabaseStatus *datav1.ProteinDatabaseStatus) error {
	podLogOpts := corev1.PodLogOptions{
		Container: "downloader",
		TailLines: util.Int64Ptr(100),
		Follow:    false,
	}

	req := o.kubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to open pod log stream: %w", err)
	}
	defer podLogs.Close()

	reader := bufio.NewReader(podLogs)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading pod logs: %w", err)
		}

		var logEntry LogEntry
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			continue
		}

		if logEntry.DatasetName != "" {
			var downloadStatus datav1.ProteinDatabaseDownloadStatus
			if logEntry.Size == 0 {
				downloadStatus = datav1.ProteinDatabaseDownloadStatusNotStarted
			} else if logEntry.Size > 0 && logEntry.Size < logEntry.Dataset().Size() {
				downloadStatus = datav1.ProteinDatabaseDownloadStatusDownloading
			} else if logEntry.Size == logEntry.Dataset().Size() {
				downloadStatus = datav1.ProteinDatabaseDownloadStatusCompleted
			}

			progress := datav1.ProteinDatabaseDatasetDownloadProgress{
				DownloadStatus: downloadStatus,
				Size:           logEntry.Size,
				TotalSize:      logEntry.Dataset().Size(),
				Progress:       util.FormatPercentage(logEntry.Size, logEntry.Dataset().Size()),
				LastUpdate:     &metav1.Time{Time: logEntry.Time},
			}

			switch logEntry.Dataset() {
			case downloaderTypes.DatasetRFam:
				if proteinDatabaseStatus.Datasets.RFam.LastUpdate != nil {
					progress.Delta = progress.Size - proteinDatabaseStatus.Datasets.RFam.Size
					progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Time.Sub(proteinDatabaseStatus.Datasets.RFam.LastUpdate.Time)}
					progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
				}
				if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
					progress.Delta = 0
					progress.DeltaDuration = nil
					progress.DownloadSpeed = ""
				}
				proteinDatabaseStatus.Datasets.RFam = progress
				break
			case downloaderTypes.DatasetBFD:
				break
			}

		}
	}

	return nil
}

//func (o *logObserver) aggregateDownloadMetrics(proteinDatabaseStatus *datav1.ProteinDatabaseStatus) {
//	var size int64
//	var totalSize int64
//	var downloadSpeed string
//}

func (o *logObserver) updateStatus(ctx context.Context, pd *datav1.ProteinDatabase, proteinDatabaseStatus *datav1.ProteinDatabaseStatus) error {
	proteinDatabase := &datav1.ProteinDatabase{}
	if err := o.client.Get(ctx, types.NamespacedName{Name: pd.Name, Namespace: pd.Namespace}, proteinDatabase); err != nil {
		return fmt.Errorf("failed to get latest ProteinDatabase: %w", err)
	}

	proteinDatabase.Status = *proteinDatabaseStatus
	proteinDatabase.Status.LastUpdate = util.GetNow()

	if err := o.client.Status().Update(ctx, proteinDatabase); err != nil {
		return fmt.Errorf("failed to update ProteinDatabase status: %w", err)
	}

	return nil
}
