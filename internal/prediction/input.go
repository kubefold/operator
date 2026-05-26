package prediction

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/alphafold"
)

type InputEncoder interface {
	Encode(prediction *datav1.ProteinConformationPrediction, forPredictionPhase bool) (string, error)
}

type inputEncoder struct{}

func NewInputEncoder() InputEncoder {
	return &inputEncoder{}
}

func (e *inputEncoder) Encode(prediction *datav1.ProteinConformationPrediction, forPredictionPhase bool) (string, error) {
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
		Dialect:    "alphafold3",
		Version:    1,
	}
	if forPredictionPhase {
		empty := ""
		emptyList := make([]string, 0)
		input.Sequences[0].Protein.Templates = &emptyList
		input.Sequences[0].Protein.UnpairedMSA = &empty
		input.Sequences[0].Protein.PairedMSA = &empty
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal fold input: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encoded), nil
}
