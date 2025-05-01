package util

import (
	"fmt"
	"strings"
)

func FormatSize(size int64, unit string) string {
	if unit == "" {
		unit = "B"
	}

	// For bytes, use a human-readable format
	if strings.ToUpper(unit) == "BYTES" || strings.ToUpper(unit) == "B" {
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

	return fmt.Sprintf("%d %s", size, unit)
}

func FormatPercentage(a, b int64) string {
	return fmt.Sprintf("%.1f%%", float64(a)/float64(b)*100)
}
