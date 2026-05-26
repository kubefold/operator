package database

import (
	datav1 "github.com/kubefold/operator/api/v1"
)

const sizeBufferGigabytes = int64(2)

type Sizer interface {
	RequestedGigabytes(database *datav1.ProteinDatabase) int64
}

type sizer struct {
	enumerator DatasetEnumerator
	buffer     int64
}

func NewSizer(enumerator DatasetEnumerator) Sizer {
	return &sizer{enumerator: enumerator, buffer: sizeBufferGigabytes}
}

func (s *sizer) RequestedGigabytes(database *datav1.ProteinDatabase) int64 {
	var totalBytes int64
	for _, dataset := range s.enumerator.FromSpec(database) {
		totalBytes += dataset.Size()
	}
	return totalBytes/1024/1024/1024 + s.buffer
}
