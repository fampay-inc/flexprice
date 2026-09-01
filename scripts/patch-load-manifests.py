#!/usr/bin/env python3
"""Strip resources the load environment must not have, in place.

SecurityPolicy is the Envoy ext_authz to hagrid; SecurityGroupPolicy references a
prod SG absent from the load account; Namespace is created by arrakis itself.

--expect-hagrid fails when nothing was stripped, so a score template rename
cannot silently ship manifests that still reach hagrid. flexprice omits it.
"""

import argparse
import sys

import yaml

DROP_KINDS = {
    "SecurityPolicy",
    "SecurityGroupPolicy",
    "Application",
    "Namespace",
}

AUTH_LABEL = "security"

# score's alb-ingress patch template hardcodes the group per env and ignores any
# user value, so this is the one annotation config.load.json cannot reach.
ALB_GROUP_ANNOTATION = "alb.ingress.kubernetes.io/group.name"
ALB_GROUP = "eks-internal-lb-group"


def main(path, expect_hagrid):
    with open(path) as fh:
        docs = [d for d in yaml.safe_load_all(fh) if d]

    kept, dropped, unlabelled, regrouped = [], [], 0, 0

    for doc in docs:
        kind = doc.get("kind", "")
        if kind in DROP_KINDS:
            dropped.append(f"{kind}/{doc.get('metadata', {}).get('name', '?')}")
            continue

        if kind == "Ingress":
            ann = doc.setdefault("metadata", {}).setdefault("annotations", {})
            if ann.get(ALB_GROUP_ANNOTATION) != ALB_GROUP:
                ann[ALB_GROUP_ANNOTATION] = ALB_GROUP
                regrouped += 1

        # SecurityPolicy attaches via spec.targetSelectors.matchLabels, so the
        # label is what makes a route authenticated.
        if kind == "HTTPRoute":
            labels = doc.get("metadata", {}).get("labels") or {}
            if AUTH_LABEL in labels:
                del labels[AUTH_LABEL]
                unlabelled += 1
                if not labels:
                    del doc["metadata"]["labels"]

        kept.append(doc)

    if expect_hagrid:
        if not any(d.startswith("SecurityPolicy/") for d in dropped):
            sys.exit(
                "patch-load-manifests: no SecurityPolicy found. The score template "
                "may have renamed it — verify hagrid is genuinely absent before "
                "proceeding."
            )
        if unlabelled == 0:
            sys.exit(
                f"patch-load-manifests: no HTTPRoute carried a '{AUTH_LABEL}' label. "
                "The auth-class wiring has changed; re-check what attaches ext_authz."
            )

    with open(path, "w") as fh:
        yaml.dump_all(kept, fh, default_flow_style=False, sort_keys=False,
                      explicit_start=True)

    for d in dropped:
        print(f"  dropped  {d}")
    print(f"  unlabelled {unlabelled} HTTPRoute(s)")
    print(f"  regrouped  {regrouped} Ingress to {ALB_GROUP}")
    print(f"  {len(kept)} of {len(docs)} documents kept")


if __name__ == "__main__":
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("manifests")
    ap.add_argument(
        "--expect-hagrid",
        action="store_true",
        help="fail if no hagrid SecurityPolicy or auth-class route was found",
    )
    args = ap.parse_args()
    main(args.manifests, args.expect_hagrid)
