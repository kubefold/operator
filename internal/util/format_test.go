package util

import "testing"

func TestFormatPercentageZeroDenominator(t *testing.T) {
	if got := FormatPercentage(100, 0); got != "0.0%" {
		t.Fatalf("FormatPercentage(100, 0) = %q, want 0.0%%", got)
	}
	if got := FormatPercentage(100, -1); got != "0.0%" {
		t.Fatalf("FormatPercentage(100, -1) = %q, want 0.0%%", got)
	}
}

func TestFormatPercentage(t *testing.T) {
	if got := FormatPercentage(50, 100); got != "50.0%" {
		t.Fatalf("FormatPercentage(50, 100) = %q, want 50.0%%", got)
	}
	if got := FormatPercentage(0, 100); got != "0.0%" {
		t.Fatalf("FormatPercentage(0, 100) = %q, want 0.0%%", got)
	}
	if got := FormatPercentage(100, 100); got != "100.0%" {
		t.Fatalf("FormatPercentage(100, 100) = %q, want 100.0%%", got)
	}
}
