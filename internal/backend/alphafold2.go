package backend

import (
	"encoding/base64"
	"strings"

	datav1 "github.com/kubefold/operator/api/v1"
)

type alphafold2Backend struct{}

func newAlphafold2Backend() Backend {
	return &alphafold2Backend{}
}

func (a alphafold2Backend) Dialect() string {
	return DialectAlphafold2
}

func (a alphafold2Backend) InputFilename() string {
	return "fold_input.fasta"
}

func (a alphafold2Backend) PrepareInput(prediction *datav1.ProteinConformationPrediction, _ bool) (string, error) {
	chainIDs := prediction.Spec.Protein.ID
	if len(chainIDs) == 0 {
		chainIDs = []string{"A"}
	}
	var builder strings.Builder
	for _, chainID := range chainIDs {
		builder.WriteString(">chain_")
		builder.WriteString(chainID)
		builder.WriteString("\n")
		builder.WriteString(prediction.Spec.Protein.Sequence)
		builder.WriteString("\n")
	}
	return base64.StdEncoding.EncodeToString([]byte(builder.String())), nil
}
