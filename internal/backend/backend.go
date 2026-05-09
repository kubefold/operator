package backend

import (
	"fmt"
	"strings"

	datav1 "github.com/kubefold/operator/api/v1"
)

const (
	DialectAlphafold2 = "alphafold2"
	DialectAlphafold3 = "alphafold3"
)

type Backend interface {
	Dialect() string
	InputFilename() string
	PrepareInput(prediction *datav1.ProteinConformationPrediction, predictionPhase bool) (string, error)
}

func Resolve(dialect string) (Backend, error) {
	switch dialect {
	case DialectAlphafold2:
		return newAlphafold2Backend(), nil
	case DialectAlphafold3:
		return newAlphafold3Backend(), nil
	default:
		return nil, fmt.Errorf("unsupported backend dialect %q (expected %s or %s)", dialect, DialectAlphafold2, DialectAlphafold3)
	}
}

func DetectDialect(image string) (string, error) {
	normalizedImage := strings.ToLower(image)
	switch {
	case strings.Contains(normalizedImage, DialectAlphafold2):
		return DialectAlphafold2, nil
	case strings.Contains(normalizedImage, DialectAlphafold3):
		return DialectAlphafold3, nil
	default:
		return "", fmt.Errorf("cannot detect dialect from backend image %q (expected substring %s or %s)", image, DialectAlphafold2, DialectAlphafold3)
	}
}
