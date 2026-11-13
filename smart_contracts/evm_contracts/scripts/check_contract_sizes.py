#!/usr/bin/env python3
"""Fail when a deployable TigerWallet contract exceeds the EIP-170 code size limit.

`forge build --sizes` exits non-zero for ANY artifact over 24,576 bytes. That
includes the vendored eth-infinitism ERC-4337 reference EntryPoint, which this
repository never deploys: production clients call the canonical EntryPoint that
is already deployed on-chain, and only the test suite instantiates the vendored
copy. Failing the build on it would block every release for a contract that
cannot reach mainnet anyway.

This gate keeps the real protection (a TigerWallet contract that grows past the
limit fails the build) while excluding the vendored reference implementations.
"""

import json
import subprocess
import sys

# Vendored eth-infinitism ERC-4337 reference sources (GPL-3.0). Never deployed
# from this repository; production uses the canonical on-chain EntryPoint.
VENDORED = {
    "EntryPoint",
    "EntryPointSimulations",
    "SenderCreator",
    "StakeManager",
    "NonceManager",
}

LIMIT = 24576  # EIP-170 runtime code size limit


def sizes(profile):
    cmd = ["forge", "build", "--sizes", "--json"]
    if profile:
        cmd += ["--profile", profile]
    out = subprocess.run(cmd, capture_output=True, text=True)
    if out.returncode != 0 and not out.stdout.strip():
        sys.stderr.write(out.stderr)
        sys.exit(f"forge build failed for profile {profile or 'default'}")
    return json.loads(out.stdout)


def main():
    failures = []
    checked = 0
    for profile in (None, "dex"):
        for name, data in sizes(profile).items():
            if name in VENDORED:
                continue
            checked += 1
            runtime = data["runtime_size"]
            if runtime > LIMIT:
                failures.append(
                    f"{name} [{profile or 'default'}] runtime {runtime} B "
                    f"exceeds EIP-170 limit of {LIMIT} B"
                )

    if failures:
        print("Contract size check FAILED:")
        for f in failures:
            print("  " + f)
        return 1

    print(f"Contract size check passed ({checked} deployable contracts <= {LIMIT} B)")
    print(f"Excluded vendored reference contracts (never deployed): {sorted(VENDORED)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
