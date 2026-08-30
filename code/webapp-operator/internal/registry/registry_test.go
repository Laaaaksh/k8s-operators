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

package registry

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Registry", func() {
	It("registers a hostname, and re-registering the same owner is a no-op", func() {
		Expect(Register(ctx, k8sClient, "team-a", "site", "a.example.com")).To(Succeed())
		Expect(Register(ctx, k8sClient, "team-a", "site", "a.example.com")).To(Succeed())

		var cm corev1.ConfigMap
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: Namespace, Name: ConfigMapName}, &cm)).To(Succeed())
		Expect(cm.Data["a.example.com"]).To(Equal("team-a/site"))
	})

	It("rejects a hostname already registered to a different WebApp", func() {
		Expect(Register(ctx, k8sClient, "team-a", "site", "shared.example.com")).To(Succeed())

		err := Register(ctx, k8sClient, "team-b", "other", "shared.example.com")
		Expect(err).To(HaveOccurred())
		var taken *ErrHostNameTaken
		Expect(errorsAs(err, &taken)).To(BeTrue())
		Expect(taken.Owner).To(Equal("team-a/site"))
	})

	It("Deregister removes only the calling WebApp's entry, and tolerates being called twice", func() {
		Expect(Register(ctx, k8sClient, "team-a", "site", "gone.example.com")).To(Succeed())

		Expect(Deregister(ctx, k8sClient, "team-a", "site")).To(Succeed())
		Expect(Deregister(ctx, k8sClient, "team-a", "site")).To(Succeed()) // idempotent

		var cm corev1.ConfigMap
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: Namespace, Name: ConfigMapName}, &cm)).To(Succeed())
		_, stillThere := cm.Data["gone.example.com"]
		Expect(stillThere).To(BeFalse())
	})
})

// errorsAs is a thin wrapper so the test file doesn't need its own
// "errors" import just for this one call.
func errorsAs(err error, target **ErrHostNameTaken) bool {
	e, ok := err.(*ErrHostNameTaken)
	if ok {
		*target = e
	}
	return ok
}
