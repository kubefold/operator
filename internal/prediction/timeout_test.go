package prediction

import (
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTimeoutCheckerNoStartTime(t *testing.T) {
	checker := &timeoutChecker{now: func() time.Time { return time.Unix(0, 0) }}
	job := &batchv1.Job{}
	if checker.IsTimedOut(job, time.Hour) {
		t.Fatal("job without StartTime must not be timed out")
	}
}

func TestTimeoutCheckerExceededDefault(t *testing.T) {
	start := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	checker := &timeoutChecker{now: func() time.Time { return start.Add(2 * time.Hour) }}
	job := &batchv1.Job{
		Status: batchv1.JobStatus{StartTime: &metav1.Time{Time: start}},
	}
	if !checker.IsTimedOut(job, time.Hour) {
		t.Fatal("expected timeout after 2h when default is 1h")
	}
}

func TestTimeoutCheckerWithinDefault(t *testing.T) {
	start := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	checker := &timeoutChecker{now: func() time.Time { return start.Add(30 * time.Minute) }}
	job := &batchv1.Job{
		Status: batchv1.JobStatus{StartTime: &metav1.Time{Time: start}},
	}
	if checker.IsTimedOut(job, time.Hour) {
		t.Fatal("must not be timed out before threshold")
	}
}

func TestTimeoutCheckerUsesActiveDeadlineSeconds(t *testing.T) {
	start := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	checker := &timeoutChecker{now: func() time.Time { return start.Add(2 * time.Minute) }}
	deadline := int64(60)
	job := &batchv1.Job{
		Spec:   batchv1.JobSpec{ActiveDeadlineSeconds: &deadline},
		Status: batchv1.JobStatus{StartTime: &metav1.Time{Time: start}},
	}
	if !checker.IsTimedOut(job, 24*time.Hour) {
		t.Fatal("ActiveDeadlineSeconds=60 must take precedence over default 24h")
	}
}
