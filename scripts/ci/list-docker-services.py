#!/usr/bin/env python3
"""Emit the container image build matrix for CI.

Parses every compose file in the repository and resolves each service's build
context, Dockerfile and image name. The output is a GitHub Actions matrix so CI
builds *the whole stack* rather than a hand-maintained subset that silently
drifts away from what actually ships.

Usage:
    list-docker-services.py [--compose FILE ...] [--output FILE]

Writes {"include": [{"name","context","dockerfile","image"}, ...]} to stdout
(or --output). Exit code is non-zero when a build context or Dockerfile is
missing, so the pipeline fails loudly instead of skipping a service.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys

DEFAULT_COMPOSE = ["docker-compose.yml", "deployments/docker/docker-compose.yml"]

# Pull-only infrastructure: nothing in this repository builds them.
IMAGE_ONLY_SERVICES = {"postgres", "redis"}

# Operator-supplied Dockerfile templates, not deployable stack services.
EXCLUDED_CONTEXT_PREFIXES = ("devops/docker/", "infrastructure/docker/")


def service_blocks(text):
    """Split a compose file into (service_name, block_text) pairs."""
    blocks = []
    current_name = None
    current = []
    in_services = False

    for line in text.splitlines():
        if re.match(r"^services:\s*$", line):
            in_services = True
            continue
        if in_services and re.match(r"^[a-zA-Z]", line):
            in_services = False
        if not in_services:
            continue

        m = re.match(r"^  ([A-Za-z0-9_.-]+):\s*$", line)
        if m:
            if current_name is not None:
                blocks.append((current_name, "\n".join(current)))
            current_name = m.group(1)
            current = []
            continue
        if current_name is not None:
            current.append(line)

    if current_name is not None:
        blocks.append((current_name, "\n".join(current)))
    return blocks


def resolve(compose_path):
    root = os.path.dirname(os.path.abspath(compose_path))
    with open(compose_path, encoding="utf-8") as fh:
        text = fh.read()

    found = []
    for name, block in service_blocks(text):
        if name in IMAGE_ONLY_SERVICES:
            continue
        ctx = re.search(r"^\s*context:\s*(\S+)\s*$", block, re.M)
        if not ctx:
            continue
        context = ctx.group(1)
        dockerfile = "Dockerfile"
        df = re.search(r"^\s*dockerfile:\s*(\S+)\s*$", block, re.M)
        if df:
            dockerfile = df.group(1)

        # Build arguments are part of the image identity: the shared
        # root-context Go Dockerfile selects which module it compiles via
        # TARGET_MODULE, so dropping them would silently build the wrong
        # binary (the default) into a named service image.
        build_args = {}
        args_block = re.search(r"^\s{6,}args:\s*$", block, re.M)
        if args_block:
            for line in block[args_block.end():].splitlines():
                stripped = line.strip()
                if not stripped or stripped.startswith("#"):
                    continue
                arg = re.match(r"^\s{8,}([A-Za-z_][A-Za-z0-9_]*):\s*(.*?)\s*$", line)
                if not arg:
                    break
                value = arg.group(2).strip().strip("'\"")
                build_args[arg.group(1)] = value

        abs_context = os.path.normpath(os.path.join(root, context))
        rel_context = os.path.relpath(abs_context, os.getcwd())
        abs_dockerfile = os.path.normpath(os.path.join(abs_context, dockerfile))

        if rel_context.startswith(".."):
            continue
        normalised = rel_context[2:] if rel_context.startswith("./") else rel_context
        if any(normalised.startswith(p) for p in EXCLUDED_CONTEXT_PREFIXES):
            continue
        if not os.path.isdir(abs_context):
            print(
                "error: %s: service %r build context %r does not exist"
                % (compose_path, name, rel_context),
                file=sys.stderr,
            )
            raise SystemExit(2)
        if not os.path.isfile(abs_dockerfile):
            print(
                "error: %s: service %r references missing Dockerfile %r"
                % (compose_path, name, os.path.relpath(abs_dockerfile, os.getcwd())),
                file=sys.stderr,
            )
            raise SystemExit(2)

        found.append(
            {
                "name": name,
                "context": rel_context,
                "dockerfile": os.path.relpath(abs_dockerfile, abs_context),
                "image": name,
                "build_args": ["%s=%s" % (k, v) for k, v in sorted(build_args.items())],
            }
        )
    return found


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--compose", action="append", default=None)
    ap.add_argument("--output")
    args = ap.parse_args()

    compose_files = args.compose or DEFAULT_COMPOSE
    seen = {}
    for path in compose_files:
        if not os.path.isfile(path):
            continue
        for entry in resolve(path):
            # First compose file wins: docker-compose.yml is the canonical
            # full stack; deployments/docker is the reduced infra variant.
            seen.setdefault(entry["name"], entry)

    payload = json.dumps({"include": [seen[k] for k in sorted(seen)]}, indent=2)
    if args.output:
        with open(args.output, "w", encoding="utf-8") as fh:
            fh.write(payload + "\n")
    else:
        print(payload)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
