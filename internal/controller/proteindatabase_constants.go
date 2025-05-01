package controller

import (
	"time"
)

const (
	ProteinDatabaseFinalizer        = "data.kubefold.io/finalizer"
	PersistentVolumeClaimNameSuffix = "-data"
	PersistentVolumeClaimSize       = "1Gi"
	//DownloaderImage                 = "ghcr.io/kubefold/downloader:latest"
	DownloaderImage           = "downloader"
	DownloaderImagePullPolicy = "Never"
	//DownloaderImagePullPolicy = "Always"
	ReconcileInterval = 10 * time.Second
)
