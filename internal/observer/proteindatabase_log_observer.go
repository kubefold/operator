package observer

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/database"
	"github.com/kubefold/operator/internal/shared"
)

const (
	defaultInterval  = 500 * time.Millisecond
	defaultTailLines = int64(100)
	databaseLabelKey = shared.LabelDatabase
)

var log = logf.Log.WithName("proteindatabase_log_observer")

type LogObserver interface {
	Start(ctx context.Context) error
}

type Option func(*logObserver)

func WithInterval(interval time.Duration) Option {
	return func(o *logObserver) {
		o.interval = interval
	}
}

func WithTailLines(tailLines int64) Option {
	return func(o *logObserver) {
		o.tailLines = tailLines
	}
}

type logObserver struct {
	client     client.Client
	kubeClient kubernetes.Interface
	reader     PodLogReader
	progress   ProgressUpdater
	publisher  StatusPublisher
	interval   time.Duration
	tailLines  int64
	waitGroup  sync.WaitGroup
}

func NewLogObserver(c client.Client, kubeClient kubernetes.Interface, options ...Option) LogObserver {
	enumerator := database.NewDatasetEnumerator()
	o := &logObserver{
		client:     c,
		kubeClient: kubeClient,
		reader:     NewPodLogReader(kubeClient),
		progress:   NewProgressUpdater(),
		publisher:  NewStatusPublisher(c, NewMetricsAggregator(enumerator)),
		interval:   defaultInterval,
		tailLines:  defaultTailLines,
	}
	for _, apply := range options {
		apply(o)
	}
	return o
}

func (o *logObserver) Start(ctx context.Context) error {
	log.Info("Log observer started", "interval", o.interval)
	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			o.tick(ctx)
		case <-ctx.Done():
			log.Info("Context done, stopping log observer")
			o.waitGroup.Wait()
			return nil
		}
	}
}

func (o *logObserver) tick(ctx context.Context) {
	o.waitGroup.Add(1)
	defer o.waitGroup.Done()
	if err := o.updateAllStatuses(ctx); err != nil {
		log.Error(err, "Error updating ProteinDatabase statuses")
	}
}

func (o *logObserver) updateAllStatuses(ctx context.Context) error {
	databases := &datav1.ProteinDatabaseList{}
	if err := o.client.List(ctx, databases); err != nil {
		return fmt.Errorf("failed to list protein databases: %w", err)
	}
	for i := range databases.Items {
		databasePointer := &databases.Items[i]
		if err := o.processDatabase(ctx, databasePointer); err != nil {
			log.Error(err, "Error processing ProteinDatabase", "name", databasePointer.Name, "namespace", databasePointer.Namespace)
			continue
		}
	}
	return nil
}

func (o *logObserver) processDatabase(ctx context.Context, db *datav1.ProteinDatabase) error {
	pods, err := o.findDownloadPods(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to find download pods: %w", err)
	}
	if len(pods) == 0 {
		return nil
	}

	newStatus := db.Status.DeepCopy()
	for _, pod := range pods {
		entries, err := o.reader.StreamLines(ctx, pod, o.tailLines)
		if err != nil {
			log.Error(err, "Error reading pod logs", "pod", pod.Name)
			continue
		}
		for _, entry := range entries {
			o.progress.Update(newStatus, entry)
		}
	}
	return o.publisher.Publish(ctx, db, newStatus)
}

func (o *logObserver) findDownloadPods(ctx context.Context, db *datav1.ProteinDatabase) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	err := o.client.List(ctx, podList,
		&client.ListOptions{
			Namespace: db.Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				databaseLabelKey: db.Name,
			}),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	return podList.Items, nil
}
