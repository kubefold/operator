package observer

import (
	"bufio"
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubefold/operator/internal/util"
)

const downloaderContainer = "downloader"

type PodLogReader interface {
	StreamLines(ctx context.Context, pod corev1.Pod, tailLines int64) ([]LogEntry, error)
}

type podLogReader struct {
	kubeClient kubernetes.Interface
}

func NewPodLogReader(kubeClient kubernetes.Interface) PodLogReader {
	return &podLogReader{kubeClient: kubeClient}
}

func (r *podLogReader) StreamLines(ctx context.Context, pod corev1.Pod, tailLines int64) ([]LogEntry, error) {
	options := corev1.PodLogOptions{
		Container: downloaderContainer,
		TailLines: util.Int64Ptr(tailLines),
		Follow:    false,
	}
	request := r.kubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &options)
	stream, err := request.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open pod log stream: %w", err)
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			log.Error(closeErr, "failed to close pod log stream", "pod", pod.Name)
		}
	}()

	reader := bufio.NewReader(stream)
	entries := make([]LogEntry, 0)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			if entry, ok := ParseLogEntry([]byte(line)); ok {
				entries = append(entries, entry)
			}
		}
		if err != nil {
			if err == io.EOF {
				return entries, nil
			}
			return nil, fmt.Errorf("error reading pod logs: %w", err)
		}
	}
}
