#!/usr/bin/env python3
"""Verify that every chain RPC in the shipped registry answers eth_chainId.

Run from the repository root:

    python3 tools/verify_chain_rpcs.py          # every endpoint
    python3 tools/verify_chain_rpcs.py --new    # only entries added since the
                                                # baseline in tools/cache


Reads go/wallet_api/chains_evm_data.go, posts eth_chainId to each primary RPC
(and each alternate) and compares the returned id with the declared one. A
mismatch or a timeout is reported so the entry can be fixed or removed rather
than shipped as a chain that does not work.
"""
import concurrent.futures
import json
import re
import sys
import urllib.error
import urllib.request

TARGET = "go/wallet_api/chains_evm_data.go"
BASELINE = "tools/cache/chains_evm_data.baseline.go"
TIMEOUT = 12


def parse_chains(path):
    """Parses the generated registry into (id, name, rpc, alternates) tuples.

    Entries are split on their brace boundaries first: matching fields across
    the whole file with a non-greedy regex silently pairs one entry's name with
    the next entry's alternates.
    """
    text = open(path, encoding="utf-8").read()
    start = text.find("var evmMainnet = []ChainConfig{")
    end = text.find("const evmMainnetCount")
    body = text[start:end]

    chains = []
    depth = 0
    entry = ""
    for line in body.splitlines(keepends=True):
        if line.strip().startswith("{"):
            depth += 1
            entry = ""
        if depth > 0:
            entry += line
        if depth > 0 and line.strip().rstrip(",") == "}":
            depth = 0
            cid = re.search(r"ID:\s*(\d+)", entry)
            name = re.search(r'Name:\s*"([^"]+)"', entry)
            rpc = re.search(r'RPCEndpoint:\s*"([^"]+)"', entry)
            alts = re.search(r"RPCAlternates:\s*\[\]string\{([^}]*)\}", entry)
            if cid and name and rpc:
                chains.append(
                    (
                        int(cid.group(1)),
                        name.group(1),
                        rpc.group(1),
                        re.findall(r'"([^"]+)"', alts.group(1)) if alts else [],
                    )
                )
    return chains


def probe(chain_id, name, url):
    payload = json.dumps(
        {"jsonrpc": "2.0", "id": 1, "method": "eth_chainId", "params": []}
    ).encode()
    request = urllib.request.Request(
        url,
        data=payload,
        headers={"Content-Type": "application/json", "User-Agent": "tigerwallet-registry-check/1.0"},
    )
    try:
        with urllib.request.urlopen(request, timeout=TIMEOUT) as response:
            body = json.loads(response.read().decode("utf-8", "replace"))
    except Exception as exc:  # noqa: BLE001 - the point is to record any failure
        return (chain_id, name, url, None, type(exc).__name__)
    result = body.get("result")
    if isinstance(result, str) and result.startswith("0x"):
        return (chain_id, name, url, int(result, 16), None)
    error = body.get("error")
    message = error.get("message") if isinstance(error, dict) else str(error)
    return (chain_id, name, url, None, (message or "no result")[:70])


def main():
    chains = parse_chains(TARGET)
    only_new = "--new" in sys.argv
    if only_new:
        old_ids = {
            int(m) for m in re.findall(r"ID:\s*(\d+),", open(BASELINE, encoding="utf-8").read())
        }
        chains = [c for c in chains if c[0] not in old_ids]

    targets = []
    for cid, name, rpc, alts in chains:
        targets.append((cid, name, rpc))
        for alt in alts:
            targets.append((cid, name, alt))

    print(f"probing {len(targets)} endpoint(s) across {len(chains)} chain(s)")
    ok = mismatched = unreachable = 0
    problems = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
        for cid, name, url, got, error in pool.map(lambda t: probe(*t), targets):
            if got == cid:
                ok += 1
            elif got is None:
                unreachable += 1
                problems.append(("unreachable", cid, name, url, error))
            else:
                mismatched += 1
                problems.append(("mismatch", cid, name, url, f"returned {got}"))

    print(f"ok: {ok}  unreachable: {unreachable}  mismatched: {mismatched}")
    for kind, cid, name, url, detail in problems:
        print(f"  [{kind}] {cid} {name} {url} -> {detail}")
    return 0 if mismatched == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
