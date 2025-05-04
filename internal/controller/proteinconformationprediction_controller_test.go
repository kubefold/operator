/*
Copyright 2025 Mateusz Woźniak <wozniakmat@student.agh.edu.pl>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datav1 "github.com/kubefold/operator/api/v1"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("ProteinConformationPrediction Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		proteinconformationprediction := &datav1.ProteinConformationPrediction{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind ProteinConformationPrediction")
			err := k8sClient.Get(ctx, typeNamespacedName, proteinconformationprediction)
			if err != nil && errors.IsNotFound(err) {
				resource := &datav1.ProteinConformationPrediction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: datav1.ProteinConformationPredictionSpec{
						Protein: datav1.ProteinConformationPredictionProtein{
							ID:       []string{"test-id"},
							Sequence: "TESTSEQUENCE",
						},
						Database: "test-database",
						Destination: datav1.ProteinConformationPredictionDestination{
							S3: datav1.ProteinConformationPredictionDestinationS3{
								Bucket: "test-bucket",
								Region: "test-region",
							},
						},
						Model: datav1.ProteinConformationPredictionModel{
							Weights: datav1.ProteinConformationPredictionModelWeights{
								HTTP: "http://test-weights",
							},
						},
						Job: datav1.ProteinConformationPredictionJob{
							SearchNodeSelector: v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "test-key",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"test-value"},
											},
										},
									},
								},
							},
							PredictionNodeSelector: v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									{
										MatchExpressions: []v1.NodeSelectorRequirement{
											{
												Key:      "test-key",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"test-value"},
											},
										},
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &datav1.ProteinConformationPrediction{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ProteinConformationPrediction")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ProteinConformationPredictionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
