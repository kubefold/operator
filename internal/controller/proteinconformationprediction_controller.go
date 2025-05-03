/*
Copyright 2025 Mateusz Woźniak <wozniakmat@student.agh.edu.pl>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
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

// ProteinConformationPredictionReconciler reconciles a ProteinConformationPrediction object
type ProteinConformationPredictionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteinconformationpredictions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteinconformationpredictions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteinconformationpredictions/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProteinConformationPredictionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the ProteinConformationPrediction instance
	pred := &datav1.ProteinConformationPrediction{}
	err := r.Get(ctx, req.NamespacedName, pred)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		log.Error(err, "Failed to get ProteinConformationPrediction")
		return ctrl.Result{}, err
	}

	// Initialize status if not set
	if pred.Status.Phase == "" {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseNotStarted
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle different phases
	switch pred.Status.Phase {
	case datav1.ProteinConformationPredictionStatusPhaseNotStarted:
		return r.handleNotStarted(ctx, pred)
	case datav1.ProteinConformationPredictionStatusPhaseAligning:
		return r.handleAligning(ctx, pred)
	case datav1.ProteinConformationPredictionStatusPhasePredicting:
		return r.handlePredicting(ctx, pred)
	case datav1.ProteinConformationPredictionStatusPhaseCompleted, datav1.ProteinConformationPredictionStatusPhaseFailed:
		// Nothing to do for these terminal states
		return ctrl.Result{}, nil
	default:
		// Unknown state
		log.Info("Unknown phase", "Phase", pred.Status.Phase)
		return ctrl.Result{}, nil
	}
}

// handleNotStarted handles the NotStarted phase - creates PVC and first stage job
func (r *ProteinConformationPredictionReconciler) handleNotStarted(ctx context.Context, pred *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Create PVC first
	pvcName := fmt.Sprintf("%s-data", pred.Name)
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: pred.Namespace}, pvc)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the PVC
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

	// Check if the search job already exists
	jobName := fmt.Sprintf("%s-search", pred.Name)
	job := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: pred.Namespace}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the search job
			job = r.newSearchJob(pred, jobName, pvcName)
			if err := controllerutil.SetControllerReference(pred, job, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference for search job")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, job); err != nil {
				log.Error(err, "Failed to create search job")
				return ctrl.Result{}, err
			}

			// Update status
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

// handleAligning handles the Aligning phase - waits for first job to complete
func (r *ProteinConformationPredictionReconciler) handleAligning(ctx context.Context, pred *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Check the status of the search job
	jobName := fmt.Sprintf("%s-search", pred.Name)
	job := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: pred.Namespace}, job)
	if err != nil {
		log.Error(err, "Failed to get search job")
		return ctrl.Result{}, err
	}

	// Check if the job has completed
	if job.Status.Succeeded > 0 {
		// Update status to start prediction phase
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhasePredicting
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		log.Info("Search job completed, moving to prediction phase")
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the job has failed
	if job.Status.Failed > 0 {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		log.Info("Search job failed")
		return ctrl.Result{}, nil
	}

	// Job is still running, requeue after short delay
	return ctrl.Result{RequeueAfter: time.Second * 10}, nil
}

// handlePredicting handles the Predicting phase - creates second job and waits for completion
func (r *ProteinConformationPredictionReconciler) handlePredicting(ctx context.Context, pred *datav1.ProteinConformationPrediction) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Check if the prediction job already exists
	jobName := fmt.Sprintf("%s-predict", pred.Name)
	job := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: pred.Namespace}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the prediction job
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

	// Check if the job has completed
	if job.Status.Succeeded > 0 {
		// Update status to completed
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseCompleted
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}

		// Now that we're done, delete the PVC
		pvcName := fmt.Sprintf("%s-data", pred.Name)
		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: pred.Namespace}, pvc)
		if err == nil {
			// PVC exists, delete it
			if err := r.Delete(ctx, pvc); err != nil {
				log.Error(err, "Failed to delete PVC")
				return ctrl.Result{}, err
			}
			log.Info("Deleted PVC", "Name", pvcName)
		}

		log.Info("Prediction job completed, resource is now in completed state")
		return ctrl.Result{}, nil
	}

	// Check if the job has failed
	if job.Status.Failed > 0 {
		pred.Status.Phase = datav1.ProteinConformationPredictionStatusPhaseFailed
		if err := r.Status().Update(ctx, pred); err != nil {
			log.Error(err, "Failed to update ProteinConformationPrediction status")
			return ctrl.Result{}, err
		}
		log.Info("Prediction job failed")
		return ctrl.Result{}, nil
	}

	// Job is still running, requeue after short delay
	return ctrl.Result{RequeueAfter: time.Second * 10}, nil
}

// Helper function to create a new PVC
func (r *ProteinConformationPredictionReconciler) newPVC(pred *datav1.ProteinConformationPrediction, pvcName string) *corev1.PersistentVolumeClaim {
	storageClass := "hostpath"

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pred.Namespace,
			Labels: map[string]string{
				"app": pred.Name,
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

// Helper function to create a new search job
func (r *ProteinConformationPredictionReconciler) newSearchJob(pred *datav1.ProteinConformationPrediction, jobName, pvcName string) *batchv1.Job {
	// Set up environment variables, volumes, and volume mounts
	backoffLimit := int32(2)

	// Setup the job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: pred.Namespace,
			Labels: map[string]string{
				"app": pred.Name,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "search",
							Image:           "sleeper",
							ImagePullPolicy: corev1.PullNever,
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

		// Convert NodeSelector to simple key-value pairs for the pod spec
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

// Helper function to create a new prediction job
func (r *ProteinConformationPredictionReconciler) newPredictionJob(pred *datav1.ProteinConformationPrediction, jobName, pvcName string) *batchv1.Job {
	// Set up environment variables, volumes, and volume mounts
	backoffLimit := int32(2)

	// Setup the job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: pred.Namespace,
			Labels: map[string]string{
				"app": pred.Name,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "predict",
							Image:           "sleeper",
							ImagePullPolicy: corev1.PullNever,
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
	if pred.Spec.Job.PredictionNodeSelector.NodeSelectorTerms != nil {
		job.Spec.Template.Spec.NodeSelector = map[string]string{}

		// Convert NodeSelector to simple key-value pairs for the pod spec
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

// SetupWithManager sets up the controller with the Manager.
func (r *ProteinConformationPredictionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1.ProteinConformationPrediction{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&batchv1.Job{}).
		Named("proteinconformationprediction").
		Complete(r)
}
