package util

import "time"

func CalculateDownloadSpeed(previousSize, currentSize int64, previousTime, currentTime time.Time) int64 {
	deltaBytes := currentSize - previousSize
	deltaSeconds := currentTime.Sub(previousTime).Seconds()
	if deltaSeconds <= 0 {
		return 0
	}
	speedBps := float64(deltaBytes) / deltaSeconds
	return int64(speedBps)
}
