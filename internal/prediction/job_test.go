package prediction

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	datav1 "github.com/kubefold/operator/api/v1"
)

func makePrediction() *datav1.ProteinConformationPrediction {
	return &datav1.ProteinConformationPrediction{
		Spec: datav1.ProteinConformationPredictionSpec{
			Protein: datav1.ProteinConformationPredictionProtein{
				Sequence: "ABCDE", ID: []string{"id-1"},
			},
			Database: "db",
			Destination: datav1.ProteinConformationPredictionDestination{
				S3: datav1.ProteinConformationPredictionDestinationS3{Bucket: "bkt", Region: "us-east-1"},
			},
			Model: datav1.ProteinConformationPredictionModel{
				Weights: datav1.ProteinConformationPredictionModelWeights{HTTP: "https://example.com/w"},
			},
			Job: datav1.ProteinConformationPredictionJob{
				PredictionNodeSelector: corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{Key: "zone", Operator: corev1.NodeSelectorOpIn, Values: []string{"a", "b"}},
							},
						},
					},
				},
			},
		},
	}
}

func TestBuildPredictWeightsViaEnvVar(t *testing.T) {
	builder := NewJobBuilder(NewNodeSelectorTranslator())
	prediction := makePrediction()

	job := builder.BuildPredict(prediction, "pred-1", "pvc-1", "encoded")
	weightsContainer := findContainer(job.Spec.Template.Spec.InitContainers, "weights-placement")
	if weightsContainer == nil {
		t.Fatal("weights-placement init container missing")
	}

	for _, arg := range weightsContainer.Command {
		if strings.Contains(arg, prediction.Spec.Model.Weights.HTTP) {
			t.Fatalf("URL must not be interpolated into command, found: %q", arg)
		}
	}

	envFound := false
	for _, env := range weightsContainer.Env {
		if env.Name == weightsURLEnvVar {
			if env.Value != prediction.Spec.Model.Weights.HTTP {
				t.Fatalf("WEIGHTS_URL env value mismatch: %q vs %q", env.Value, prediction.Spec.Model.Weights.HTTP)
			}
			envFound = true
		}
	}
	if !envFound {
		t.Fatalf("WEIGHTS_URL env var not found")
	}
}

func TestBuildPredictUsesAffinityNotNodeSelector(t *testing.T) {
	builder := NewJobBuilder(NewNodeSelectorTranslator())
	prediction := makePrediction()
	job := builder.BuildPredict(prediction, "pred-1", "pvc-1", "encoded")

	if job.Spec.Template.Spec.NodeSelector != nil {
		t.Fatal("NodeSelector must remain unset; affinity is used instead")
	}
	if job.Spec.Template.Spec.Affinity == nil {
		t.Fatal("expected Affinity to be populated from PredictionNodeSelector")
	}
	terms := job.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(terms) != 1 || len(terms[0].MatchExpressions[0].Values) != 2 {
		t.Fatalf("multi-value In operator was collapsed: %+v", terms)
	}
}

func TestBuildSearchUsesAffinity(t *testing.T) {
	builder := NewJobBuilder(NewNodeSelectorTranslator())
	prediction := makePrediction()
	prediction.Spec.Job.SearchNodeSelector = corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{
			{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{Key: "gpu", Operator: corev1.NodeSelectorOpExists},
				},
			},
		},
	}
	job := builder.BuildSearch(prediction, "pred-search", "pvc-1", "encoded")
	if job.Spec.Template.Spec.Affinity == nil {
		t.Fatal("expected Affinity to be set")
	}
	terms := job.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if terms[0].MatchExpressions[0].Operator != corev1.NodeSelectorOpExists {
		t.Fatal("Exists operator was dropped during translation")
	}
}

func findContainer(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}
