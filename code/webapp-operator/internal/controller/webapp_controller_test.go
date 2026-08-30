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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"

	appsv1beta1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1beta1"
	"github.com/Laaaaksh/k8s-operators/code/webapp-operator/internal/registry"
)

const (
	webAppName = "site"
	testImage  = "nginx:1.27"
)

// These tests exercise the real controller running against the real
// envtest API server (started once in suite_test.go's BeforeSuite) -
// nothing here calls Reconcile directly. That's the point: it's the only
// way to actually prove SetupWithManager's Owns() watches work, not just
// that Reconcile's logic is correct in isolation.
//
// envtest runs a real kube-apiserver and etcd, but no kubelet - Pods a
// Deployment creates never actually start, so a Deployment's
// status.readyReplicas never advances on its own. Tests that need "the
// Deployment is ready" simulate what a kubelet normally does: patch the
// Deployment's status subresource directly. That is real Kubernetes
// behavior (status is just another subresource you can write), not a
// mock - see curriculum/04-envtest.md for why this is the honest
// boundary of what envtest can and can't stand in for.
var _ = Describe("WebApp Controller", func() {
	ctx := context.Background()
	var ns string

	BeforeEach(func() {
		ns = "wt-" + utilrand.String(8)
		Expect(k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: ns},
		})).To(Succeed())
	})

	It("creates a Deployment and Service matching Spec, and reports Available once the Deployment is ready", func() {
		webapp := &appsv1beta1.WebApp{
			ObjectMeta: metav1.ObjectMeta{Name: webAppName, Namespace: ns},
			Spec: appsv1beta1.WebAppSpec{
				Image:   testImage,
				Port:    8080,
				Scaling: appsv1beta1.ScalingSpec{MinReplicas: 2, MaxReplicas: 2},
			},
		}
		Expect(k8sClient.Create(ctx, webapp)).To(Succeed())

		key := types.NamespacedName{Name: webAppName, Namespace: ns}
		var dep appsv1.Deployment
		Eventually(func() error {
			return k8sClient.Get(ctx, key, &dep)
		}).Should(Succeed())
		Expect(*dep.Spec.Replicas).To(Equal(int32(2)))
		Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal(testImage))

		var svc corev1.Service
		Eventually(func() error {
			return k8sClient.Get(ctx, key, &svc)
		}).Should(Succeed())
		Expect(svc.Spec.Ports[0].Port).To(Equal(int32(8080)))

		By("WebApp reports Progressing while the Deployment has no ready replicas")
		Eventually(func() string {
			_ = k8sClient.Get(ctx, key, webapp)
			return string(webapp.Status.Phase)
		}).Should(Equal(string(appsv1beta1.WebAppPhaseProgressing)))

		By("simulating pods becoming ready, the way a kubelet would drive Deployment status")
		Expect(k8sClient.Get(ctx, key, &dep)).To(Succeed())
		dep.Status.ObservedGeneration = dep.Generation
		dep.Status.ReadyReplicas = 2
		dep.Status.Replicas = 2
		Expect(k8sClient.Status().Update(ctx, &dep)).To(Succeed())

		Eventually(func() string {
			_ = k8sClient.Get(ctx, key, webapp)
			return string(webapp.Status.Phase)
		}).Should(Equal(string(appsv1beta1.WebAppPhaseAvailable)))
		Eventually(func() []metav1.Condition {
			_ = k8sClient.Get(ctx, key, webapp)
			return webapp.Status.Conditions
		}).Should(ContainElement(HaveField("Type", appsv1beta1.ConditionAvailable)))
	})

	It("recreates a deleted owned Service without any test code calling Reconcile", func() {
		webapp := &appsv1beta1.WebApp{
			ObjectMeta: metav1.ObjectMeta{Name: webAppName, Namespace: ns},
			Spec: appsv1beta1.WebAppSpec{
				Image: testImage, Port: 8080,
				Scaling: appsv1beta1.ScalingSpec{MinReplicas: 1, MaxReplicas: 1},
			},
		}
		Expect(k8sClient.Create(ctx, webapp)).To(Succeed())

		key := types.NamespacedName{Name: webAppName, Namespace: ns}
		var svc corev1.Service
		Eventually(func() error {
			return k8sClient.Get(ctx, key, &svc)
		}).Should(Succeed())
		originalUID := svc.UID

		Expect(k8sClient.Delete(ctx, &svc)).To(Succeed())

		// This Eventually is the whole point of the test: nothing here
		// calls Reconcile. The only thing that can make this pass is
		// SetupWithManager's Owns(&corev1.Service{}) watch noticing the
		// delete and re-queuing the WebApp on its own.
		Eventually(func() types.UID {
			var recreated corev1.Service
			if err := k8sClient.Get(ctx, key, &recreated); err != nil {
				return ""
			}
			return recreated.UID
		}).ShouldNot(Or(Equal(originalUID), BeEmpty()))
	})

	It("corrects a Deployment's replica count if something else changes it out of band", func() {
		webapp := &appsv1beta1.WebApp{
			ObjectMeta: metav1.ObjectMeta{Name: webAppName, Namespace: ns},
			Spec: appsv1beta1.WebAppSpec{
				Image: testImage, Port: 8080,
				Scaling: appsv1beta1.ScalingSpec{MinReplicas: 3, MaxReplicas: 3},
			},
		}
		Expect(k8sClient.Create(ctx, webapp)).To(Succeed())

		key := types.NamespacedName{Name: webAppName, Namespace: ns}
		var dep appsv1.Deployment
		Eventually(func() error {
			return k8sClient.Get(ctx, key, &dep)
		}).Should(Succeed())

		By("something other than the operator scales the Deployment directly")
		wrong := int32(1)
		dep.Spec.Replicas = &wrong
		Expect(k8sClient.Update(ctx, &dep)).To(Succeed())

		Eventually(func() int32 {
			_ = k8sClient.Get(ctx, key, &dep)
			return *dep.Spec.Replicas
		}).Should(Equal(int32(3)))
	})

	It("registers its hostname, and deregisters and finishes deleting on delete", func() {
		hostName := ns + ".apps.example.com"
		webapp := &appsv1beta1.WebApp{
			ObjectMeta: metav1.ObjectMeta{Name: webAppName, Namespace: ns},
			Spec: appsv1beta1.WebAppSpec{
				Image: testImage, Port: 8080, HostName: hostName,
				Scaling: appsv1beta1.ScalingSpec{MinReplicas: 1, MaxReplicas: 1},
			},
		}
		Expect(k8sClient.Create(ctx, webapp)).To(Succeed())

		key := types.NamespacedName{Name: webAppName, Namespace: ns}
		By("the finalizer is attached before the registry entry appears")
		Eventually(func() []string {
			_ = k8sClient.Get(ctx, key, webapp)
			return webapp.Finalizers
		}).Should(ContainElement(appsv1beta1.FinalizerName))

		By("the hostname is registered")
		var cm corev1.ConfigMap
		regKey := types.NamespacedName{Namespace: registry.Namespace, Name: registry.ConfigMapName}
		Eventually(func() string {
			_ = k8sClient.Get(ctx, regKey, &cm)
			return cm.Data[hostName]
		}).Should(Equal(ns + "/" + webAppName))

		By("deleting the WebApp deregisters the hostname and the object actually goes away")
		Expect(k8sClient.Delete(ctx, webapp)).To(Succeed())

		Eventually(func() bool {
			_ = k8sClient.Get(ctx, regKey, &cm)
			_, present := cm.Data[hostName]
			return present
		}).Should(BeFalse())

		Eventually(func() bool {
			err := k8sClient.Get(ctx, key, webapp)
			return errors.IsNotFound(err)
		}).Should(BeTrue())
	})
})
