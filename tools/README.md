# Registry tools

Scripts that generate and check the chain registries under `go/wallet_api`.
Run them from the repository root.

## Regenerating the EVM registry

```bash
python3 tools/fetch_chain_registry.py   # downloads tools/cache/chains.json
python3 tools/gen_evm_chains.py         # rewrites chains_evm_data.go
cd go/wallet_api && gofmt -w chains_evm_data.go
```

The source of truth is [ethereum-lists/chains](https://github.com/ethereum-lists/chains),
published as `chainid.network/chains.json`. The download is a separate step so
regeneration works offline once the file is cached.

Entries already present in the generated file are preserved byte for byte, so
hand-checked RPC alternates and derivation paths survive a regeneration. New
chains are appended from the upstream data.

Only mainnet chains are emitted. A chain is skipped when its name or short name
matches a testnet pattern, when the registry marks it deprecated, or when the
registry publishes no `https` RPC for it. Shipping an entry with no reachable
endpoint would present a chain as usable when it is not.

## Checking coverage

```bash
python3 tools/check_chain_coverage.py
```

Reports which of the chains listed in `tools/requested_all_chains.txt` are
present in the shipped registries and names the ones that are not.

## Verifying that the endpoints work

```bash
python3 tools/verify_chain_rpcs.py          # every endpoint in the registry
python3 tools/verify_chain_rpcs.py --new    # only entries added since the baseline
```

Posts `eth_chainId` to every primary endpoint and every alternate, and compares
the answer with the declared chain id. A chain id mismatch is a real fault and
fails the run. An unreachable endpoint is reported but does not fail the run:
public endpoints go down regularly, and `go/wallet_api/rpc_failover.go` retries
the alternates at runtime.

`--new` diffs against `tools/cache/chains_evm_data.baseline.go`, which records
the registry as it stood before the expansion, so newly added entries can be
checked without waiting for all 293 endpoints.
