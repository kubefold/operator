package controller

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	ProteinDatabaseFinalizer        = "data.kubefold.io/finalizer"
	PersistentVolumeClaimNameSuffix = "-data"
	PersistentVolumeClaimSize       = "1Gi"
	ReconcileInterval               = 10 * time.Second

	DownloaderImage           = "ghcr.io/kubefold/downloader"
	DownloaderImagePullPolicy = corev1.PullAlways

	ManagerImage           = "ghcr.io/kubefold/manager"
	ManagerImagePullPolicy = corev1.PullAlways

	AlphafoldImage           = "public.ecr.aws/m5b1c9c0/alphafold3:latest"
	AlphafoldImagePullPolicy = corev1.PullAlways
)
