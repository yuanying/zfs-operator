/*
Copyright 2020 O.Yuanying <yuanying@fraction.jp>

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

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/mistifyio/go-zfs/v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zfsv1alpha1 "github.com/yuanying/zfs-operator/api/v1alpha1"
)

const (
	VolumeCleanupFinalizer = "volume.zfs.unstable.cloud/cleanup"
)

// VolumeReconciler reconciles a Volume object
type VolumeReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	NodeName string
}

//+kubebuilder:rbac:groups=zfs.unstable.cloud,resources=volumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=zfs.unstable.cloud,resources=volumes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=zfs.unstable.cloud,resources=volumes/finalizers,verbs=update

func (r *VolumeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	vol := &zfsv1alpha1.Volume{}
	if err := r.Get(ctx, req.NamespacedName, vol); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Unable to fetch Volume - skipping")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Unable to fetch Volume")
		return ctrl.Result{}, err
	}
	if vol.Spec.NodeName != r.NodeName {
		log.Info("NodeName is different -- skipping", "VolumeNodeName", vol.Spec.NodeName, "NodeName", r.NodeName)
		return ctrl.Result{}, nil
	}

	log = log.WithValues("volName", vol.Spec.VolumeName)

	if !vol.DeletionTimestamp.IsZero() {
		if err := r.deleteVolume(log, vol); err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(vol, VolumeCleanupFinalizer)
		if err := r.Update(ctx, vol); err != nil {
			log.Error(err, "Failed to update Volume")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	t := metav1.Now()
	if err := r.reconcileVolume(log, vol, t); err != nil {
		r.Status().Update(ctx, vol)
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, vol); err != nil {
		log.Error(err, "Failed to update volume status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VolumeReconciler) reconcileVolume(log logr.Logger, vol *zfsv1alpha1.Volume, t metav1.Time) error {
	if !containsFinalizer(vol, VolumeCleanupFinalizer) {
		controllerutil.AddFinalizer(vol, VolumeCleanupFinalizer)
		if err := r.Update(context.Background(), vol); err != nil {
			log.Error(err, "Unable to update")
			return err
		}
	}

	found, err := r.findDataset(log, vol)
	if err != nil {
		return err
	}
	if found != nil && vol.Spec.Properties != nil {
		for k, v := range vol.Spec.Properties {
			if err := found.SetProperty(k, v); err != nil {
				msg := fmt.Sprintf("Failed to update property[%v] to %v", k, v)
				log.Error(err, msg)
				vol.SetConditionReason(zfsv1alpha1.VolumeConditionReady, corev1.ConditionFalse, "FailedUpdateProperty", msg, t)
				return err
			}
		}
	} else {
		// Create Dataset
		volSize := vol.Spec.Capacity[corev1.ResourceStorage]
		volSizeBytes := volSize.Value()
		_, err = zfs.CreateVolume(vol.Spec.VolumeName, uint64(volSizeBytes), vol.Spec.Properties)
		if err != nil {
			msg := "Failed to create zfs volume"
			log.Error(err, msg)
			vol.SetConditionReason(zfsv1alpha1.VolumeConditionReady, corev1.ConditionFalse, "FailedGetDataset", msg, t)
			return err
		}

	}
	vol.SetCondition(zfsv1alpha1.VolumeConditionReady, corev1.ConditionTrue, t)
	return nil
}

// To handle "dataset does not exist" error, findDataset try to find dataset from parent dataset name
func (r *VolumeReconciler) findDataset(log logr.Logger, vol *zfsv1alpha1.Volume) (*zfs.Dataset, error) {
	names := strings.Split(vol.Spec.VolumeName, "/")
	prefix := strings.Join(names[:len(names)-1], "/")
	// This assumes all zfs volume belong to parent dataset
	// In otherwords, this reconciler can't create root dataset
	dss, err := zfs.Volumes(prefix)
	if err != nil {
		msg := "Failed to get zfs dataset info"
		log.Error(err, msg)
		return nil, err
	}
	for _, ds := range dss {
		if ds.Name == vol.Spec.VolumeName {
			log.Info("Dataset found")
			return ds, nil
		}
	}
	log.Info("Dataset not found")
	return nil, nil
}

func (r *VolumeReconciler) deleteVolume(log logr.Logger, vol *zfsv1alpha1.Volume) error {
	found, err := r.findDataset(log, vol)
	if err != nil {
		return err
	}
	if found != nil {
		log.Info("Try to destroy zfs volume")
		if err = found.Destroy(zfs.DestroyDefault); err != nil {
			log.Error(err, "Failed to destroy dataset")
			return err
		}
	}

	return nil
}

func containsFinalizer(vol *zfsv1alpha1.Volume, finalizer string) bool {
	f := vol.GetFinalizers()
	for _, e := range f {
		if e == finalizer {
			return true
		}
	}
	return false
}

func (r *VolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zfsv1alpha1.Volume{}).
		Complete(r)
}
