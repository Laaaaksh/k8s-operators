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
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	appsv1beta1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1beta1"
)

// This is a plain `go test`, not a Ginkgo spec in the envtest suite - it's
// worth noticing the difference. Metrics logic is pure computation on a
// prometheus.Registry; it never touches the API server, so it doesn't need
// the real kube-apiserver envtest boots. Reaching for envtest here would
// only make the test slower without making it more correct - see
// curriculum/04-envtest.md's note on when *not* to reach for it.
func TestRecordPhaseMetric(t *testing.T) {
	recordPhaseMetric("team-a", "site", appsv1beta1.WebAppPhaseProgressing)

	if got := testutil.ToFloat64(webappPhase.WithLabelValues("team-a", "site", "Progressing")); got != 1 {
		t.Fatalf("Progressing gauge = %v, want 1", got)
	}

	// Moving to Available must both set the new series and clear the old
	// one - a stale Progressing=1 left behind would make a dashboard sum
	// double-count this WebApp.
	recordPhaseMetric("team-a", "site", appsv1beta1.WebAppPhaseAvailable)

	if got := testutil.ToFloat64(webappPhase.WithLabelValues("team-a", "site", "Available")); got != 1 {
		t.Fatalf("Available gauge = %v, want 1", got)
	}
	if n := testutil.CollectAndCount(webappPhase, "webapp_operator_webapp_phase"); n != 1 {
		t.Fatalf("expected exactly 1 series for team-a/site after transition, got %d", n)
	}

	deletePhaseMetric("team-a", "site")
	if n := testutil.CollectAndCount(webappPhase, "webapp_operator_webapp_phase"); n != 0 {
		t.Fatalf("expected 0 series after delete, got %d", n)
	}
}
