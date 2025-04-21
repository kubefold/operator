package controller

import (
	"time"
)

const (
	ProteinDatabaseFinalizer        = "data.kubefold.io/finalizer"
	PersistentVolumeClaimNameSuffix = "-data"
	PersistentVolumeClaimSize       = "1Gi"
	DownloaderImage                 = "downloader"
	ReconcileInterval               = 10 * time.Second
)

var datasets = []string{"datasetA", "datasetB", "datasetC", "datasetD"}
