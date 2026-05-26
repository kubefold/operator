package prediction

import (
	"strings"
	"testing"

	datav1 "github.com/kubefold/operator/api/v1"
)

func TestSequencePrefix(t *testing.T) {
	v := NewSpecValidator(nil)

	cases := []struct {
		name     string
		sequence string
		want     string
	}{
		{"empty", "", ""},
		{"short under threshold", "ABCD", "ABCD"},
		{"exactly threshold", "ABCDEFGHIJ", "ABCDEFGHIJ"},
		{"over threshold", "ABCDEFGHIJKLMNO", "ABCDEFGHIJ..."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := v.SequencePrefix(tc.sequence)
			if got != tc.want {
				t.Fatalf("SequencePrefix(%q) = %q, want %q", tc.sequence, got, tc.want)
			}
		})
	}
}

func TestValidateSpec(t *testing.T) {
	makeValid := func() *datav1.ProteinConformationPrediction {
		return &datav1.ProteinConformationPrediction{
			Spec: datav1.ProteinConformationPredictionSpec{
				Protein: datav1.ProteinConformationPredictionProtein{
					Sequence: "ABCDEFG",
					ID:       []string{"id-1"},
				},
				Database: "db-1",
				Destination: datav1.ProteinConformationPredictionDestination{
					S3: datav1.ProteinConformationPredictionDestinationS3{
						Bucket: "bucket",
						Region: "us-east-1",
					},
				},
				Model: datav1.ProteinConformationPredictionModel{
					Weights: datav1.ProteinConformationPredictionModelWeights{
						HTTP: "https://example.com/weights.zst",
					},
				},
			},
		}
	}

	cases := []struct {
		name      string
		mutate    func(*datav1.ProteinConformationPrediction)
		wantErr   bool
		errSubstr string
	}{
		{"valid", func(p *datav1.ProteinConformationPrediction) {}, false, ""},
		{"empty sequence", func(p *datav1.ProteinConformationPrediction) { p.Spec.Protein.Sequence = "" }, true, "sequence"},
		{"empty database", func(p *datav1.ProteinConformationPrediction) { p.Spec.Database = "" }, true, "database"},
		{"empty bucket", func(p *datav1.ProteinConformationPrediction) { p.Spec.Destination.S3.Bucket = "" }, true, "bucket"},
		{"empty region", func(p *datav1.ProteinConformationPrediction) { p.Spec.Destination.S3.Region = "" }, true, "region"},
		{"http scheme", func(p *datav1.ProteinConformationPrediction) { p.Spec.Model.Weights.HTTP = "http://example.com/x" }, true, "URL"},
		{"shell injection in url", func(p *datav1.ProteinConformationPrediction) {
			p.Spec.Model.Weights.HTTP = "https://example.com/x;rm -rf /"
		}, true, "URL"},
		{"empty weights url", func(p *datav1.ProteinConformationPrediction) { p.Spec.Model.Weights.HTTP = "" }, true, "URL"},
	}

	v := NewSpecValidator(nil)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prediction := makeValid()
			tc.mutate(prediction)
			err := v.Validate(prediction)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
