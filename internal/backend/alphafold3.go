package backend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/alphafold"
)

type alphafold3Backend struct{}

func newAlphafold3Backend() Backend {
	return &alphafold3Backend{}
}

func (a alphafold3Backend) Dialect() string {
	return DialectAlphafold3
}

func (a alphafold3Backend) InputFilename() string {
	return "fold_input.json"
}

func (a alphafold3Backend) PrepareInput(prediction *datav1.ProteinConformationPrediction, predictionPhase bool) (string, error) {
	input := alphafold.Input{
		Name: fmt.Sprintf("%s-%s", prediction.Namespace, prediction.Name),
		Sequences: []alphafold.Sequence{
			{
				Protein: alphafold.Protein{
					Sequence: prediction.Spec.Protein.Sequence,
					ID:       prediction.Spec.Protein.ID,
				},
			},
		},
		ModelSeeds: prediction.Spec.Model.Seeds,
		Dialect:    DialectAlphafold3,
		Version:    1,
	}
	if predictionPhase {
		empty := ""
		emptyList := make([]string, 0)
		input.Sequences[0].Protein.Templates = &emptyList
		input.Sequences[0].Protein.UnpairedMSA = &empty
		input.Sequences[0].Protein.PairedMSA = &empty
	}

	inputJson, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fold input: %w", err)
	}

	return base64.StdEncoding.EncodeToString(inputJson), nil
}
