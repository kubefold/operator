package database

import (
	"context"
	"strings"

	downloaderTypes "github.com/kubefold/downloader/pkg/types"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

const (
	DownloaderImage     = "ghcr.io/kubefold/downloader"
	databaseMountPath   = "/public_databases"
	downloaderContainer = "downloader"
)

type JobReconciler interface {
	EnsureAll(ctx context.Context, database *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim) error
}

type jobReconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	enumerator DatasetEnumerator
}

func NewJobReconciler(c client.Client, scheme *runtime.Scheme, enumerator DatasetEnumerator) JobReconciler {
	return &jobReconciler{client: c, scheme: scheme, enumerator: enumerator}
}

func (j *jobReconciler) EnsureAll(ctx context.Context, database *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim) error {
	log := logf.FromContext(ctx)
	for _, dataset := range j.enumerator.FromSpec(database) {
		if err := j.ensureSingle(ctx, database, pvc, dataset); err != nil {
			log.Error(err, "Failed to ensure job for dataset", "dataset", dataset)
			return err
		}
	}
	return nil
}

func (j *jobReconciler) ensureSingle(ctx context.Context, database *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim, dataset downloaderTypes.Dataset) error {
	log := logf.FromContext(ctx)
	jobName := normalizeJobName(shared.DownloaderJobName(database.Name, dataset.ShortName()))

	existing := &batchv1.Job{}
	err := j.client.Get(ctx, types.NamespacedName{Name: jobName, Namespace: database.Namespace}, existing)
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	job := j.build(database, pvc, dataset, jobName)
	if err := controllerutil.SetControllerReference(database, job, j.scheme); err != nil {
		return err
	}
	log.Info("Creating downloader job", "jobName", jobName, "dataset", dataset)
	if err := j.client.Create(ctx, job); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (j *jobReconciler) build(database *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim, dataset downloaderTypes.Dataset, jobName string) *batchv1.Job {
	labels := shared.DownloaderJobLabels(database.Name, dataset.ShortName())
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: database.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:            downloaderContainer,
							Image:           DownloaderImage,
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{Name: "DATASET", Value: dataset.String()},
								{Name: "DESTINATION", Value: databaseMountPath},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "databases", MountPath: databaseMountPath},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "databases",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvc.Name,
								},
							},
						},
					},
				},
			},
		},
	}
}

func normalizeJobName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}
