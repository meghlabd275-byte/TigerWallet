import Foundation
import Combine

// MARK: - Base ViewModel

class BaseViewModel: ObservableObject {
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var isError = false
    
    func showError(_ message: String) {
        errorMessage = message
        isError = true
    }
    
    func clearError() {
        errorMessage = nil
        isError = false
    }
}

// MARK: - Dashboard ViewModel

class DashboardViewModel: BaseViewModel {
    @Published var analyticsData: AnalyticsData?
    @Published var recentActivity: [ActivityItem] = []
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadDashboard()
    }
    
    func loadDashboard() {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let data = try await apiService.getAnalyticsOverview()
                self.analyticsData = data
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func refresh() {
        loadDashboard()
    }
}

struct ActivityItem: Identifiable {
    let id = UUID()
    let type: ActivityType
    let message: String
    let timestamp: Date
}

enum ActivityType {
    case userVerified
    case transaction
    case kyc
    case token
    case suspicious
}

// MARK: - Users ViewModel

class UsersViewModel: BaseViewModel {
    @Published var users: [PlatformUser] = []
    @Published var selectedUser: PlatformUser?
    @Published var statusFilter: UserStatus?
    @Published var kycFilter: KYCStatus?
    @Published var searchQuery = ""
    @Published var currentPage = 1
    @Published var totalPages = 1
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadUsers()
    }
    
    func loadUsers(page: Int = 1) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let response = try await apiService.getUsers(
                    page: page,
                    status: statusFilter?.rawValue,
                    kycStatus: kycFilter?.rawValue,
                    search: searchQuery.isEmpty ? nil : searchQuery
                )
                self.users = response.data
                self.currentPage = response.pagination.page
                self.totalPages = response.pagination.totalPages
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func suspendUser(_ user: PlatformUser, reason: String) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.suspendUser(id: user.id, reason: reason)
                self.loadUsers(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func banUser(_ user: PlatformUser, reason: String) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.banUser(id: user.id, reason: reason)
                self.loadUsers(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func activateUser(_ user: PlatformUser) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.activateUser(id: user.id)
                self.loadUsers(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func search(_ query: String) {
        searchQuery = query
        loadUsers(page: 1)
    }
    
    func filterByStatus(_ status: UserStatus?) {
        statusFilter = status
        loadUsers(page: 1)
    }
    
    func filterByKYC(_ status: KYCStatus?) {
        kycFilter = status
        loadUsers(page: 1)
    }
    
    func nextPage() {
        if currentPage < totalPages {
            loadUsers(page: currentPage + 1)
        }
    }
    
    func previousPage() {
        if currentPage > 1 {
            loadUsers(page: currentPage - 1)
        }
    }
    
    func refresh() {
        loadUsers(page: currentPage)
    }
}

// MARK: - Transactions ViewModel

class TransactionsViewModel: BaseViewModel {
    @Published var transactions: [Transaction] = []
    @Published var selectedTransaction: Transaction?
    @Published var statusFilter: TransactionStatus?
    @Published var typeFilter: TransactionType?
    @Published var chainFilter: String?
    @Published var flaggedOnly = false
    @Published var currentPage = 1
    @Published var totalPages = 1
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadTransactions()
    }
    
    func loadTransactions(page: Int = 1) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let response = try await apiService.getTransactions(
                    page: page,
                    status: statusFilter?.rawValue,
                    type: typeFilter?.rawValue,
                    chain: chainFilter,
                    flagged: flaggedOnly ? true : nil
                )
                self.transactions = response.data
                self.currentPage = response.pagination.page
                self.totalPages = response.pagination.totalPages
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func flagTransaction(_ transaction: Transaction, reason: String) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.flagTransaction(id: transaction.id, reason: reason)
                self.loadTransactions(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func unflagTransaction(_ transaction: Transaction) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.unflagTransaction(id: transaction.id)
                self.loadTransactions(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func filterByStatus(_ status: TransactionStatus?) {
        statusFilter = status
        loadTransactions(page: 1)
    }
    
    func filterByType(_ type: TransactionType?) {
        typeFilter = type
        loadTransactions(page: 1)
    }
    
    func filterByChain(_ chain: String?) {
        chainFilter = chain
        loadTransactions(page: 1)
    }
    
    func toggleFlaggedOnly() {
        flaggedOnly.toggle()
        loadTransactions(page: 1)
    }
    
    func refresh() {
        loadTransactions(page: currentPage)
    }
}

// MARK: - KYC ViewModel

class KYCViewModel: BaseViewModel {
    @Published var applications: [KYCApplication] = []
    @Published var selectedApplication: KYCApplication?
    @Published var statusFilter: KYCApplicationStatus?
    @Published var levelFilter: Int?
    @Published var currentPage = 1
    @Published var totalPages = 1
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadApplications()
    }
    
    func loadApplications(page: Int = 1) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let response = try await apiService.getKYCApplications(
                    page: page,
                    status: statusFilter?.rawValue,
                    level: levelFilter
                )
                self.applications = response.data
                self.currentPage = response.pagination.page
                self.totalPages = response.pagination.totalPages
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func approveKYC(_ application: KYCApplication) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.approveKYC(id: application.id)
                self.loadApplications(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func rejectKYC(_ application: KYCApplication, reason: String) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.rejectKYC(id: application.id, reason: reason)
                self.loadApplications(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func filterByStatus(_ status: KYCApplicationStatus?) {
        statusFilter = status
        loadApplications(page: 1)
    }
    
    func filterByLevel(_ level: Int?) {
        levelFilter = level
        loadApplications(page: 1)
    }
    
    func refresh() {
        loadApplications(page: currentPage)
    }
}

// MARK: - Tokens ViewModel

class TokensViewModel: BaseViewModel {
    @Published var tokens: [Token] = []
    @Published var listingRequests: [TokenListingRequest] = []
    @Published var selectedToken: Token?
    @Published var chainFilter: String?
    @Published var activeOnly = false
    @Published var currentPage = 1
    @Published var totalPages = 1
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadTokens()
    }
    
    func loadTokens(page: Int = 1) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let response = try await apiService.getTokens(
                    page: page,
                    chain: chainFilter,
                    isActive: activeOnly ? true : nil
                )
                self.tokens = response.data
                self.currentPage = response.pagination.page
                self.totalPages = response.pagination.totalPages
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func loadListingRequests() {
        isLoading = true
        
        Task { @MainActor in
            do {
                let response = try await apiService.getTokenListings()
                self.listingRequests = response.data
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func activateToken(_ token: Token) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.activateToken(id: token.id)
                self.loadTokens(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func deactivateToken(_ token: Token) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.deactivateToken(id: token.id)
                self.loadTokens(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func verifyToken(_ token: Token) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.verifyToken(id: token.id)
                self.loadTokens(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func approveListing(_ request: TokenListingRequest) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.approveTokenListing(id: request.id)
                self.loadListingRequests()
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func rejectListing(_ request: TokenListingRequest, reason: String) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.rejectTokenListing(id: request.id, reason: reason)
                self.loadListingRequests()
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func filterByChain(_ chain: String?) {
        chainFilter = chain
        loadTokens(page: 1)
    }
    
    func toggleActiveOnly() {
        activeOnly.toggle()
        loadTokens(page: 1)
    }
    
    func refresh() {
        loadTokens(page: currentPage)
    }
}

// MARK: - Withdrawals ViewModel

class WithdrawalsViewModel: BaseViewModel {
    @Published var withdrawals: [WithdrawalRequest] = []
    @Published var selectedWithdrawal: WithdrawalRequest?
    @Published var statusFilter: WithdrawalStatus?
    @Published var tokenFilter: String?
    @Published var chainFilter: String?
    @Published var currentPage = 1
    @Published var totalPages = 1
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadWithdrawals()
    }
    
    func loadWithdrawals(page: Int = 1) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let response = try await apiService.getWithdrawals(
                    page: page,
                    status: statusFilter?.rawValue,
                    token: tokenFilter,
                    chain: chainFilter
                )
                self.withdrawals = response.data
                self.currentPage = response.pagination.page
                self.totalPages = response.pagination.totalPages
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func approveWithdrawal(_ withdrawal: WithdrawalRequest) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.approveWithdrawal(id: withdrawal.id)
                self.loadWithdrawals(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func rejectWithdrawal(_ withdrawal: WithdrawalRequest, reason: String) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.rejectWithdrawal(id: withdrawal.id, reason: reason)
                self.loadWithdrawals(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func processWithdrawal(_ withdrawal: WithdrawalRequest, txHash: String) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.processWithdrawal(id: withdrawal.id, txHash: txHash)
                self.loadWithdrawals(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func filterByStatus(_ status: WithdrawalStatus?) {
        statusFilter = status
        loadWithdrawals(page: 1)
    }
    
    func filterByToken(_ token: String?) {
        tokenFilter = token
        loadWithdrawals(page: 1)
    }
    
    func filterByChain(_ chain: String?) {
        chainFilter = chain
        loadWithdrawals(page: 1)
    }
    
    func refresh() {
        loadWithdrawals(page: currentPage)
    }
}

// MARK: - White Label ViewModel

class WhiteLabelsViewModel: BaseViewModel {
    @Published var whiteLabels: [WhiteLabel] = []
    @Published var selectedWhiteLabel: WhiteLabel?
    @Published var statusFilter: WhiteLabelStatus?
    @Published var searchQuery = ""
    @Published var currentPage = 1
    @Published var totalPages = 1
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadWhiteLabels()
    }
    
    func loadWhiteLabels(page: Int = 1) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let response = try await apiService.getWhiteLabels(
                    page: page,
                    status: statusFilter?.rawValue,
                    search: searchQuery.isEmpty ? nil : searchQuery
                )
                self.whiteLabels = response.data
                self.currentPage = response.pagination.page
                self.totalPages = response.pagination.totalPages
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func activateWhiteLabel(_ whiteLabel: WhiteLabel) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.activateWhiteLabel(id: whiteLabel.id)
                self.loadWhiteLabels(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func suspendWhiteLabel(_ whiteLabel: WhiteLabel) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.suspendWhiteLabel(id: whiteLabel.id)
                self.loadWhiteLabels(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func filterByStatus(_ status: WhiteLabelStatus?) {
        statusFilter = status
        loadWhiteLabels(page: 1)
    }
    
    func search(_ query: String) {
        searchQuery = query
        loadWhiteLabels(page: 1)
    }
    
    func refresh() {
        loadWhiteLabels(page: currentPage)
    }
}

// MARK: - Bots ViewModel

class BotsViewModel: BaseViewModel {
    @Published var bots: [BotInstance] = []
    @Published var selectedBot: BotInstance?
    @Published var statusFilter: BotStatus?
    @Published var typeFilter: String?
    @Published var currentPage = 1
    @Published var totalPages = 1
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadBots()
    }
    
    func loadBots(page: Int = 1) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let response = try await apiService.getBots(
                    page: page,
                    status: statusFilter?.rawValue,
                    botType: typeFilter
                )
                self.bots = response.data
                self.currentPage = response.pagination.page
                self.totalPages = response.pagination.totalPages
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func startBot(_ bot: BotInstance) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.startBot(id: bot.id)
                self.loadBots(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func stopBot(_ bot: BotInstance) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.stopBot(id: bot.id)
                self.loadBots(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func pauseBot(_ bot: BotInstance) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.pauseBot(id: bot.id)
                self.loadBots(page: currentPage)
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func filterByStatus(_ status: BotStatus?) {
        statusFilter = status
        loadBots(page: 1)
    }
    
    func filterByType(_ type: String?) {
        typeFilter = type
        loadBots(page: 1)
    }
    
    func refresh() {
        loadBots(page: currentPage)
    }
}

// MARK: - System ViewModel

class SystemViewModel: BaseViewModel {
    @Published var services: [SystemStatus] = []
    @Published var databases: [SystemStatus] = []
    @Published var networks: [SystemStatus] = []
    @Published var health: SystemHealth?
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadSystemStatus()
    }
    
    func loadSystemStatus() {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let status = try await apiService.getSystemStatus()
                self.services = status.services
                self.databases = status.databases
                self.networks = status.networks
                
                let health = try await apiService.getSystemHealth()
                self.health = health
                
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func refresh() {
        loadSystemStatus()
    }
}

// MARK: - Fees ViewModel

class FeesViewModel: BaseViewModel {
    @Published var feeConfigs: [FeeConfig] = []
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadFeeConfigs()
    }
    
    func loadFeeConfigs() {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let configs = try await apiService.getFeeConfigs()
                self.feeConfigs = configs
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func updateFeeConfig(_ config: FeeConfig, updatedData: FeeConfig) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                _ = try await apiService.updateFeeConfig(id: config.id, request: UpdateFeeConfigRequest(
                    feeAmountUsd: updatedData.feeAmountUsd,
                    feePercentage: updatedData.feePercentage,
                    minFeeUsd: updatedData.minFeeUsd,
                    maxFeeUsd: updatedData.maxFeeUsd
                ))
                self.loadFeeConfigs()
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func refresh() {
        loadFeeConfigs()
    }
}

// MARK: - Blockchains ViewModel

class BlockchainsViewModel: BaseViewModel {
    @Published var blockchains: [Blockchain] = []
    
    private let apiService = AdminAPIService.shared
    
    override init() {
        super.init()
        loadBlockchains()
    }
    
    func loadBlockchains() {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                let chains = try await apiService.getBlockchains()
                self.blockchains = chains
                self.isLoading = false
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func activateBlockchain(_ blockchain: Blockchain) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.activateBlockchain(id: blockchain.id)
                self.loadBlockchains()
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func deactivateBlockchain(_ blockchain: Blockchain) {
        isLoading = true
        clearError()
        
        Task { @MainActor in
            do {
                try await apiService.deactivateBlockchain(id: blockchain.id)
                self.loadBlockchains()
            } catch {
                self.showError(error.localizedDescription)
                self.isLoading = false
            }
        }
    }
    
    func refresh() {
        loadBlockchains()
    }
}
