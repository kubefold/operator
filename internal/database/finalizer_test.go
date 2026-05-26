package database

import (
	"context"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := datav1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme corev1: %v", err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme batchv1: %v", err)
	}
	return scheme
}

func TestFinalizerHandleDeletionRemovesLabeledJobsAndPVC(t *testing.T) {
	scheme := newScheme(t)
	database := &datav1.ProteinDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "alpha",
			Namespace:  "default",
			Finalizers: []string{Finalizer},
		},
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alpha-bfd-downloader",
			Namespace: "default",
			Labels:    map[string]string{shared.LabelDatabase: "alpha"},
		},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shared.DatabasePVCName("alpha"),
			Namespace: "default",
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(database, job, pvc).Build()

	reconciler := NewFinalizerReconciler(c)
	if _, err := reconciler.HandleDeletion(context.Background(), database); err != nil {
		t.Fatalf("HandleDeletion failed: %v", err)
	}

	if err := c.Get(context.Background(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, &batchv1.Job{}); !errors.IsNotFound(err) {
		t.Fatalf("expected job to be deleted, got: %v", err)
	}
	if err := c.Get(context.Background(), types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &corev1.PersistentVolumeClaim{}); !errors.IsNotFound(err) {
		t.Fatalf("expected pvc to be deleted, got: %v", err)
	}
}

func TestFinalizerHandleDeletionSkipsPVCWithPreboundSelector(t *testing.T) {
	scheme := newScheme(t)
	database := &datav1.ProteinDatabase{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "beta",
			Namespace:  "default",
			Finalizers: []string{Finalizer},
		},
		Spec: datav1.ProteinDatabaseSpec{
			Volume: datav1.ProteinDatabaseVolume{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"role": "static"}},
			},
		},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shared.DatabasePVCName("beta"),
			Namespace: "default",
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(database, pvc).Build()

	reconciler := NewFinalizerReconciler(c)
	if _, err := reconciler.HandleDeletion(context.Background(), database); err != nil {
		t.Fatalf("HandleDeletion failed: %v", err)
	}

	if err := c.Get(context.Background(), types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, &corev1.PersistentVolumeClaim{}); err != nil {
		t.Fatalf("PVC bound to pre-existing PV must not be deleted: %v", err)
	}
}
