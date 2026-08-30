/*
Copyright 2026.

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

package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appsv1alpha1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1alpha1"
	appsv1beta1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1beta1"
	"github.com/Laaaaksh/k8s-operators/code/webapp-operator/internal/registry"
)

const (
	webAppName    = "site"
	testImage     = "nginx:1.27"
	testNamespace = "default"
)

var _ = Describe("WebApp Webhook (Hub version)", func() {
	var (
		obj       *appsv1beta1.WebApp
		oldObj    *appsv1beta1.WebApp
		validator WebAppCustomValidator
		defaulter WebAppCustomDefaulter
	)

	BeforeEach(func() {
		obj = &appsv1beta1.WebApp{
			ObjectMeta: metav1.ObjectMeta{Name: webAppName, Namespace: testNamespace},
			Spec: appsv1beta1.WebAppSpec{
				Image: testImage, Port: 8080,
				Scaling: appsv1beta1.ScalingSpec{MinReplicas: 1, MaxReplicas: 3},
			},
		}
		oldObj = obj.DeepCopy()
		validator = WebAppCustomValidator{}
		defaulter = WebAppCustomDefaulter{}
	})

	Context("Defaulting", func() {
		It("derives hostName from name and namespace when unset", func() {
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(obj.Spec.HostName).To(Equal("site.default.apps.example.com"))
		})
	})

	Context("Validating", func() {
		It("rejects minReplicas greater than maxReplicas", func() {
			obj.Spec.Scaling = appsv1beta1.ScalingSpec{MinReplicas: 5, MaxReplicas: 2}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be <= maxReplicas"))
		})

		It("admits minReplicas equal to maxReplicas", func() {
			obj.Spec.Scaling = appsv1beta1.ScalingSpec{MinReplicas: 2, MaxReplicas: 2}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects a WebApp created in the reserved registry namespace", func() {
			obj.Namespace = registry.Namespace
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("reserved for the WebApp registry"))
		})

		It("rejects changing hostName once set", func() {
			oldObj.Spec.HostName = "old.example.com"
			obj.Spec.HostName = "new.example.com"
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})
	})

	// These exercise api/v1alpha1/webapp_conversion.go and
	// api/v1beta1/webapp_conversion.go directly - the ConvertTo/ConvertFrom
	// methods the API server calls, not this package's webhook handlers.
	// Worth having both: the webhook tests above prove admission logic;
	// these prove the Hub/spoke conversion those requests get funneled
	// through before or after admission runs, for whichever version wasn't
	// submitted.
	Context("Conversion between v1alpha1 and v1beta1", func() {
		It("round-trips a WebApp with equal min/max losslessly", func() {
			original := &appsv1alpha1.WebApp{
				ObjectMeta: metav1.ObjectMeta{Name: webAppName, Namespace: testNamespace},
				Spec: appsv1alpha1.WebAppSpec{
					Image: testImage, Port: 8080, Replicas: 2, HostName: "site.example.com",
				},
			}

			hub := &appsv1beta1.WebApp{}
			Expect(original.ConvertTo(hub)).To(Succeed())
			Expect(hub.Spec.Scaling.MinReplicas).To(Equal(int32(2)))
			Expect(hub.Spec.Scaling.MaxReplicas).To(Equal(int32(2)))
			Expect(hub.Spec.Image).To(Equal(testImage))

			roundTripped := &appsv1alpha1.WebApp{}
			Expect(roundTripped.ConvertFrom(hub)).To(Succeed())
			Expect(roundTripped.Spec).To(Equal(original.Spec))
		})

		It("documents the lossy direction: a widened v1beta1 range collapses to MinReplicas on the way down", func() {
			hub := &appsv1beta1.WebApp{
				ObjectMeta: metav1.ObjectMeta{Name: webAppName, Namespace: testNamespace},
				Spec: appsv1beta1.WebAppSpec{
					Image: testImage, Port: 8080,
					Scaling: appsv1beta1.ScalingSpec{MinReplicas: 2, MaxReplicas: 10},
				},
			}

			spoke := &appsv1alpha1.WebApp{}
			Expect(spoke.ConvertFrom(hub)).To(Succeed())
			Expect(spoke.Spec.Replicas).To(Equal(int32(2)), "MaxReplicas has nowhere to go in v1alpha1")

			// Converting back up does NOT recover the original range - this
			// is the round-trip data loss api/v1alpha1/webapp_conversion.go
			// documents. Asserting it here, not just in a comment, is what
			// makes it a claim this repo has actually checked rather than
			// one it's only telling you about.
			back := &appsv1beta1.WebApp{}
			Expect(spoke.ConvertTo(back)).To(Succeed())
			Expect(back.Spec.Scaling.MaxReplicas).To(Equal(int32(2)), "not the original 10")
		})

		// Everything above calls ConvertTo/ConvertFrom directly - real Go
		// function calls, no HTTP involved. This test is the one that
		// actually proves the wiring: it writes a WebApp as v1alpha1
		// through k8sClient (started against the real envtest API server
		// in this package's BeforeSuite, with the manager's conversion
		// webhook registered and serving), then reads the SAME object back
		// as v1beta1. If that round trip produces the expected Scaling
		// values, the API server genuinely called this service's /convert
		// endpoint over HTTPS mid-request - not a mock, the actual
		// admission-time behavior a real cluster would exhibit.
		It("converts through the real API server via the live conversion webhook", func() {
			created := &appsv1alpha1.WebApp{
				ObjectMeta: metav1.ObjectMeta{Name: "live-convert", Namespace: testNamespace},
				Spec: appsv1alpha1.WebAppSpec{
					Image: testImage, Port: 8080, Replicas: 4, HostName: "live-convert.example.com",
				},
			}
			Expect(k8sClient.Create(ctx, created)).To(Succeed())

			var asHub appsv1beta1.WebApp
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "live-convert", Namespace: testNamespace}, &asHub)).To(Succeed())
			Expect(asHub.Spec.Scaling.MinReplicas).To(Equal(int32(4)))
			Expect(asHub.Spec.Scaling.MaxReplicas).To(Equal(int32(4)))
			Expect(asHub.Spec.Image).To(Equal(testImage))
		})
	})
})
