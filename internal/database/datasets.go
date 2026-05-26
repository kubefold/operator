package database

import (
	downloaderTypes "github.com/kubefold/downloader/pkg/types"

	datav1 "github.com/kubefold/operator/api/v1"
)

type DatasetEnumerator interface {
	FromSpec(database *datav1.ProteinDatabase) []downloaderTypes.Dataset
	All() []downloaderTypes.Dataset
}

type datasetEnumerator struct{}

func NewDatasetEnumerator() DatasetEnumerator {
	return &datasetEnumerator{}
}

type datasetEntry struct {
	dataset downloaderTypes.Dataset
	enabled func(*datav1.ProteinDatabase) bool
}

var datasetEntries = []datasetEntry{
	{downloaderTypes.DatasetMGYClusters, func(p *datav1.ProteinDatabase) bool { return p.Spec.Datasets.MGYClusters }},
	{downloaderTypes.DatasetBFD, func(p *datav1.ProteinDatabase) bool { return p.Spec.Datasets.BFD }},
	{downloaderTypes.DatasetUniRef90, func(p *datav1.ProteinDatabase) bool { return p.Spec.Datasets.UniRef90 }},
	{downloaderTypes.DatasetUniProt, func(p *datav1.ProteinDatabase) bool { return p.Spec.Datasets.UniProt }},
	{downloaderTypes.DatasetPDB, func(p *datav1.ProteinDatabase) bool { return p.Spec.Datasets.PDB }},
	{downloaderTypes.DatasetPDBSeqReq, func(p *datav1.ProteinDatabase) bool { return p.Spec.Datasets.PDBSeqReq }},
	{downloaderTypes.DatasetRNACentral, func(p *datav1.ProteinDatabase) bool { return p.Spec.Datasets.RNACentral }},
	{downloaderTypes.DatasetNT, func(p *datav1.ProteinDatabase) bool { return p.Spec.Datasets.NT }},
	{downloaderTypes.DatasetRFam, func(p *datav1.ProteinDatabase) bool { return p.Spec.Datasets.RFam }},
}

func (e *datasetEnumerator) FromSpec(database *datav1.ProteinDatabase) []downloaderTypes.Dataset {
	out := make([]downloaderTypes.Dataset, 0, len(datasetEntries))
	for _, entry := range datasetEntries {
		if entry.enabled(database) {
			out = append(out, entry.dataset)
		}
	}
	return out
}

func (e *datasetEnumerator) All() []downloaderTypes.Dataset {
	out := make([]downloaderTypes.Dataset, 0, len(datasetEntries))
	for _, entry := range datasetEntries {
		out = append(out, entry.dataset)
	}
	return out
}
