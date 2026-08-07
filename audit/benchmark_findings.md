# Verified Wallet Benchmark Findings

## Trust Wallet official developer documentation
Source: https://developer.trustwallet.com/developer

The documentation states that Trust Wallet is a self-custody multi-chain wallet supporting 100+ blockchains and millions of assets, with storing, sending, receiving, swapping, earning, and dApp connectivity. It documents a TrustConnect SDK for EVM, Solana, and Bitcoin; browser-extension and mobile WalletConnect integrations; deep linking; Barz ERC-4337 smart-wallet integration; asset and dApp listing workflows; staking validator listing; and Wallet Core as an open-source library.

## MetaMask official developer documentation
Source: https://docs.metamask.io/

The documentation states that MetaMask supports extension and mobile dApp connection across multichain, EVM, and Solana networks. It documents embedded wallets with social or custom authentication, smart accounts with programmable behaviors and granular permission sharing, agent wallets with mandatory transaction security, and Snaps for custom networks, account types, and APIs.

## Audit implications

The benchmark must test TigerWallet for: one shared signing/core contract across every client; native extension and mobile dApp connectivity; EVM, Solana, and Bitcoin provider interoperability; embedded-wallet and recovery options; ERC-4337 smart accounts with permissions; transaction security and simulation hooks; a modular extension/plugin API; dApp and asset listing workflows; staking support; and production-grade chain, market, token, and transaction data fetchers. Claims in repository-authored comparison documents are not accepted as evidence until the code builds and tests against live or contract-verified interfaces.

## Coinbase Wallet official SDK documentation
Source: https://www.coinbase.com/developer-platform/products/wallet-sdk

Coinbase documents dApp support across Solana and EVM-compatible chains including Avalanche, BNB Chain, Polygon, Optimism, and others; support for hundreds of thousands of assets and NFTs; fiat onramp coverage; native mobile integration; end-to-end encryption; batch processing; and compatibility with web3-react, web3modal, Web3-Onboard, and wagmi. The official page links to the open-source Coinbase Wallet SDK.

## OKX Wallet official Web3 page
Source: https://web3.okx.com/

The official OKX Web3 landing page is a benchmark source for a self-custody, multi-chain wallet integrating on-chain applications, DEX swaps, staking, and cross-network asset management. The page was dynamically rendered in the browser and requires direct documentation/API verification before treating individual feature claims as implementation requirements.

## Additional benchmark set

The top-ten benchmark set for this audit is: Trust Wallet, MetaMask, Coinbase Wallet, OKX Wallet, Bitget Wallet, Phantom, Ledger Live, Trezor Suite, Exodus, and Crypto.com Onchain Wallet. This set intentionally covers software multi-chain wallets, browser/mobile providers, smart-account platforms, and hardware-wallet companions rather than treating hardware devices as interchangeable with self-custody software wallets.
