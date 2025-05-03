package controller

import (
	corev1 "k8s.io/api/core/v1"
	"time"
)

const (
	ProteinDatabaseFinalizer        = "data.kubefold.io/finalizer"
	PersistentVolumeClaimNameSuffix = "-data"
	PersistentVolumeClaimSize       = "1Gi"
	//DownloaderImage                 = "ghcr.io/kubefold/downloader:v0.0.9"
	DownloaderImage           = "ghcr.io/kubefold/downloader"
	DownloaderImagePullPolicy = corev1.PullAlways
	//DownloaderImagePullPolicy = "Always"
	ReconcileInterval = 10 * time.Second

	ManagerImage           = "ghcr.io/kubefold/manager"
	ManagerImagePullPolicy = corev1.PullAlways

	AlphafoldImage           = "public.ecr.aws/m5b1c9c0/alphafold3:latest"
	AlphafoldImagePullPolicy = corev1.PullAlways
)
