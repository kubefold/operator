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
			case downloaderTypes.DatasetBFD:
				if proteinDatabaseStatus.Datasets.BFD.LastUpdate != nil {
					progress.Delta = progress.Size - proteinDatabaseStatus.Datasets.BFD.Size
					progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Time.Sub(proteinDatabaseStatus.Datasets.BFD.LastUpdate.Time)}
					progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
				}
				if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
					progress.Delta = 0
					progress.DeltaDuration = nil
					progress.DownloadSpeed = ""
				}
				proteinDatabaseStatus.Datasets.BFD = progress
			case downloaderTypes.DatasetUniProt:
				if proteinDatabaseStatus.Datasets.UniProt.LastUpdate != nil {
					progress.Delta = progress.Size - proteinDatabaseStatus.Datasets.UniProt.Size
					progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Time.Sub(proteinDatabaseStatus.Datasets.UniProt.LastUpdate.Time)}
					progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
				}
				if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
					progress.Delta = 0
					progress.DeltaDuration = nil
					progress.DownloadSpeed = ""
				}
				proteinDatabaseStatus.Datasets.UniProt = progress
			case downloaderTypes.DatasetUniRef90:
				if proteinDatabaseStatus.Datasets.UniRef90.LastUpdate != nil {
					progress.Delta = progress.Size - proteinDatabaseStatus.Datasets.UniRef90.Size
					progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Time.Sub(proteinDatabaseStatus.Datasets.UniRef90.LastUpdate.Time)}
					progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
				}
				if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
					progress.Delta = 0
					progress.DeltaDuration = nil
					progress.DownloadSpeed = ""
				}
				proteinDatabaseStatus.Datasets.UniRef90 = progress
			case downloaderTypes.DatasetRNACentral:
				if proteinDatabaseStatus.Datasets.RNACentral.LastUpdate != nil {
					progress.Delta = progress.Size - proteinDatabaseStatus.Datasets.RNACentral.Size
					progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Time.Sub(proteinDatabaseStatus.Datasets.RNACentral.LastUpdate.Time)}
					progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
				}
				if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
					progress.Delta = 0
					progress.DeltaDuration = nil
					progress.DownloadSpeed = ""
				}
				proteinDatabaseStatus.Datasets.RNACentral = progress
			case downloaderTypes.DatasetPDB:
				if proteinDatabaseStatus.Datasets.PDB.LastUpdate != nil {
					progress.Delta = progress.Size - proteinDatabaseStatus.Datasets.PDB.Size
					progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Time.Sub(proteinDatabaseStatus.Datasets.PDB.LastUpdate.Time)}
					progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
				}
				if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
					progress.Delta = 0
					progress.DeltaDuration = nil
					progress.DownloadSpeed = ""
				}
				proteinDatabaseStatus.Datasets.PDB = progress
			case downloaderTypes.DatasetPDBSeqReq:
				if proteinDatabaseStatus.Datasets.PDBSeqReq.LastUpdate != nil {
					progress.Delta = progress.Size - proteinDatabaseStatus.Datasets.PDBSeqReq.Size
					progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Time.Sub(proteinDatabaseStatus.Datasets.PDBSeqReq.LastUpdate.Time)}
					progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
				}
				if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
					progress.Delta = 0
					progress.DeltaDuration = nil
					progress.DownloadSpeed = ""
				}
				proteinDatabaseStatus.Datasets.PDBSeqReq = progress
			case downloaderTypes.DatasetNT:
				if proteinDatabaseStatus.Datasets.NT.LastUpdate != nil {
					progress.Delta = progress.Size - proteinDatabaseStatus.Datasets.NT.Size
					progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Time.Sub(proteinDatabaseStatus.Datasets.NT.LastUpdate.Time)}
					progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
				}
				if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
					progress.Delta = 0
					progress.DeltaDuration = nil
					progress.DownloadSpeed = ""
				}
				proteinDatabaseStatus.Datasets.NT = progress
			case downloaderTypes.DatasetMGYClusters:
				if proteinDatabaseStatus.Datasets.MGYClusters.LastUpdate != nil {
					progress.Delta = progress.Size - proteinDatabaseStatus.Datasets.MGYClusters.Size
					progress.DeltaDuration = &metav1.Duration{Duration: progress.LastUpdate.Time.Sub(proteinDatabaseStatus.Datasets.MGYClusters.LastUpdate.Time)}
					progress.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(progress.Delta, progress.DeltaDuration.Duration))
				}
				if progress.DownloadStatus == datav1.ProteinDatabaseDownloadStatusCompleted {
					progress.Delta = 0
					progress.DeltaDuration = nil
					progress.DownloadSpeed = ""
				}
				proteinDatabaseStatus.Datasets.MGYClusters = progress
			}
		}
	}

	return nil
}

func (o *logObserver) aggregateDownloadMetrics(proteinDatabaseStatus *datav1.ProteinDatabaseStatus) {
	var size int64
	var totalSize int64
	var delta int64
	var deltaDuration time.Duration

	progresses := []datav1.ProteinDatabaseDatasetDownloadProgress{
		proteinDatabaseStatus.Datasets.RFam,
		proteinDatabaseStatus.Datasets.BFD,
		proteinDatabaseStatus.Datasets.UniProt,
		proteinDatabaseStatus.Datasets.UniRef90,
		proteinDatabaseStatus.Datasets.RNACentral,
		proteinDatabaseStatus.Datasets.PDB,
		proteinDatabaseStatus.Datasets.PDBSeqReq,
		proteinDatabaseStatus.Datasets.NT,
		proteinDatabaseStatus.Datasets.MGYClusters,
	}

	for _, progress := range progresses {
		size += progress.Size
		totalSize += progress.TotalSize
		delta += progress.Delta
		if progress.DeltaDuration != nil {
			deltaDuration += progress.DeltaDuration.Duration
		}
	}

	proteinDatabaseStatus.Size = util.FormatSize(size)
	proteinDatabaseStatus.TotalSize = util.FormatSize(totalSize)
	if totalSize > 0 {
		proteinDatabaseStatus.Progress = util.FormatPercentage(size, totalSize)
	}
	proteinDatabaseStatus.DownloadSpeed = util.FormatSpeed(util.CalculateDownloadSpeed(delta, deltaDuration))

	if size == 0 {
		proteinDatabaseStatus.DownloadStatus = datav1.ProteinDatabaseDownloadStatusNotStarted
	} else if size > 0 && size < totalSize {
		proteinDatabaseStatus.DownloadStatus = datav1.ProteinDatabaseDownloadStatusDownloading
	} else if size == totalSize {
		proteinDatabaseStatus.DownloadStatus = datav1.ProteinDatabaseDownloadStatusCompleted
	}
}

func (o *logObserver) updateStatus(ctx context.Context, pd *datav1.ProteinDatabase, proteinDatabaseStatus *datav1.ProteinDatabaseStatus) error {
	proteinDatabase := &datav1.ProteinDatabase{}
	if err := o.client.Get(ctx, types.NamespacedName{Name: pd.Name, Namespace: pd.Namespace}, proteinDatabase); err != nil {
		return fmt.Errorf("failed to get latest ProteinDatabase: %w", err)
	}

	o.aggregateDownloadMetrics(proteinDatabaseStatus)

	proteinDatabase.Status = *proteinDatabaseStatus
	proteinDatabase.Status.LastUpdate = util.GetNow()

	if err := o.client.Status().Update(ctx, proteinDatabase); err != nil {
		return fmt.Errorf("failed to update ProteinDatabase status: %w", err)
	}

	return nil
}
