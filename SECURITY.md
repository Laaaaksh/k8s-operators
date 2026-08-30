# Security Policy

k8s-operators is a learning curriculum plus one real, runnable operator
(`code/webapp-operator`) built to teach the concepts, not to be installed in
a production cluster. It does have a real attack surface, though - a running
controller with a ServiceAccount, RBAC, and TLS-terminated admission
webhooks - so it's worth being specific about what that means here.

## What belongs in a private report

Worth reporting privately:

- A gap between what `curriculum/08-least-privilege-rbac.md` or a code
  comment claims about the operator's RBAC and what `config/rbac/role.yaml`
  actually grants.
- A webhook (`internal/webhook/v1beta1`) that fails open on a TLS or
  authentication error instead of rejecting the request.
- Anything in `internal/registry` or the finalizer path that could let one
  WebApp read, hijack, or block another WebApp's registry entry across
  namespaces in a way this repo doesn't already document as expected
  multi-tenant behavior.
- A build script, Makefile target, or CI workflow that fetches and executes
  something from the network without saying so.

Not a security issue, just a normal bug report (open a public issue instead):

- A reconcile loop that produces the wrong Deployment/Service spec.
- A flaky or incorrect test.
- A dead or incorrect link in the curated resources.
- envtest- or kind-specific quirks that don't reflect a real cluster.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting:

> https://github.com/Laaaaksh/k8s-operators/security/advisories/new

That reaches the maintainer privately so any real issue can be fixed before
it's discussed in public.

## Credits

Reporters who wish to be credited may say so in the private report; otherwise
reports are handled without attribution.
