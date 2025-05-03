package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/kubefold/operator/internal/alphafold"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datav1 "github.com/kubefold/operator/api/v1"
)

type ProteinConformationPredictionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteinconformationpredictions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteinconformationpredictions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteinconformationpredictions/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *ProteinConformationPredictionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	pred := &datav1.ProteinConformationPrediction{}
	err := r.Get(ctx, req.NamespacedName, pred)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ProteinConformationPrediction")
		return ctrl.Result{}, err
	}

	if pred.Status.Phase != datav1.ProteinConformationPredictionStatusPhaseFailed &&
		pred.Status.Phase != datav1.ProteinConformationPredictionStatusPhaseCompleted {

		searchJobName := fmt.Sprintf("%s-search", pred.Name)
		searchJob := &batchv1.Job{}
		err := r.Get(ctx, types.NamespacedName{Name: searchJobName, Namespace: pred.Namespace}, searchJob)
		if err == nil {
			if searchJob.Status.Failed > 0 {
				log.Info("Search job failed, updating status", "Job", searchJobName)
				pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
				if err := r.Status().Update(ctx, pred); err != nil {
					log.Error(err, "Failed to update ProteinConformationPrediction status")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
		}

		if pred.Status.Phase == datav1.ProteinConformationPredictionStatusPhasePredicting {
			predJobName := fmt.Sprintf("%s-predict", pred.Name)
			predJob := &batchv1.Job{}
			err := r.Get(ctx, types.NamespacedName{Name: predJobName, Namespace: pred.Namespace}, predJob)
			if err == nil {
				if predJob.Status.Failed > 0 {
					log.Info("Prediction job failed, updating status", "Job", predJobName)
					pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
					if err := r.Status().Update(ctx, pred); err != nil {
						log.Error(err, "Failed to update ProteinConformationPrediction status")
						return ctrl.Result{}, err
					}
					return ctrl.Result{}, nil
				}
			}
		}

		if pred.Status.Phase == datav1.ProteinConformationPredictionStatusPhaseUploadingArtifacts {
			uploadJobName := fmt.Sprintf("%s-upload", pred.Name)
			uploadJob := &batchv1.Job{}
			err := r.Get(ctx, types.NamespacedName{Name: uploadJobName, Namespace: pred.Namespace}, uploadJob)
			if err == nil {
				if uploadJob.Status.Failed > 0 {
					log.Info("Upload job failed, updating status", "Job", uploadJobName)
					pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
					if err := r.Status().Update(ctx, pred); err != nil {
						log.Error(err, "Failed to update ProteinConformationPrediction status")
						return ctrl.Result{}, err
					}
					return ctrl.Result{}, nil
				}
			}
		}
	}

	if pred.Status.Phase == "" {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseNotStarted
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	switch pred.Status.Phase {
	case datav1.ProteinConformationPredictionStatusPhaseNotStarted:
		return r.handleNotStarted(ctx, pred)
	case datav1.ProteinConformationPredictionStatusPhaseAligning:
		return r.handleAligning(ctx, pred)
	case datav1.ProteinConformationPredictionStatusPhasePredicting:
		return r.handlePredicting(ctx, pred)
	case datav1.ProteinConformationPredictionStatusPhaseUploadingArtifacts:
		return r.handleUploadingArtifacts(ctx, pred)
	case datav1.ProteinConformationPredictionStatusPhaseCompleted, datav1.ProteinConformationPredictionStatusPhaseFailed:
		return ctrl.Result{}, nil
	default:
		log.Info("Unknown phase", "Phase", pred.Status.Phase)
		return ctrl.Result{}, nil
	}
}

func (r *ProteinConformationPredictionReconciler) handleNotStarted(ctx context.Context, pred *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	proteinDB := &datav1.ProteinDatabase{}
	err := r.Get(ctx, types.NamespacedName{Name: pred.Spec.Database, Namespace: pred.Namespace}, proteinDB)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Waiting for ProteinDatabase to be created", "Database", pred.Spec.Database)
			return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		}
		log.Error(err, "Failed to get ProteinDatabase")
		return ctrl.Result{}, err
	}

	pvcName := fmt.Sprintf("%s-data", pred.Name)
	pvc := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: pred.Namespace}, pvc)
	if err != nil {
		if errors.IsNotFound(err) {
			pvc = r.newPVC(pred, pvcName)
			if err := controllerutil.SetControllerReference(pred, pvc, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference for PVC")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, pvc); err != nil {
				log.Error(err, "Failed to create PVC")
				return ctrl.Result{}, err
			}
			log.Info("Created PVC", "Name", pvcName)
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get PVC")
		return ctrl.Result{}, err
	}

	jobName := fmt.Sprintf("%s-search", pred.Name)
	job := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: pred.Namespace}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			encodedInput, err := r.prepareFoldInput(pred)
			if err != nil {
				log.Error(err, "Failed to prepare FoldInput")
				return ctrl.Result{}, err
			}

			job = r.newSearchJob(pred, jobName, pvcName, encodedInput)
			if err := controllerutil.SetControllerReference(pred, job, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference for search job")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, job); err != nil {
				log.Error(err, "Failed to create search job")
				return ctrl.Result{}, err
			}

			pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseAligning
			if err := r.Status().Update(ctx, pred); err != nil {
				log.Error(err, "Failed to update ProteinConformationPrediction status")
				return ctrl.Result{}, err
			}

			log.Info("Created search job and updated status", "Name", jobName)
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get search job")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ProteinConformationPredictionReconciler) handleAligning(ctx context.Context, pred *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	jobName := fmt.Sprintf("%s-search", pred.Name)
	job := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: pred.Namespace}, job)
	if err != nil {
		log.Error(err, "Failed to get search job")
		return ctrl.Result{}, err
	}

	if job.Status.Succeeded > 0 {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhasePredicting
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		log.Info("Search job completed, moving to prediction phase")
		return ctrl.Result{Requeue: true}, nil
	}

	if job.Status.Failed > 0 {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		log.Info("Search job failed")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: time.Second * 10}, nil
}

