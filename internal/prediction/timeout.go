package prediction

import (
	"time"

	batchv1 "k8s.io/api/batch/v1"
)

const DefaultJobTimeout = 24 * time.Hour

type TimeoutChecker interface {
	IsTimedOut(job *batchv1.Job, defaultTimeout time.Duration) bool
}

type timeoutChecker struct {
	now func() time.Time
}

func NewTimeoutChecker() TimeoutChecker {
	return &timeoutChecker{now: time.Now}
}

func (c *timeoutChecker) IsTimedOut(job *batchv1.Job, defaultTimeout time.Duration) bool {
	if job.Status.StartTime == nil {
		return false
	}
	timeout := defaultTimeout
	if job.Spec.ActiveDeadlineSeconds != nil {
		timeout = time.Duration(*job.Spec.ActiveDeadlineSeconds) * time.Second
	}
	return c.now().Sub(job.Status.StartTime.Time) > timeout
}
