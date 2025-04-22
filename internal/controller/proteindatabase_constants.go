package controller

import (
	"time"
)

const (
	ProteinDatabaseFinalizer        = "data.kubefold.io/finalizer"
	PersistentVolumeClaimNameSuffix = "-data"
	PersistentVolumeClaimSize       = "1Gi"
	DownloaderImage                 = "downloader"
	DownloaderImagePullPolicy       = "Never"
	ReconcileInterval               = 10 * time.Second
)

type Dataset string

func (d Dataset) String() string {
	return string(d)
}

func (d Dataset) ShortName() string {
	switch d {
	case DatasetMHYClusters:
		return "mhy-clusters"
	case DatasetBFD:
		return "bfd"
	case DatasetUniRef90:
		return "uniref90"
	case DatasetUniProt:
		return "uniprot"
	case DatasetPDB:
		return "pdb"
	case DatasetPDBSeqReq:
		return "pdb-seqreq"
	case DatasetRNACentral:
		return "rnacentral"
	case DatasetRFam:
		return "rfam"
	case DatasetNT:
		return "nt"
	default:
		return "unknown"
	}
}

const (
	DatasetMHYClusters Dataset = "mgy_clusters_2022_05.fa"
	DatasetBFD         Dataset = "bfd-first_non_consensus_sequences.fasta"
	DatasetUniRef90    Dataset = "uniref90_2022_05.fa"
	DatasetUniProt     Dataset = "uniprot_all_2021_04.fa"
	DatasetPDB         Dataset = "pdb_2022_09_28_mmcif_files.tar"
	DatasetPDBSeqReq   Dataset = "pdb_seqres_2022_09_28.fasta"
	DatasetRNACentral  Dataset = "rnacentral_active_seq_id_90_cov_80_linclust.fasta"
	DatasetNT          Dataset = "nt_rna_2023_02_23_clust_seq_id_90_cov_80_rep_seq.fasta"
	DatasetRFam        Dataset = "rfam_14_9_clust_seq_id_90_cov_80_rep_seq.fasta"
)