func (r *ProteinConformationPredictionReconciler) handlePredicting(ctx context.Context, pred *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	jobName := fmt.Sprintf("%s-predict", pred.Name)
	job := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: pred.Namespace}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			pvcName := fmt.Sprintf("%s-data", pred.Name)
			job = r.newPredictionJob(pred, jobName, pvcName)
			if err := controllerutil.SetControllerReference(pred, job, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference for prediction job")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, job); err != nil {
				log.Error(err, "Failed to create prediction job")
				return ctrl.Result{}, err
			}
			log.Info("Created prediction job", "Name", jobName)
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get prediction job")
		return ctrl.Result{}, err
	}

	if job.Status.Succeeded > 0 {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseUploadingArtifacts
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		log.Info("Prediction job completed, moving to uploading artifacts phase")
		return ctrl.Result{Requeue: true}, nil
	}

	if job.Status.Failed > 0 {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		log.Info("Prediction job failed")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: time.Second * 10}, nil
}

func (r *ProteinConformationPredictionReconciler) handleUploadingArtifacts(ctx context.Context, pred *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	jobName := fmt.Sprintf("%s-upload", pred.Name)
	job := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: pred.Namespace}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			pvcName := fmt.Sprintf("%s-data", pred.Name)
			job = r.newUploadArtifactsJob(pred, jobName, pvcName)
			if err := controllerutil.SetControllerReference(pred, job, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference for upload artifacts job")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, job); err != nil {
				log.Error(err, "Failed to create upload artifacts job")
				return ctrl.Result{}, err
			}
			log.Info("Created upload artifacts job", "Name", jobName)
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to get upload artifacts job")
		return ctrl.Result{}, err
	}

	if job.Status.Succeeded > 0 {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseCompleted
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}

		pvcName := fmt.Sprintf("%s-data", pred.Name)
		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: pred.Namespace}, pvc)
		if err == nil {
			if err := r.Delete(ctx, pvc); err != nil {
				log.Error(err, "Failed to delete PVC")
				return ctrl.Result{}, err
			}
			log.Info("Deleted PVC", "Name", pvcName)
		}

		log.Info("Upload artifacts job completed, resource is now in completed state")
		return ctrl.Result{}, nil
	}

	if job.Status.Failed > 0 {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		log.Info("Upload artifacts job failed")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: time.Second * 10}, nil
}

func (r *ProteinConformationPredictionReconciler) newPVC(pred *datav1.ProteinConformationPrediction, pvcName string) *corev1.PersistentVolumeClaim {
	storageClass := "fsx-sc"

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pred.Namespace,
			Labels: map[string]string{
				"app":                          pred.Name,
				"data.kubefold.io/prediction":  pred.Name,
				"app.kubernetes.io/name":       "proteinconformationprediction-data",
				"app.kubernetes.io/instance":   pred.Name,
				"app.kubernetes.io/managed-by": "kubefold-operator",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
			StorageClassName: &storageClass,
		},
	}
}

