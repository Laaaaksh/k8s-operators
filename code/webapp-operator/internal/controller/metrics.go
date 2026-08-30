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
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	appsv1beta1 "github.com/Laaaaksh/k8s-operators/code/webapp-operator/api/v1beta1"
)

// controller-runtime already registers a standard set of metrics on this
// same registry for every controller, for free -
// controller_runtime_reconcile_total (with a "result" label of
// success/error/requeue) and controller_runtime_reconcile_time_seconds
// among them. See https://book.kubebuilder.io/reference/metrics.html.
// Nothing below duplicates those - they already answer "is reconciliation
// healthy." webappPhase exists because it answers a WebApp-specific
// question controller-runtime has no way to know: "what phase is this
// WebApp in right now." That's the dividing line for when a custom metric
// earns its keep, rather than just re-exposing what's already free.
var webappPhase = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "webapp_operator_webapp_phase",
		Help: "Always 1 for the WebApp's current status.phase; absent for every other phase. " +
			"A kube-state-metrics-style '_info' gauge: sum by (phase) to get a count per phase.",
	},
	[]string{"namespace", "name", "phase"},
)

func init() {
	metrics.Registry.MustRegister(webappPhase)
}

// allPhases lists every value WebAppPhase can take, so recordPhaseMetric
// can delete the (namespace,name,phase) series for whichever phases the
// WebApp is NOT currently in. Without that cleanup, a WebApp that moved
// from Progressing to Available would leave a stale
// webapp_operator_webapp_phase{phase="Progressing"} series reading 1
// forever - Prometheus has no concept of "this label combination no
// longer applies" unless the exporter deletes it.
var allPhases = []appsv1beta1.WebAppPhase{
	appsv1beta1.WebAppPhasePending,
	appsv1beta1.WebAppPhaseProgressing,
	appsv1beta1.WebAppPhaseAvailable,
	appsv1beta1.WebAppPhaseDegraded,
}

// recordPhaseMetric updates webappPhase after every reconcile - one Set
// for the current phase, one Delete for each of the others. Called
// unconditionally, even when the phase didn't change, because Delete on a
// series that isn't there is a harmless no-op and this stays correct
// without the reconciler having to track "what was the phase last time."
func recordPhaseMetric(namespace, name string, phase appsv1beta1.WebAppPhase) {
	for _, p := range allPhases {
		labels := phaseLabels(namespace, name, p)
		if p == phase {
			webappPhase.With(labels).Set(1)
		} else {
			webappPhase.Delete(labels)
		}
	}
}

// deletePhaseMetric removes every series for a WebApp that's being
// deleted, called from the finalizer path. Otherwise a deleted WebApp's
// last-known phase would read as 1 forever - the gauge would claim a
// WebApp exists after `kubectl get webapp` shows nothing.
func deletePhaseMetric(namespace, name string) {
	for _, p := range allPhases {
		webappPhase.Delete(phaseLabels(namespace, name, p))
	}
}

func phaseLabels(namespace, name string, phase appsv1beta1.WebAppPhase) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "name": name, "phase": string(phase)}
}
