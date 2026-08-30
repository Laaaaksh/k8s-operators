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

// Hub marks WebApp as the conversion hub: the version every other served
// version converts through, and the version this repo's controller and
// webhooks are written against from Stage 10 onward. It's an empty
// method - satisfying sigs.k8s.io/controller-runtime/pkg/conversion.Hub
// is a type-level declaration, not behavior. See
// api/v1alpha1/webapp_conversion.go for the spoke side of this, and
// curriculum/10-versioning.md for why exactly one version gets to be Hub.
func (*WebApp) Hub() {}
