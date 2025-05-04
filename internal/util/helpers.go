package util

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Int64Ptr(i int64) *int64 {
	return &i
}

func GetNow() *metav1.Time {
	return &metav1.Time{Time: time.Now()}
}
