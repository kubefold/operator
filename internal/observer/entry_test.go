package observer

import (
	"testing"
)

func TestParseLogEntryValid(t *testing.T) {
	line := []byte(`{"dataset":"bfd","type":"progress","size":1024,"total":2048,"unit":"B","level":"info"}`)
	entry, ok := ParseLogEntry(line)
	if !ok {
		t.Fatal("expected ok=true for valid JSON")
	}
	if entry.DatasetName != "bfd" {
		t.Fatalf("DatasetName = %q, want bfd", entry.DatasetName)
	}
	if entry.Size != 1024 || entry.Total != 2048 {
		t.Fatalf("size/total mismatch: %d/%d", entry.Size, entry.Total)
	}
}

func TestParseLogEntryInvalid(t *testing.T) {
	if _, ok := ParseLogEntry([]byte("not json")); ok {
		t.Fatal("expected ok=false for invalid JSON")
	}
	if _, ok := ParseLogEntry([]byte("")); ok {
		t.Fatal("expected ok=false for empty input")
	}
}
