package controller

import (
	"context"
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strings"

	datav1 "github.com/kubefold/operator/api/v1"
)

type JobHandler struct {
	client client.Client
	scheme *runtime.Scheme
}

func (j *JobHandler) ensureJobs(ctx context.Context, pd *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim) error {
	log := logf.FromContext(ctx)

	for _, dataset := range getDatasets(pd) {
		if err := j.ensureJob(ctx, pd, pvc, dataset); err != nil {
			log.Error(err, "Failed to ensure job for dataset", "dataset", dataset)
			return err
		}
	}

	return nil
}

func (j *JobHandler) ensureJob(ctx context.Context, pd *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim, dataset Dataset) error {
	log := logf.FromContext(ctx)
	jobName := strings.ReplaceAll(strings.ToLower(fmt.Sprintf("%s-%s-downloader", pd.Name, dataset.ShortName())), "_", "-")

	existingJob := &batchv1.Job{}
	err := j.client.Get(ctx, types.NamespacedName{Name: jobName, Namespace: pd.Namespace}, existingJob)
	if err == nil {
		log.Info("Job already exists", "jobName", jobName)
		return nil
	}

	if !errors.IsNotFound(err) {
		return err
	}

	job := j.createJobSpec(pd, pvc, dataset, jobName)

	if err := controllerutil.SetControllerReference(pd, job, j.scheme); err != nil {
		return err
	}

	log.Info("Creating downloader job", "jobName", jobName, "dataset", dataset)
	if err := j.client.Create(ctx, job); err != nil {
		return err
	}

	return nil
}

func (j *JobHandler) createJobSpec(pd *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim, dataset Dataset, jobName string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: pd.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "proteindatabase-downloader",
				"app.kubernetes.io/instance":   pd.Name,
				"app.kubernetes.io/managed-by": "kubefold-operator",
				"kubefold.io/dataset":          dataset.ShortName(),
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:            "downloader",
							Image:           DownloaderImage,
							ImagePullPolicy: DownloaderImagePullPolicy,
							Env: []corev1.EnvVar{
								{
									Name:  "DATASET",
									Value: dataset.String(),
								},
								{
									Name:  "DESTINATION",
									Value: "/public_databases",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "databases",
									MountPath: "/public_databases",
								},
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
