package util

import "testing"

const zeroPercent = "0.0%"

func TestFormatPercentageZeroDenominator(t *testing.T) {
	if got := FormatPercentage(100, 0); got != zeroPercent {
		t.Fatalf("FormatPercentage(100, 0) = %q, want %s", got, zeroPercent)
	}
	if got := FormatPercentage(100, -1); got != zeroPercent {
		t.Fatalf("FormatPercentage(100, -1) = %q, want %s", got, zeroPercent)
	}
}

func TestFormatPercentage(t *testing.T) {
	if got := FormatPercentage(50, 100); got != "50.0%" {
		t.Fatalf("FormatPercentage(50, 100) = %q, want 50.0%%", got)
	}
	if got := FormatPercentage(0, 100); got != zeroPercent {
		t.Fatalf("FormatPercentage(0, 100) = %q, want %s", got, zeroPercent)
	}
	if got := FormatPercentage(100, 100); got != "100.0%" {
		t.Fatalf("FormatPercentage(100, 100) = %q, want 100.0%%", got)
	}
}
