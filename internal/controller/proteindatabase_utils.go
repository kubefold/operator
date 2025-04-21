package controller

import datav1 "github.com/kubefold/operator/api/v1"

func getDatasets(pd *datav1.ProteinDatabase) []Dataset {
	o := make([]Dataset, 0)
	if pd.Spec.Datasets.MHYClusters {
		o = append(o, DatasetMHYClusters)
	}
	if pd.Spec.Datasets.BFD {
		o = append(o, DatasetBFD)
	}
	if pd.Spec.Datasets.UniRef90 {
		o = append(o, DatasetUniRef90)
	}
	if pd.Spec.Datasets.PDB {
		o = append(o, DatasetPDB)
	}
	if pd.Spec.Datasets.PDBSeqReq {
		o = append(o, DatasetPDBSeqReq)
	}
	if pd.Spec.Datasets.RNACentral {
		o = append(o, DatasetRNACentral)
	}
	if pd.Spec.Datasets.NT {
		o = append(o, DatasetNT)
	}
	if pd.Spec.Datasets.RFam {
		o = append(o, DatasetRFam)
	}
	return o
}
