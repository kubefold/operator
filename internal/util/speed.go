package util

import "time"

func CalculateDownloadSpeed(delta int64, duration time.Duration) int64 {
	deltaSeconds := duration.Seconds()
	if deltaSeconds <= 0 {
		return 0
	}
	speedBps := float64(delta) / deltaSeconds
	return int64(speedBps)
}
