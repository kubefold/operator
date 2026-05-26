package shared

import "maps"

const (
	LabelApp          = "app"
	LabelDatabase     = "data.kubefold.io/database"
	LabelPrediction   = "data.kubefold.io/prediction"
	LabelDataset      = "data.kubefold.io/dataset"
	LabelStep         = "data.kubefold.io/step"
	LabelAppName      = "app.kubernetes.io/name"
	LabelAppInstance  = "app.kubernetes.io/instance"
	LabelAppManagedBy = "app.kubernetes.io/managed-by"
	ManagedByOperator = "kubefold-operator"
)

func DatabaseLabels(databaseName, appName string) map[string]string {
	return map[string]string{
		LabelDatabase:     databaseName,
		LabelAppName:      appName,
		LabelAppInstance:  databaseName,
		LabelAppManagedBy: ManagedByOperator,
	}
}

func DownloaderJobLabels(databaseName, datasetShortName string) map[string]string {
	labels := DatabaseLabels(databaseName, "proteindatabase-downloader")
	labels[LabelDataset] = datasetShortName
	return labels
}

func PredictionLabels(predictionName, appName, step string) map[string]string {
	return map[string]string{
		LabelApp:          predictionName,
		LabelPrediction:   predictionName,
		LabelStep:         step,
		LabelAppName:      appName,
		LabelAppInstance:  predictionName,
		LabelAppManagedBy: ManagedByOperator,
	}
}

func PredictionPVCLabels(predictionName string) map[string]string {
	return map[string]string{
		LabelApp:          predictionName,
		LabelPrediction:   predictionName,
		LabelAppName:      "proteinconformationprediction-data",
		LabelAppInstance:  predictionName,
		LabelAppManagedBy: ManagedByOperator,
	}
}

func MergeLabels(base, overrides map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(overrides))
	maps.Copy(merged, base)
	maps.Copy(merged, overrides)
	return merged
}
