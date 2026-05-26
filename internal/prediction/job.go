package prediction

import (
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

const (
	ManagerImage     = "ghcr.io/kubefold/manager"
	AlphafoldImage   = "public.ecr.aws/k3x1v3b7/alphafold3:latest"
	containerData    = "data"
	containerDB      = "database"
	mountPathData    = "/data"
	mountPathDB      = "/public_databases"
	weightsURLEnvVar = "WEIGHTS_URL"
	weightsDownload  = `set -eu; mkdir -p /data/models; wget --tries=3 --timeout=30 -O /data/models/af3.bin.zst "$WEIGHTS_URL"; unzstd /data/models/af3.bin.zst`
	jobBackoffLimit  = int32(2)

	containerInputPlacement   = "input-placement"
	containerWeightsPlacement = "weights-placement"
	containerSearch           = "search"
	containerPredict          = "predict"
	containerUpload           = "upload"
	containerNotify           = "notify"
)

var disallowPrivilegeEscalation = false

type JobBuilder interface {
	BuildSearch(prediction *datav1.ProteinConformationPrediction, jobName, pvcName, encodedInput string) *batchv1.Job
	BuildPredict(prediction *datav1.ProteinConformationPrediction, jobName, pvcName, encodedInput string) *batchv1.Job
	BuildUpload(prediction *datav1.ProteinConformationPrediction, jobName, pvcName string) *batchv1.Job
}

type jobBuilder struct {
	nodeSelector NodeSelectorTranslator
}

func NewJobBuilder(nodeSelector NodeSelectorTranslator) JobBuilder {
	return &jobBuilder{nodeSelector: nodeSelector}
}

func (b *jobBuilder) BuildSearch(prediction *datav1.ProteinConformationPrediction, jobName, pvcName, encodedInput string) *batchv1.Job {
	backoffLimit := jobBackoffLimit
	labels := shared.PredictionLabels(prediction.Name, "proteinconformationprediction-search", phaseSearchKey)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: prediction.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{inputPlacementContainer(encodedInput)},
					Containers:     []corev1.Container{searchContainer(prediction.Spec.Job.Profile)},
					Volumes:        predictionVolumes(pvcName, prediction.Spec.Database),
				},
			},
		},
	}

	job.Spec.Template.Spec.Affinity = b.nodeSelector.ToAffinity(&prediction.Spec.Job.SearchNodeSelector)
	return job
}

func (b *jobBuilder) BuildPredict(prediction *datav1.ProteinConformationPrediction, jobName, pvcName, encodedInput string) *batchv1.Job {
	backoffLimit := jobBackoffLimit
	labels := shared.PredictionLabels(prediction.Name, "proteinconformationprediction-predict", phasePredictKey)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: prediction.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						inputPlacementContainer(encodedInput),
						weightsPlacementContainer(prediction.Spec.Model.Weights.HTTP),
					},
					Containers: []corev1.Container{predictContainer(prediction.Spec.Job.Profile)},
					Volumes:    predictionVolumes(pvcName, prediction.Spec.Database),
				},
			},
		},
	}

	job.Spec.Template.Spec.Affinity = b.nodeSelector.ToAffinity(&prediction.Spec.Job.PredictionNodeSelector)
	return job
}

func (b *jobBuilder) BuildUpload(prediction *datav1.ProteinConformationPrediction, jobName, pvcName string) *batchv1.Job {
	backoffLimit := jobBackoffLimit
	labels := shared.PredictionLabels(prediction.Name, "proteinconformationprediction-upload", phaseUploadKey)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: prediction.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						uploadContainer(prediction),
						notifyContainer(prediction),
					},
					Volumes: []corev1.Volume{predictionDataVolume(pvcName)},
				},
			},
		},
	}
}

func inputPlacementContainer(encodedInput string) corev1.Container {
	return corev1.Container{
		Name:            containerInputPlacement,
		Image:           ManagerImage,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: restrictedSecurityContext(),
		Env: []corev1.EnvVar{
			{Name: "INPUT_PATH", Value: "/data/af_input"},
			{Name: "OUTPUT_PATH", Value: "/data/af_output"},
			{Name: "ENCODED_INPUT", Value: encodedInput},
		},
		VolumeMounts: predictionVolumeMounts(),
	}
}

