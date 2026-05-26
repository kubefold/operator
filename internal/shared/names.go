package shared

import "fmt"

const (
	dataPVCSuffix    = "-data"
	searchJobSuffix  = "-search"
	predictJobSuffix = "-predict"
	uploadJobSuffix  = "-upload"
	downloaderSuffix = "-downloader"
)

func PredictionDataPVCName(predictionName string) string {
	return predictionName + dataPVCSuffix
}

func DatabasePVCName(databaseName string) string {
	return databaseName + dataPVCSuffix
}

func SearchJobName(predictionName string) string {
	return predictionName + searchJobSuffix
}

func PredictJobName(predictionName string) string {
	return predictionName + predictJobSuffix
}

func UploadJobName(predictionName string) string {
	return predictionName + uploadJobSuffix
}

func DownloaderJobName(databaseName, datasetShortName string) string {
	return fmt.Sprintf("%s-%s%s", databaseName, datasetShortName, downloaderSuffix)
}
