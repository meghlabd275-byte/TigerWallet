#!/usr/bin/env python3
"""Regression tests for the EVM chain-registry generator.

The generator rewrites go/wallet_api/chains_evm_data.go, a checked-in file that
Go builds from. Two defects lived here unnoticed because nothing ran the
generator after the file was committed:

  * parse_existing matched from the opening brace, so the leading indentation was
    dropped and a regenerated file was not gofmt-clean;
  * comment lines were emitted as "// " even when empty, adding trailing
    whitespace to the same effect.

Both are silent: the file still compiles. They only surface as a large,
meaningless diff the next time someone regenerates. These tests pin the property
that matters - regenerating from the committed file reproduces it exactly - so a
regression fails here rather than as an unexplained diff.

Run: python3 tools/test_gen_evm_chains.py
"""
import importlib.util
import os
import subprocess
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GEN = os.path.join(ROOT, "tools", "gen_evm_chains.py")
TARGET = os.path.join(ROOT, "go", "wallet_api", "chains_evm_data.go")

failures = []


def check(condition, message):
    print(f"  {'ok  ' if condition else 'FAIL'} {message}")
    if not condition:
        failures.append(message)


def load_generator():
    spec = importlib.util.spec_from_file_location("gen_evm_chains", GEN)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main():
    if not os.path.exists(os.path.join(ROOT, "tools", "cache", "chains.json")):
        print("tools/cache/chains.json is absent; run tools/fetch_chain_registry.py first")
        return 1

    gen = load_generator()
    entries, text = gen.parse_existing(TARGET)
    platform_ids = {c["chainId"] for c in gen.PLATFORM_CHAINS}

    print("parse_existing preserves each entry verbatim")
    check(len(entries) > 100, f"parsed {len(entries)} entries")
    bad_indent = [cid for cid, raw in entries.items() if not raw.startswith("\t{")]
    check(not bad_indent, f"every parsed entry keeps its indentation (bad: {bad_indent[:5]})")
    bad_id = [cid for cid, raw in entries.items() if f"ID: {cid}," not in raw]
    check(not bad_id, f"every parsed entry keeps its ID field (bad: {bad_id[:5]})")
    bad_close = [cid for cid, raw in entries.items() if not raw.endswith("\t},")]
    check(not bad_close, f"every parsed entry ends at the closing brace (bad: {bad_close[:5]})")

    # The platform chains are emitted from PLATFORM_CHAINS rather than preserved,
    # so the parsed count is the file's entry count minus those.
    expected_parsed = text.count("\t{") - len(platform_ids - set(entries))
    check(
        len(entries) == expected_parsed,
        f"parsed {len(entries)} entries, expected {expected_parsed} "
        f"(file has {text.count(chr(9) + '{')} entries, {len(platform_ids)} platform chains)",
    )

    print("regeneration reproduces the committed file")
    before = open(TARGET, encoding="utf-8").read()
    result = subprocess.run([sys.executable, GEN], cwd=ROOT, capture_output=True, text=True)
    check(result.returncode == 0, f"generator exits 0 (stderr: {result.stderr.strip()[:200]})")
    after = open(TARGET, encoding="utf-8").read()
    if before != after:
        # Leave the tree usable before reporting the failure.
        open(TARGET, "w", encoding="utf-8").write(before)
    check(before == after, "regenerating from the committed file changes nothing")

    print("generated output is clean")
    trailing = [i + 1 for i, line in enumerate(after.splitlines()) if line != line.rstrip()]
    check(not trailing, f"no line has trailing whitespace (bad lines: {trailing[:5]})")

    print()
    if failures:
        print(f"{len(failures)} check(s) failed")
        return 1
    print("all checks passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
