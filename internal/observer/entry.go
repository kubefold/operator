package observer

import (
	"encoding/json"
	"time"

	downloaderTypes "github.com/kubefold/downloader/pkg/types"
)

type LogEntry struct {
	DatasetName string    `json:"dataset"`
	Type        string    `json:"type"`
	Msg         string    `json:"msg"`
	Size        int64     `json:"size"`
	Total       int64     `json:"total"`
	Unit        string    `json:"unit"`
	Hash        string    `json:"hash,omitempty"`
	Level       string    `json:"level"`
	Time        time.Time `json:"time,omitempty"`
}

func (e *LogEntry) Dataset() downloaderTypes.Dataset {
	return downloaderTypes.Dataset(e.DatasetName)
}

func ParseLogEntry(line []byte) (LogEntry, bool) {
	var entry LogEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return LogEntry{}, false
	}
	return entry, true
}