func weightsPlacementContainer(weightsURL string) corev1.Container {
	return corev1.Container{
		Name:            containerWeightsPlacement,
		Image:           ManagerImage,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: restrictedSecurityContext(),
		Command:         []string{"sh", "-c", weightsDownload},
		Env: []corev1.EnvVar{
			{Name: weightsURLEnvVar, Value: weightsURL},
		},
		VolumeMounts: predictionVolumeMounts(),
	}
}

func searchContainer(profile datav1.ProteinConformationPredictionProfile) corev1.Container {
	return corev1.Container{
		Name:            containerSearch,
		Image:           AlphafoldImage,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: restrictedSecurityContext(),
		Resources:       resourcesForPhase(profile, phaseSearchKey),
		Command:         []string{"uv"},
		Args: []string{
			"run", "python3", "run_alphafold.py",
			"--json_path=/data/af_input/fold_input.json",
			"--output_dir=/data/af_output",
			"--model_dir=/data/models",
			"--db_dir=/public_databases",
			"--run_inference=false",
		},
		VolumeMounts: predictionVolumeMounts(),
	}
}

func predictContainer(profile datav1.ProteinConformationPredictionProfile) corev1.Container {
	requirements := resourcesForPhase(profile, phasePredictKey)
	requirements.Requests["nvidia.com/gpu"] = resource.MustParse("1")
	if requirements.Limits == nil {
		requirements.Limits = corev1.ResourceList{}
	}
	requirements.Limits["nvidia.com/gpu"] = resource.MustParse("1")
	return corev1.Container{
		Name:            containerPredict,
		Image:           AlphafoldImage,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: restrictedSecurityContext(),
		Resources:       requirements,
		Command:         []string{"uv"},
		Args: []string{
			"run", "python3", "run_alphafold.py",
			"--json_path=/data/af_input/fold_input.json",
			"--output_dir=/data/af_output",
			"--model_dir=/data/models",
			"--db_dir=/public_databases",
			"--run_data_pipeline=false",
		},
		VolumeMounts: predictionVolumeMounts(),
	}
}

func uploadContainer(prediction *datav1.ProteinConformationPrediction) corev1.Container {
	return corev1.Container{
		Name:            containerUpload,
		Image:           ManagerImage,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: restrictedSecurityContext(),
		Resources:       resourcesForPhase(prediction.Spec.Job.Profile, phaseUploadKey),
		Env: []corev1.EnvVar{
			{Name: "INPUT_PATH", Value: "/data/af_input"},
			{Name: "OUTPUT_PATH", Value: "/data/af_output"},
			{Name: "BUCKET", Value: prediction.Spec.Destination.S3.Bucket},
			{Name: "AWS_REGION", Value: prediction.Spec.Destination.S3.Region},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: containerData, MountPath: mountPathData},
		},
	}
}

func notifyContainer(prediction *datav1.ProteinConformationPrediction) corev1.Container {
	return corev1.Container{
		Name:            containerNotify,
		Image:           ManagerImage,
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: restrictedSecurityContext(),
		Resources:       resourcesForPhase(prediction.Spec.Job.Profile, phaseUploadKey),
		Env: []corev1.EnvVar{
			{Name: "INPUT_PATH", Value: "/data/af_input"},
			{Name: "OUTPUT_PATH", Value: "/data/af_output"},
			{Name: "NOTIFICATION_PHONES", Value: strings.Join(prediction.Spec.Notifications.SMS, ",")},
			{Name: "NOTIFICATION_MESSAGE", Value: fmt.Sprintf("Protein Conformation Prediction %s in namespace %s completed. Artifacts has been uploaded to %s", prediction.Name, prediction.Namespace, prediction.Spec.Destination.S3.Bucket)},
			{Name: "AWS_REGION", Value: prediction.Spec.Destination.S3.Region},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: containerData, MountPath: mountPathData},
		},
	}
}

func predictionVolumes(pvcName, databaseName string) []corev1.Volume {
	return []corev1.Volume{
		predictionDataVolume(pvcName),
		{
			Name: containerDB,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: shared.DatabasePVCName(databaseName),
				},
			},
		},
	}
}

func predictionDataVolume(pvcName string) corev1.Volume {
	return corev1.Volume{
		Name: containerData,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		},
	}
}

func predictionVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: containerData, MountPath: mountPathData},
		{Name: containerDB, MountPath: mountPathDB},
	}
}

func restrictedSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: &disallowPrivilegeEscalation,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}
