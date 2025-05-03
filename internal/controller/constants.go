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
	DownloaderImage           = "downloader"
	DownloaderImagePullPolicy = corev1.PullNever
	//DownloaderImagePullPolicy = "Always"
	ReconcileInterval = 10 * time.Second

	ManagerImage           = "manager"
	ManagerImagePullPolicy = corev1.PullNever
)
