package controller

import (
	"github.com/kubefold/downloader/pkg/types"
	datav1 "github.com/kubefold/operator/api/v1"
)

func getDatasets(pd *datav1.ProteinDatabase) []types.Dataset {
	o := make([]types.Dataset, 0)
	if pd.Spec.Datasets.MGYClusters {
		o = append(o, types.DatasetMGYClusters)
	}
	if pd.Spec.Datasets.BFD {
		o = append(o, types.DatasetBFD)
	}
	if pd.Spec.Datasets.UniRef90 {
		o = append(o, types.DatasetUniRef90)
	}
	if pd.Spec.Datasets.UniProt {
		o = append(o, types.DatasetUniProt)
	}
	if pd.Spec.Datasets.PDB {
		o = append(o, types.DatasetPDB)
	}
	if pd.Spec.Datasets.PDBSeqReq {
		o = append(o, types.DatasetPDBSeqReq)
	}
	if pd.Spec.Datasets.RNACentral {
		o = append(o, types.DatasetRNACentral)
	}
	if pd.Spec.Datasets.NT {
		o = append(o, types.DatasetNT)
	}
	if pd.Spec.Datasets.RFam {
		o = append(o, types.DatasetRFam)
	}
	return o
}