func (r *ProteinConformationPredictionReconciler) prepareFoldInput(pred *datav1.ProteinConformationPrediction) (string, error) {
	input := alphafold.Input{
		Name: fmt.Sprintf("%s-%s", pred.Namespace, pred.Name),
		Sequences: []alphafold.Sequence{
			{
				Protein: alphafold.Protein{
					Sequence:  pred.Spec.Protein.Sequence,
					ID:        pred.Spec.Protein.ID,
					Templates: make([]string, 0),
				},
			},
		},
		ModelSeeds: pred.Spec.Model.Seeds,
		Dialect:    "alphafold3",
		Version:    1,
	}

	inputJson, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fold input: %w", err)
	}

	return base64.StdEncoding.EncodeToString(inputJson), nil
}

func (r *ProteinConformationPredictionReconciler) newSearchJob(pred *datav1.ProteinConformationPrediction, jobName, pvcName, encodedInput string) *batchv1.Job {
	backoffLimit := int32(2)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: pred.Namespace,
			Labels: map[string]string{
				"app":                          pred.Name,
				"data.kubefold.io/prediction":  pred.Name,
				"data.kubefold.io/step":        "search",
				"app.kubernetes.io/name":       "proteinconformationprediction-search",
				"app.kubernetes.io/instance":   pred.Name,
				"app.kubernetes.io/managed-by": "kubefold-operator",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                          pred.Name,
						"data.kubefold.io/prediction":  pred.Name,
						"data.kubefold.io/step":        "search",
						"app.kubernetes.io/name":       "proteinconformationprediction-search",
						"app.kubernetes.io/instance":   pred.Name,
						"app.kubernetes.io/managed-by": "kubefold-operator",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						{
							Name:            "input-placement",
							Image:           ManagerImage,
							ImagePullPolicy: ManagerImagePullPolicy,
							Env: []corev1.EnvVar{
								{
									Name:  "INPUT_PATH",
									Value: "/data/af_input",
								},
								{
									Name:  "OUTPUT_PATH",
									Value: "/data/af_output",
								},
								{
									Name:  "ENCODED_INPUT",
									Value: encodedInput,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "database",
									MountPath: "/public_databases",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "search",
							Image:           AlphafoldImage,
							ImagePullPolicy: AlphafoldImagePullPolicy,
							Command:         []string{"python"},
							Args: []string{
								"run_alphafold.py",
								"--json_path=/data/af_input/fold_input.json",
								"--output_dir=/data/af_output",
								"--model_dir=/data/models",
								"--db_dir=/public_databases",
								"--run_inference=false",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "database",
									MountPath: "/public_databases",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
						{
							Name: "database",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-data", pred.Spec.Database),
								},
							},
						},
					},
				},
			},
		},
	}

	// Add node selector if specified
	if pred.Spec.Job.SearchNodeSelector.NodeSelectorTerms != nil {
		job.Spec.Template.Spec.NodeSelector = map[string]string{}
		for _, term := range pred.Spec.Job.SearchNodeSelector.NodeSelectorTerms {
			for _, exp := range term.MatchExpressions {
				if exp.Operator == corev1.NodeSelectorOpIn && len(exp.Values) > 0 {
					job.Spec.Template.Spec.NodeSelector[exp.Key] = exp.Values[0]
				}
			}
		}
	}

	return job
}

