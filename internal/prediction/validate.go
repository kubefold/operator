package prediction

import (
	"fmt"

	datav1 "github.com/kubefold/operator/api/v1"
	"github.com/kubefold/operator/internal/shared"
)

const sequencePrefixLength = 10

type SpecValidator interface {
	Validate(prediction *datav1.ProteinConformationPrediction) error
	SequencePrefix(sequence string) string
}

type specValidator struct {
	allowedWeightsHosts []string
}

func NewSpecValidator(allowedWeightsHosts []string) SpecValidator {
	return &specValidator{allowedWeightsHosts: allowedWeightsHosts}
}

func (v *specValidator) Validate(prediction *datav1.ProteinConformationPrediction) error {
	if prediction.Spec.Protein.Sequence == "" {
		return fmt.Errorf("protein sequence cannot be empty")
	}
	if prediction.Spec.Database == "" {
		return fmt.Errorf("database reference cannot be empty")
	}
	if prediction.Spec.Destination.S3.Bucket == "" {
		return fmt.Errorf("destination S3 bucket cannot be empty")
	}
	if prediction.Spec.Destination.S3.Region == "" {
		return fmt.Errorf("destination S3 region cannot be empty")
	}
	if err := shared.ValidateHTTPSURL(
		prediction.Spec.Model.Weights.HTTP,
		shared.WithAllowedHosts(v.allowedWeightsHosts...),
	); err != nil {
		return fmt.Errorf("model weights URL is invalid: %w", err)
	}
	return nil
}

func (v *specValidator) SequencePrefix(sequence string) string {
	if len(sequence) <= sequencePrefixLength {
		return sequence
	}
	return sequence[:sequencePrefixLength] + "..."
}
