package observer

import (
	"context"
	"time"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("log_observer")

type LogObserver interface {
	Start(ctx context.Context) error
}

type logObserver struct {
	client     client.Client
	kubeClient kubernetes.Interface
	stopCh     chan struct{}
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
	log.Info("Starting log observer")
	go o.run(ctx)
	return nil
}

func (o *logObserver) Stop() {
	log.Info("Stopping log observer")
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
	panic("implement me")
}
