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

// Package leaderelection isn't operator business logic - there's nothing
// to import from cmd/main.go here. It exists to make one specific claim
// about --leader-elect checkable instead of trusted on faith: "when two
// replicas of this manager run at once, exactly one of them is active,
// and if that one dies, the other takes over." That's the actual promise
// leader election makes, and curriculum/07-leader-election.md points here
// as its checkpoint.
package leaderelection

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	testEnv *envtest.Environment
	cfg     *rest.Config
)

func TestLeaderElection(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Leader Election Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	testEnv = &envtest.Environment{}
	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testEnv.Stop()).To(Succeed())
})

// newCandidate builds a manager configured exactly like cmd/main.go's,
// with LeaderElection on and the same LeaderElectionID two real replicas
// of this operator would share - that's what makes them contend for the
// same Lease instead of each electing itself.
func newCandidate() (manager.Manager, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                  scheme.Scheme,
		Metrics:                 metricsserver.Options{BindAddress: "0"},
		LeaderElection:          true,
		LeaderElectionID:        "53ec4676.example.com",
		LeaderElectionNamespace: "default",
		// Shortened so the test doesn't take minutes - production defaults
		// (15s/10s/2s) trade a slower failover for fewer spurious elections
		// under normal network jitter. See curriculum/07-leader-election.md
		// for why the defaults are that big.
		LeaseDuration: durationPtr(2 * time.Second),
		RenewDeadline: durationPtr(1500 * time.Millisecond),
		RetryPeriod:   durationPtr(500 * time.Millisecond),
	})
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		_ = mgr.Start(ctx)
	}()
	return mgr, cancel
}

func durationPtr(d time.Duration) *time.Duration { return &d }

var _ = Describe("Leader election", func() {
	It("elects exactly one of two contending managers, and promotes the other on failover", func() {
		Expect(createNamespaceIfMissing(cfg, "default")).To(Succeed())

		mgrA, cancelA := newCandidate()
		mgrB, cancelB := newCandidate()
		defer cancelB()

		By("exactly one of the two becomes leader")
		var leaderIsA, leaderIsB bool
		Eventually(func() bool {
			select {
			case <-mgrA.Elected():
				leaderIsA = true
			default:
			}
			select {
			case <-mgrB.Elected():
				leaderIsB = true
			default:
			}
			return leaderIsA != leaderIsB // exactly one, not both, not neither
		}, 10*time.Second, 100*time.Millisecond).Should(BeTrue())

		By("stopping the leader lets the other one take over")
		cancelA()
		if leaderIsA {
			Eventually(func() bool {
				select {
				case <-mgrB.Elected():
					return true
				default:
					return false
				}
			}, 15*time.Second, 200*time.Millisecond).Should(BeTrue(), "the standby should become leader after the incumbent stops")
		}
	})
})

func createNamespaceIfMissing(cfg *rest.Config, name string) error {
	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return err
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := c.Create(context.Background(), ns); err != nil && !isAlreadyExists(err) {
		return err
	}
	return nil
}

func isAlreadyExists(err error) bool {
	type statusErr interface{ Status() metav1.Status }
	se, ok := err.(statusErr)
	return ok && se.Status().Reason == metav1.StatusReasonAlreadyExists
}