func (r *ProteinConformationPredictionReconciler) newPredictionJob(pred *datav1.ProteinConformationPrediction, jobName, pvcName string) *batchv1.Job {

	backoffLimit := int32(2)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: pred.Namespace,
			Labels: map[string]string{
				"app":                          pred.Name,
				"data.kubefold.io/prediction":  pred.Name,
				"data.kubefold.io/step":        "predict",
				"app.kubernetes.io/name":       "proteinconformationprediction-predict",
				"app.kubernetes.io/instance":   pred.Name,
				"app.kubernetes.io/managed-by": "kubefold-operator",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                          pred.Name,
						"data.kubefold.io/prediction":  pred.Name,
						"data.kubefold.io/step":        "predict",
						"app.kubernetes.io/name":       "proteinconformationprediction-predict",
						"app.kubernetes.io/instance":   pred.Name,
						"app.kubernetes.io/managed-by": "kubefold-operator",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						{
							Name:            "weights-placement",
							Image:           ManagerImage,
							ImagePullPolicy: ManagerImagePullPolicy,
							Command: []string{
								"sh",
							},
							Args: []string{
								"-c",
								fmt.Sprintf("mkdir -p /data/models; wget -O /data/models/af3.bin.zst %s; unzstd /data/models/af3.bin.zst", pred.Spec.Model.Weights.HTTP),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "database",
									MountPath: "/public_databases",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "predict",
							Image:           AlphafoldImage,
							ImagePullPolicy: AlphafoldImagePullPolicy,
							Command:         []string{"python"},
							Args: []string{
								"run_alphafold.py",
								"--json_path=/data/af_input/fold_input.json",
								"--output_dir=/data/af_output",
								"--model_dir=/data/models",
								"--db_dir=/public_databases",
								"--run_data_pipeline=false",
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									"nvidia.com/gpu": resource.MustParse("1"),
								},
								Requests: corev1.ResourceList{
									"nvidia.com/gpu": resource.MustParse("1"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "database",
									MountPath: "/public_databases",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
						{
							Name: "database",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-data", pred.Spec.Database),
								},
							},
						},
					},
				},
			},
		},
	}

	if pred.Spec.Job.PredictionNodeSelector.NodeSelectorTerms != nil {
		job.Spec.Template.Spec.NodeSelector = map[string]string{}
		for _, term := range pred.Spec.Job.PredictionNodeSelector.NodeSelectorTerms {
			for _, exp := range term.MatchExpressions {
				if exp.Operator == corev1.NodeSelectorOpIn && len(exp.Values) > 0 {
					job.Spec.Template.Spec.NodeSelector[exp.Key] = exp.Values[0]
				}
			}
		}
	}

	return job
}

func (r *ProteinConformationPredictionReconciler) newUploadArtifactsJob(pred *datav1.ProteinConformationPrediction, jobName, pvcName string) *batchv1.Job {
	backoffLimit := int32(2)

	var phoneNumbers string
	if len(pred.Spec.Notifications.SMS) > 0 {
		phoneNumbers = pred.Spec.Notifications.SMS[0]
		for i := 1; i < len(pred.Spec.Notifications.SMS); i++ {
			phoneNumbers += "," + pred.Spec.Notifications.SMS[i]
		}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: pred.Namespace,
			Labels: map[string]string{
				"app":                          pred.Name,
				"data.kubefold.io/prediction":  pred.Name,
				"data.kubefold.io/step":        "upload",
				"app.kubernetes.io/name":       "proteinconformationprediction-upload",
				"app.kubernetes.io/instance":   pred.Name,
				"app.kubernetes.io/managed-by": "kubefold-operator",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                          pred.Name,
						"data.kubefold.io/prediction":  pred.Name,
						"data.kubefold.io/step":        "upload",
						"app.kubernetes.io/name":       "proteinconformationprediction-upload",
						"app.kubernetes.io/instance":   pred.Name,
						"app.kubernetes.io/managed-by": "kubefold-operator",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "upload",
							Image:           ManagerImage,
							ImagePullPolicy: ManagerImagePullPolicy,
							Env: []corev1.EnvVar{
								{
									Name:  "INPUT_PATH",
									Value: "/data/af_input",
								},
								{
									Name:  "OUTPUT_PATH",
									Value: "/data/af_output",
								},
								{
									Name:  "BUCKET",
									Value: pred.Spec.Destination.S3.Bucket,
								},
								{
									Name:  "AWS_REGION",
									Value: pred.Spec.Destination.S3.Region,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
							},
						},
						{
							Name:            "notify",
							Image:           ManagerImage,
							ImagePullPolicy: ManagerImagePullPolicy,
							Env: []corev1.EnvVar{
								{
									Name:  "INPUT_PATH",
									Value: "/data/af_input",
								},
								{
									Name:  "OUTPUT_PATH",
									Value: "/data/af_output",
								},
								{
									Name:  "NOTIFICATION_PHONES",
									Value: phoneNumbers,
								},
								{
									Name:  "NOTIFICATION_MESSAGE",
									Value: "Job done",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}

func (r *ProteinConformationPredictionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1.ProteinConformationPrediction{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&batchv1.Job{}).
		Named("proteinconformationprediction").
		Complete(r)
}
