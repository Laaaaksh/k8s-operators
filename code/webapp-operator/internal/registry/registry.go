// Package registry models the one piece of state this operator manages
// that Kubernetes' own garbage collector cannot clean up for it: a shared
// hostname -> WebApp mapping, held in a single ConfigMap that every WebApp
// in the cluster writes into.
//
// Why this needs a finalizer, when the Deployment and Service don't:
// ownerReferences (what makes the Deployment/Service get garbage-collected
// automatically when a WebApp is deleted) only work for objects owned by
// exactly one parent. The registry ConfigMap is shared - every WebApp in
// the cluster writes one entry into the same object - so it can't have a
// single owner. Nothing deletes a WebApp's entry for you; the controller
// has to do it itself, before the WebApp object is actually gone. That's
// exactly what a finalizer blocks deletion long enough to do. See
// curriculum/05-finalizers.md.
package registry

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigMapName and Namespace locate the cluster's single registry
// ConfigMap. A real system would make this configurable; it's fixed here
// to keep the example focused on the finalizer pattern, not on config
// plumbing.
const (
	ConfigMapName = "webapp-registry"
	Namespace     = "webapp-system"
)

// ErrHostNameTaken is returned by Register when hostName is already
// registered to a different WebApp.
type ErrHostNameTaken struct {
	HostName string
	Owner    string
}

func (e *ErrHostNameTaken) Error() string {
	return fmt.Sprintf("hostname %q is already registered to %s", e.HostName, e.Owner)
}

// entryKey is how a WebApp identifies itself in the registry: namespace
// and name, joined, so the same hostname string can't collide across two
// WebApps that happen to share a name in different namespaces registering
// the same key by accident.
func entryKey(namespace, name string) string {
	return namespace + "/" + name
}

// Register records that hostName belongs to the given WebApp, failing if
// another WebApp already holds it. It's safe to call every reconcile: if
// this WebApp already owns hostName, it's a no-op.
//
// The retry.RetryOnConflict wrapper exists because this ConfigMap is
// shared - two WebApp reconciles in different goroutines (controller-runtime
// runs up to MaxConcurrentReconciles workers) can race to update it. A
// conflict here isn't a bug, it's healthy contention; the fix is "read the
// latest version and try again," which is exactly what RetryOnConflict
// automates. Plain client.Update without this would make one of the two
// reconciles fail permanently instead of just retrying.
func Register(ctx context.Context, c client.Client, namespace, name, hostName string) error {
	if hostName == "" {
		return nil
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cm, err := getOrCreateRegistry(ctx, c)
		if err != nil {
			return err
		}

		key := entryKey(namespace, name)
		if owner, taken := cm.Data[hostName]; taken && owner != key {
			return &ErrHostNameTaken{HostName: hostName, Owner: owner}
		}
		if cm.Data[hostName] == key {
			return nil // already registered to us; nothing to do
		}

		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data[hostName] = key
		return c.Update(ctx, cm)
	})
}

// Deregister removes any entry this WebApp holds. Called from the
// reconciler's finalizer path, so it must tolerate the registry
// ConfigMap already being gone (e.g. the webapp-system namespace was
// deleted first) - that's success, not an error, since the goal
// ("no entry for this WebApp exists") is already true.
func Deregister(ctx context.Context, c client.Client, namespace, name string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var cm corev1.ConfigMap
		err := c.Get(ctx, types.NamespacedName{Namespace: Namespace, Name: ConfigMapName}, &cm)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}

		key := entryKey(namespace, name)
		found := false
		for host, owner := range cm.Data {
			if owner == key {
				delete(cm.Data, host)
				found = true
			}
		}
		if !found {
			return nil
		}
		return c.Update(ctx, &cm)
	})
}

func getOrCreateRegistry(ctx context.Context, c client.Client) (*corev1.ConfigMap, error) {
	if err := ensureNamespace(ctx, c); err != nil {
		return nil, err
	}

	var cm corev1.ConfigMap
	err := c.Get(ctx, types.NamespacedName{Namespace: Namespace, Name: ConfigMapName}, &cm)
	if apierrors.IsNotFound(err) {
		cm = corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: ConfigMapName, Namespace: Namespace},
			Data:       map[string]string{},
		}
		createErr := c.Create(ctx, &cm)
		if createErr == nil {
			// c.Create populates cm in place with the server's response
			// (resourceVersion, UID, ...) - that IS the authoritative
			// object. An earlier version of this function re-fetched it
			// with a Get here instead, on the assumption a fresher read
			// couldn't hurt. It could: mgr.GetClient() is cache-backed,
			// and a Get for a just-created object can race the cache's
			// watch before it's observed the create, returning a stale
			// NotFound for an object that demonstrably exists. Caught
			// running this repo's own operator against a real kind
			// cluster - see curriculum/04-envtest.md's note on what
			// envtest's single-process cache hides that a real cluster's
			// network latency exposes. Use what Create already gave you
			// instead of re-reading it back.
			return &cm, nil
		}
		if !apierrors.IsAlreadyExists(createErr) {
			return nil, fmt.Errorf("creating registry ConfigMap: %w", createErr)
		}
		// Lost the create race to another reconcile - this Get is for a
		// ConfigMap we know exists (the AlreadyExists above proves it),
		// so unlike the case above there's no fresher answer than reading
		// it back.
		if err := c.Get(ctx, types.NamespacedName{Namespace: Namespace, Name: ConfigMapName}, &cm); err != nil {
			return nil, err
		}
		return &cm, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting registry ConfigMap: %w", err)
	}
	return &cm, nil
}

func ensureNamespace(ctx context.Context, c client.Client) error {
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: Namespace}}
	err := c.Create(ctx, &ns)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating registry namespace: %w", err)
	}
	return nil
}
