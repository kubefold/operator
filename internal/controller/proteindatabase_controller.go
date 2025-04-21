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
	"time"

	datav1 "github.com/kubefold/operator/api/v1"
)

const (
	ProteinDatabaseFinalizer        = "data.kubefold.io/finalizer"
	PersistentVolumeClaimNameSuffix = "-data"
	PersistentVolumeClaimSize       = "1Gi"
)

// ProteinDatabaseReconciler reconciles a ProteinDatabase object
type ProteinDatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteindatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteindatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=data.kubefold.io,resources=proteindatabases/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ProteinDatabase object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *ProteinDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling ProteinDatabase", "name", req.Name, "namespace", req.Namespace)

	pd := &datav1.ProteinDatabase{}
	if err := r.Get(ctx, req.NamespacedName, pd); err != nil {
		if errors.IsNotFound(err) {
			log.Info("ProteinDatabase resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ProteinDatabase")
		return ctrl.Result{}, err
	}

	if !pd.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(pd, ProteinDatabaseFinalizer) {
			if err := r.cleanupResources(ctx, pd); err != nil {
				log.Error(err, "Failed to clean up resources")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(pd, ProteinDatabaseFinalizer)
			if err := r.Update(ctx, pd); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(pd, ProteinDatabaseFinalizer) {
		controllerutil.AddFinalizer(pd, ProteinDatabaseFinalizer)
		if err := r.Update(ctx, pd); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	pvcName := pd.Name + PersistentVolumeClaimNameSuffix
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: pd.Namespace}, pvc)

	if err != nil && errors.IsNotFound(err) {
		pvc, err = r.createPersistentVolumeClaim(ctx, pd)
		if err != nil {
			log.Error(err, "Failed to create PVC")
			return ctrl.Result{}, err
		}
		log.Info("Created new PVC", "pvcName", pvc.Name)
	} else if err != nil {
		log.Error(err, "Failed to get PVC")
		return ctrl.Result{}, err
	}

	if pvc.Status.Phase == corev1.ClaimBound {
		if pd.Status.VolumeName != pvc.Spec.VolumeName {
			log.Info("PVC is bound, updating ProteinDatabase status", "pvcName", pvc.Name, "volumeName", pvc.Spec.VolumeName)
			if err := r.updateProteinDatabaseStatus(ctx, pd, pvc); err != nil {
				log.Error(err, "Failed to update ProteinDatabase status")
				return ctrl.Result{}, err
			}
		}
	} else {
		log.Info("PVC is not bound yet", "pvcName", pvc.Name, "phase", pvc.Status.Phase)
		return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
	}

	log.Info("Reconciliation completed successfully")
	return ctrl.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
}

func (r *ProteinDatabaseReconciler) createPersistentVolumeClaim(ctx context.Context, pd *datav1.ProteinDatabase) (*corev1.PersistentVolumeClaim, error) {
	pvcName := pd.Name + PersistentVolumeClaimNameSuffix

	labels := pd.Spec.Volume.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.kubernetes.io/name"] = "proteindatabase"
	labels["app.kubernetes.io/instance"] = pd.Name
	labels["app.kubernetes.io/managed-by"] = "kubefold-operator"

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pd.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			StorageClassName: pd.Spec.Volume.StorageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(PersistentVolumeClaimSize),
				},
			},
		},
	}

	if pd.Spec.Volume.Selector != nil {
		pvc.Spec.Selector = pd.Spec.Volume.Selector
	}

	if err := controllerutil.SetControllerReference(pd, pvc, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, pvc); err != nil {
		return nil, err
	}

	return pvc, nil
}

func (r *ProteinDatabaseReconciler) cleanupResources(ctx context.Context, pd *datav1.ProteinDatabase) error {
	return nil
}

func (r *ProteinDatabaseReconciler) updateProteinDatabaseStatus(ctx context.Context, pd *datav1.ProteinDatabase, pvc *corev1.PersistentVolumeClaim) error {
	pdCopy := pd.DeepCopy()

	// Update status fields
	pdCopy.Status.VolumeName = pvc.Spec.VolumeName

	// Create a patch from the original and updated objects
	return r.Status().Update(ctx, pdCopy)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProteinDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&datav1.ProteinDatabase{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Named("proteindatabase").
		Complete(r)
}
