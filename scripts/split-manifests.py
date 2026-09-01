#!/usr/bin/env python3
"""Split a concatenated manifests.yaml into flat per-resource files.

arrakis globs `<dir>/*.yaml` non-recursively and skips any basename containing the
literal "serviceaccount" (src/external/pulumi_k8s/adapter.go). Nothing else creates
the ServiceAccount, so `felix-serviceaccount.yaml` would silently break IRSA —
hence the hyphenated suffixes below, and the assertion enforcing them.
"""

import os
import re
import sys

import yaml

# Matches the naming already in k8s/<service>/load-test/. Absent kinds fall back
# to a kebab-cased kind, which can never produce the literal "serviceaccount".
KIND_SUFFIX = {
    "ConfigMap": "configmap",
    "Deployment": "deployment",
    "ExternalSecret": "external-secret",
    "HorizontalPodAutoscaler": "hpa",
    "HTTPRoute": "http-route",
    "Ingress": "ingress",
    "PodDisruptionBudget": "pdb",
    "SecretStore": "secret-store",
    "Service": "service",
    "ServiceAccount": "service-account",
    "ServiceMonitor": "service-monitor",
}


def kebab(kind):
    return re.sub(r"(?<!^)(?=[A-Z])", "-", kind).lower()


def filename(kind, name):
    """`<name>-<suffix>.yaml`, dropping the suffix when the name already says it.

    Score names resources after their kind (`felix-api-route`), so appending
    blindly yields `felix-cache-creds-secrets-manager-secret-external-secret`.
    """
    suffix = KIND_SUFFIX.get(kind, kebab(kind))
    last_word = suffix.rsplit("-", 1)[-1]
    if name == suffix or name.endswith(f"-{suffix}") or name.endswith(f"-{last_word}"):
        return f"{name}.yaml"
    return f"{name}-{suffix}.yaml"


def main(src, dest):
    with open(src) as fh:
        docs = [d for d in yaml.safe_load_all(fh) if d]

    os.makedirs(dest, exist_ok=True)
    for stale in os.listdir(dest):
        if stale.endswith(".yaml"):
            os.remove(os.path.join(dest, stale))

    seen = {}
    for doc in docs:
        kind = doc.get("kind", "Unknown")
        name = (doc.get("metadata") or {}).get("name", "unnamed")
        fname = filename(kind, name)

        assert "serviceaccount" not in fname.lower(), (
            f"{fname} contains 'serviceaccount' — arrakis would skip this file. "
            "Use a hyphenated suffix in KIND_SUFFIX."
        )

        if fname in seen:
            sys.exit(
                f"split-manifests: {fname} collides — {seen[fname]} and {kind}/{name} "
                "map to the same filename"
            )
        seen[fname] = f"{kind}/{name}"

        with open(os.path.join(dest, fname), "w") as fh:
            yaml.dump(doc, fh, default_flow_style=False, sort_keys=False)

    print(f"  wrote {len(seen)} files to {dest}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit("usage: split-manifests.py <manifests.yaml> <dest-dir>")
    main(sys.argv[1], sys.argv[2])
