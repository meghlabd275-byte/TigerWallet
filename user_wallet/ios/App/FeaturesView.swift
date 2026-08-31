import SwiftUI

// Feature hub ("More" tab): navigation links to every feature view so all
// UserWallet functionality is reachable from the UI — mirrors the web sidebar,
// desktop nav, extension tabs, and Android FeaturesFragment.
struct FeaturesView: View {
    var body: some View {
        NavigationView {
            List {
                Section("Core") {
                    NavigationLink("Send") { SendView() }
                    NavigationLink("Receive") { ReceiveView() }
                    NavigationLink("Swap") { SwapView() }
                    NavigationLink("Bridge") { BridgeView() }
                    NavigationLink("NFTs") { NFTsView() }
                }
                Section("Earn") {
                    NavigationLink("Staking") { StakingView() }
                    NavigationLink("DeFi Hub") { DeFiView() }
                    NavigationLink("Launchpool") { LaunchpoolView() }
                    NavigationLink("Token Sales") { TokenSalesView() }
                }
                Section("Trade") {
                    NavigationLink("Trading") { TradingView() }
                    NavigationLink("Prediction Markets") { PredictionView() }
                    NavigationLink("Copy Trading") { CopyTradingView() }
                    NavigationLink("DAO Governance") { DAOView() }
                }
                Section("Services") {
                    NavigationLink("Fiat Ramp") { RampView() }
                    NavigationLink("Crypto Card") { CardsView() }
                    NavigationLink("P2P Trading") { P2PView() }
                    NavigationLink("Price Alerts") { PriceAlertsView() }
                    NavigationLink("ENS") { ENSView() }
                    NavigationLink("Security") { SecurityView() }
                    NavigationLink("Terminal") { TerminalView() }
                    NavigationLink("Fees") { FeesView() }
                }
                Section("Security & Account") {
                    NavigationLink("Approvals") { ApprovalsView() }
                    NavigationLink("Address Book") { AddressBookView() }
                    NavigationLink("Devices") { DevicesView() }
                    NavigationLink("KYC") { KycView() }
                    NavigationLink("Keystore") { KeystoreView() }
                    NavigationLink("Multisig") { MultisigView() }
                    NavigationLink("Non-EVM Chains") { NonEvmView() }
                    NavigationLink("dApps & WalletConnect") { DAppsView() }
                    NavigationLink("Backup") { BackupView() }
                }
            }
            .navigationTitle("All Features")
        }
    }
}
