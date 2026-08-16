import SwiftUI
import Combine

// MARK: - Theme Manager

class ThemeManager: ObservableObject {
    static let shared = ThemeManager()

    @Published var isDarkMode: Bool {
        didSet {
            UserDefaults.standard.set(isDarkMode, forKey: "isDarkMode")
        }
    }

    /// White-label primary/secondary colors, overlayed onto the theme accent
    /// slot. When no WL branding is present these are the TigerWallet defaults
    /// (backward compatible). Updated whenever BrandingConfig refreshes.
    @Published private(set) var primaryColor: Color
    @Published private(set) var secondaryColor: Color

    private var brandingObserver: AnyCancellable?

    private init() {
        self.isDarkMode = UserDefaults.standard.bool(forKey: "isDarkMode")
        let b = BrandingConfig.shared.branding
        self.primaryColor = Color(hex: b.primaryColor) ?? Color(hex: Branding.defaults.primaryColor)!
        self.secondaryColor = Color(hex: b.secondaryColor) ?? Color(hex: Branding.defaults.secondaryColor)!
        self.brandingObserver = BrandingConfig.shared.$branding
            .receive(on: DispatchQueue.main)
            .sink { [weak self] b in
                self?.primaryColor = Color(hex: b.primaryColor) ?? Color(hex: Branding.defaults.primaryColor)!
                self?.secondaryColor = Color(hex: b.secondaryColor) ?? Color(hex: Branding.defaults.secondaryColor)!
            }
    }

    func toggleTheme() {
        isDarkMode.toggle()
    }
}

// MARK: - Network Manager

class NetworkManager: ObservableObject {
    static let shared = NetworkManager()
    
    @Published var selectedNetwork: BlockchainNetwork = .ethereum
    @Published var networks: [BlockchainNetwork] = []
    
    private init() {
        loadNetworks()
    }
    
    private func loadNetworks() {
        networks = [
            BlockchainNetwork(id: "ethereum", name: "Ethereum", symbol: "ETH", chainId: 1, isEVM: true),
            BlockchainNetwork(id: "bsc", name: "BNB Chain", symbol: "BNB", chainId: 56, isEVM: true),
            BlockchainNetwork(id: "polygon", name: "Polygon", symbol: "MATIC", chainId: 137, isEVM: true),
            BlockchainNetwork(id: "arbitrum", name: "Arbitrum", symbol: "ETH", chainId: 42161, isEVM: true),
            BlockchainNetwork(id: "optimism", name: "Optimism", symbol: "ETH", chainId: 10, isEVM: true),
            BlockchainNetwork(id: "avalanche", name: "Avalanche", symbol: "AVAX", chainId: 43114, isEVM: true),
            BlockchainNetwork(id: "solana", name: "Solana", symbol: "SOL", chainId: 0, isEVM: false),
            BlockchainNetwork(id: "tron", name: "Tron", symbol: "TRX", chainId: 195, isEVM: false),
            BlockchainNetwork(id: "bitcoin", name: "Bitcoin", symbol: "BTC", chainId: 0, isEVM: false),
        ]
    }
}

struct BlockchainNetwork: Identifiable, Hashable {
    let id: String
    let name: String
    let symbol: String
    let chainId: Int64
    let isEVM: Bool
    
    static let ethereum = BlockchainNetwork(id: "ethereum", name: "Ethereum", symbol: "ETH", chainId: 1, isEVM: true)
}

// MARK: - Service Locator

class ServiceLocator {
    static let shared = ServiceLocator()
    
    var walletService: WalletService!
    var blockchainService: BlockchainService!
    var swapService: SwapService!
    var stakingService: StakingService!
    var nftService: NFTService!
    
    func register() {
        walletService = WalletService()
        blockchainService = BlockchainService()
        swapService = SwapService()
        stakingService = StakingService()
        nftService = NFTService()
    }
}
