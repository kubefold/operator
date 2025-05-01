package util

import (
	"fmt"
)

func FormatSize(size int64) string {
	if size < 1000 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1000*1000 {
		return fmt.Sprintf("%.2f KB", float64(size)/1000)
	} else if size < 1000*1000*1000 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1000*1000))
	} else {
		return fmt.Sprintf("%.2f GB", float64(size)/(1000*1000*1000))
	}
}

func FormatSpeed(speed int64) string {
	switch {
	case speed < 1_000:
		return fmt.Sprintf("%d B/s", speed)
	case speed < 1_000_000:
		return fmt.Sprintf("%.2f KB/s", float64(speed)/1_000)
	case speed < 1_000_000_000:
		return fmt.Sprintf("%.2f MB/s", float64(speed)/1_000_000)
	default:
		return fmt.Sprintf("%.2f GB/s", float64(speed)/1_000_000_000)
	}
}

func FormatPercentage(a, b int64) string {
	return fmt.Sprintf("%.1f%%", float64(a)/float64(b)*100)
}
