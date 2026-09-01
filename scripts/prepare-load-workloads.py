#!/usr/bin/env python3
"""Remove resources the load environment must not provision. Takes a *copy* of
.fam/workloads, never the committed files.

Must run before `score render`, not after `score k8s`: a resource type with no
local provisioner fails the entire workload at generate time, leaving nothing
to strip.
"""

import sys

import yaml

DROP_RESOURCES = {
    "flexprice-api.yaml": ["temporal", "clickhouse-migration-job"],
}


def main(workload_dir):
    for filename, resources in DROP_RESOURCES.items():
        path = f"{workload_dir}/{filename}"
        try:
            with open(path) as fh:
                doc = yaml.safe_load(fh)
        except FileNotFoundError:
            sys.exit(f"prepare-load-workloads: {path} not found")

        declared = doc.get("resources") or {}
        for name in resources:
            if name not in declared:
                sys.exit(
                    f"prepare-load-workloads: {filename} has no '{name}' resource. "
                    "It may have been renamed — confirm the load environment is not "
                    "about to provision it."
                )
            del declared[name]
            print(f"  dropped  {filename}: resources.{name}")

        with open(path, "w") as fh:
            yaml.dump(doc, fh, default_flow_style=False, sort_keys=False)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit("usage: prepare-load-workloads.py <copied-workload-dir>")
    main(sys.argv[1])
