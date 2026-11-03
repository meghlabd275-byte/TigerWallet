#!/usr/bin/env python3
"""Download the canonical EVM chain registry.

The registry is ethereum-lists/chains, published as chainid.network/chains.json.
It is cached under tools/cache/ so that regeneration works offline afterwards.

    python3 tools/fetch_chain_registry.py
"""
import json
import os
import sys
import urllib.request

URL = "https://chainid.network/chains.json"
TARGET = "tools/cache/chains.json"


def main():
    os.makedirs(os.path.dirname(TARGET), exist_ok=True)
    request = urllib.request.Request(URL, headers={"User-Agent": "tigerwallet-registry-fetch/1.0"})
    with urllib.request.urlopen(request, timeout=120) as response:
        raw = response.read()
    # Parse before writing so a truncated download does not replace a good cache.
    registry = json.loads(raw.decode("utf-8"))
    with open(TARGET, "w", encoding="utf-8") as handle:
        json.dump(registry, handle, separators=(",", ":"))
    print(f"wrote {TARGET}: {len(registry)} entries, {len(raw)} bytes")
    return 0


if __name__ == "__main__":
    sys.exit(main())
