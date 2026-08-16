//
//  AdminAPIService.swift
//  TigerAdmin - Admin API Service
//

import Foundation

class AdminAPIService {
    private let baseURL: String

    init(baseURL: String? = nil) {
        // admin/go backend (port 9093). ADMIN_API_URL env overrides when present.
        self.baseURL = baseURL
            ?? ProcessInfo.processInfo.environment["ADMIN_API_URL"]
            ?? "http://localhost:9093"
    }
    
    // MARK: - User Management APIs
    func getUsers(status: String? = nil, limit: Int = 50) async throws -> [User] {
        var endpoint = "/api/v1/admin/users?limit=\(limit)"
        if let status = status {
            endpoint += "&status=\(status)"
        }
        return try await request(endpoint: endpoint)
    }
    
    func getUser(id: String) async throws -> User {
        return try await request(endpoint: "/api/v1/admin/users/\(id)")
    }
    
    func updateUserKYC(userId: String, status: String) async throws -> User {
        let body = try JSONEncoder().encode(["kyc_status": status])
        return try await request(endpoint: "/api/v1/admin/users/\(userId)/kyc", method: "PUT", body: body)
    }
    
    func deleteUser(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/admin/users/\(id)", method: "DELETE")
    }
    
    // MARK: - Transaction APIs
    func getTransactions(status: String? = nil, limit: Int = 50) async throws -> [AdminTransaction] {
        var endpoint = "/api/v1/admin/transactions?limit=\(limit)"
        if let status = status {
            endpoint += "&status=\(status)"
        }
        return try await request(endpoint: endpoint)
    }
    
    func getTransaction(id: String) async throws -> AdminTransaction {
        return try await request(endpoint: "/api/v1/admin/transactions/\(id)")
    }
    
    func approveTransaction(id: String) async throws -> AdminTransaction {
        return try await request(endpoint: "/api/v1/admin/transactions/\(id)/approve", method: "POST")
    }
    
    func rejectTransaction(id: String, reason: String) async throws -> AdminTransaction {
        let body = try JSONEncoder().encode(["reason": reason])
        return try await request(endpoint: "/api/v1/admin/transactions/\(id)/reject", method: "POST", body: body)
    }
    
    func cancelTransaction(id: String) async throws -> AdminTransaction {
        return try await request(endpoint: "/api/v1/admin/transactions/\(id)/cancel", method: "POST")
    }
    
    // MARK: - Analytics APIs
    func getAnalytics() async throws -> AdminAnalytics {
        return try await request(endpoint: "/api/v1/admin/analytics")
    }
    
    func getUserAnalytics(period: String) async throws -> [AnalyticsData] {
        return try await request(endpoint: "/api/v1/admin/analytics/users?period=\(period)")
    }
    
    func getVolumeAnalytics(period: String) async throws -> [AnalyticsData] {
        return try await request(endpoint: "/api/v1/admin/analytics/volume?period=\(period)")
    }
    
    // MARK: - System APIs
    func getSystemStatus() async throws -> SystemStatus {
        return try await request(endpoint: "/api/v1/admin/system/status")
    }
    
    func getServiceStatus(service: String) async throws -> ServiceStatus {
        return try await request(endpoint: "/api/v1/admin/system/services/\(service)")
    }
    
    func restartService(service: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/admin/system/services/\(service)/restart", method: "POST")
    }
    
    // MARK: - Fee Configuration APIs
    func getFeeConfig() async throws -> FeeConfig {
        return try await request(endpoint: "/api/v1/admin/fees")
    }
    
    func updateFeeConfig(config: FeeConfig) async throws -> FeeConfig {
        let body = try JSONEncoder().encode(config)
        return try await request(endpoint: "/api/v1/admin/fees", method: "PUT", body: body)
    }
    
    // MARK: - Token Management APIs
    func getTokens() async throws -> [Token] {
        return try await request(endpoint: "/api/v1/admin/tokens")
    }
    
    func listToken(address: String, listing: Bool) async throws -> Token {
        let body = try JSONEncoder().encode(["listed": listing])
        return try await request(endpoint: "/api/v1/admin/tokens/\(address)/listing", method: "PUT", body: body)
    }
    
    // MARK: - Admin User APIs
    func getAdminUsers() async throws -> [AdminUser] {
        return try await request(endpoint: "/api/v1/admin/admins")
    }
    
    func createAdminUser(user: CreateAdminUserRequest) async throws -> AdminUser {
        let body = try JSONEncoder().encode(user)
        return try await request(endpoint: "/api/v1/admin/admins", method: "POST", body: body)
    }
    
    func updateAdminPermissions(adminId: String, permissions: [String]) async throws -> AdminUser {
        let body = try JSONEncoder().encode(["permissions": permissions])
        return try await request(endpoint: "/api/v1/admin/admins/\(adminId)/permissions", method: "PUT", body: body)
    }
    
    func deleteAdminUser(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/admin/admins/\(id)", method: "DELETE")
    }
    
    // MARK: - Domain APIs (admin/go backend, /api/v1/<domain>)
    // Each domain: CRUD + status (or approve/reject) against port 9093.

    func listFutures() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/futures")
    }
    func getFutures(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/futures/\(id)")
    }
    func createFutures(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/futures", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateFutures(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/futures/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteFutures(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/futures/\(id)", method: "DELETE")
    }
    func setFuturesStatus(id: String, status: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/futures/\(id)/status", method: "PUT", body: body)
    }

    func listOptions() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/options")
    }
    func getOptions(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/options/\(id)")
    }
    func createOptions(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/options", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateOptions(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/options/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteOptions(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/options/\(id)", method: "DELETE")
    }
    func setOptionsStatus(id: String, status: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/options/\(id)/status", method: "PUT", body: body)
    }

    func listCopyTrading() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/copy-trading")
    }
    func getCopyTrading(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/copy-trading/\(id)")
    }
    func createCopyTrading(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/copy-trading", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateCopyTrading(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/copy-trading/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteCopyTrading(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/copy-trading/\(id)", method: "DELETE")
    }
    func setCopyTradingStatus(id: String, status: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/copy-trading/\(id)/status", method: "PUT", body: body)
    }

    func listConvert() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/convert")
    }
    func getConvert(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/convert/\(id)")
    }
    func createConvert(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/convert", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateConvert(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/convert/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteConvert(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/convert/\(id)", method: "DELETE")
    }
    func setConvertStatus(id: String, status: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/convert/\(id)/status", method: "PUT", body: body)
    }

    func listOnRamp() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/onramp")
    }
    func getOnRamp(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/onramp/\(id)")
    }
    func createOnRamp(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/onramp", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateOnRamp(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/onramp/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteOnRamp(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/onramp/\(id)", method: "DELETE")
    }
    func approveOnRamp(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/onramp/\(id)/approve", method: "POST")
    }
    func rejectOnRamp(id: String, reason: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["reason": reason])
        return try await request(endpoint: "/api/v1/onramp/\(id)/reject", method: "POST", body: body)
    }

    func listOffRamp() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/offramp")
    }
    func getOffRamp(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/offramp/\(id)")
    }
    func createOffRamp(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/offramp", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateOffRamp(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/offramp/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteOffRamp(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/offramp/\(id)", method: "DELETE")
    }
    func approveOffRamp(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/offramp/\(id)/approve", method: "POST")
    }
    func rejectOffRamp(id: String, reason: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["reason": reason])
        return try await request(endpoint: "/api/v1/offramp/\(id)/reject", method: "POST", body: body)
    }

    func listP2PClients() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/p2p-clients")
    }
    func getP2PClient(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/p2p-clients/\(id)")
    }
    func createP2PClient(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/p2p-clients", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateP2PClient(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/p2p-clients/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteP2PClient(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/p2p-clients/\(id)", method: "DELETE")
    }
    func setP2PClientStatus(id: String, status: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/p2p-clients/\(id)/status", method: "PUT", body: body)
    }

    func listP2PMerchants() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/p2p-merchants")
    }
    func getP2PMerchant(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/p2p-merchants/\(id)")
    }
    func createP2PMerchant(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/p2p-merchants", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateP2PMerchant(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/p2p-merchants/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    // p2p-merchants has no delete / status on admin/go — expose approve/reject + transactions.
    func approveP2PMerchant(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/p2p-merchants/\(id)/approve", method: "POST")
    }
    func rejectP2PMerchant(id: String, reason: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["reason": reason])
        return try await request(endpoint: "/api/v1/p2p-merchants/\(id)/reject", method: "POST", body: body)
    }
    func listP2PMerchantTransactions(id: String) async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/p2p-merchants/\(id)/transactions")
    }

    func listPartners() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/partners")
    }
    func getPartner(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/partners/\(id)")
    }
    func createPartner(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/partners", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updatePartner(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/partners/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deletePartner(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/partners/\(id)", method: "DELETE")
    }
    func setPartnerStatus(id: String, status: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/partners/\(id)/status", method: "PUT", body: body)
    }
    func approvePartner(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/partners/\(id)/approve", method: "POST")
    }
    func rejectPartner(id: String, reason: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["reason": reason])
        return try await request(endpoint: "/api/v1/partners/\(id)/reject", method: "POST", body: body)
    }

    func listRewards() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/rewards")
    }
    func getReward(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/rewards/\(id)")
    }
    func createReward(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/rewards", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateReward(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/rewards/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteReward(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/rewards/\(id)", method: "DELETE")
    }
    func setRewardStatus(id: String, status: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/rewards/\(id)/status", method: "PUT", body: body)
    }

    func listMarketing() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/marketing")
    }
    func getMarketing(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/marketing/\(id)")
    }
    func createMarketing(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/marketing", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateMarketing(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/marketing/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteMarketing(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/marketing/\(id)", method: "DELETE")
    }
    func setMarketingStatus(id: String, status: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/marketing/\(id)/status", method: "PUT", body: body)
    }

    // MARK: - RBAC (Roles & Permissions)
    func listRoles() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/roles")
    }
    func getRole(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/roles/\(id)")
    }
    func createRole(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/roles", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updateRole(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/roles/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deleteRole(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/roles/\(id)", method: "DELETE")
    }
    func listPermissions() async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/permissions")
    }
    func getPermission(id: String) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/permissions/\(id)")
    }
    func createPermission(_ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/permissions", method: "POST", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func updatePermission(id: String, _ payload: [String: Any]) async throws -> DomainRecord {
        try await request(endpoint: "/api/v1/permissions/\(id)", method: "PUT", body: try JSONSerialization.data(withJSONObject: payload))
    }
    func deletePermission(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/permissions/\(id)", method: "DELETE")
    }
    func getAdminRoles(adminId: String) async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/admins/\(adminId)/roles")
    }
    func assignRole(adminId: String, roleId: String) async throws -> DomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["role_id": roleId])
        return try await request(endpoint: "/api/v1/admins/\(adminId)/roles", method: "POST", body: body)
    }
    func revokeRole(adminId: String, roleId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/admins/\(adminId)/roles/\(roleId)", method: "DELETE")
    }
    func getAdminPermissions(adminId: String) async throws -> [DomainRecord] {
        try await request(endpoint: "/api/v1/admins/\(adminId)/permissions")
    }

    // MARK: - Bots (/api/v1/bots) + tiers + stats
    func listBots() async throws -> [BotDomainRecord] {
        try await request(endpoint: "/api/v1/bots")
    }
    func getBot(id: String) async throws -> BotDomainRecord {
        try await request(endpoint: "/api/v1/bots/\(id)")
    }
    func createBot(_ payload: BotDomainRequest) async throws -> BotDomainRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/bots", method: "POST", body: body)
    }
    func updateBot(id: String, _ payload: BotDomainRequest) async throws -> BotDomainRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/bots/\(id)", method: "PUT", body: body)
    }
    func deleteBot(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/bots/\(id)", method: "DELETE")
    }
    func setBotStatus(id: String, status: String) async throws -> BotDomainRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/bots/\(id)/status", method: "PUT", body: body)
    }
    func getBotStats() async throws -> DomainStats {
        try await request(endpoint: "/api/v1/bots/stats")
    }
    func getBotTiers(id: String) async throws -> [BotTierRecord] {
        try await request(endpoint: "/api/v1/bots/\(id)/tiers")
    }
    func createBotTier(id: String, _ payload: BotTierRequest) async throws -> BotTierRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/bots/\(id)/tiers", method: "POST", body: body)
    }
    func updateBotTier(id: String, tierId: String, _ payload: BotTierRequest) async throws -> BotTierRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/bots/\(id)/tiers/\(tierId)", method: "PUT", body: body)
    }
    func deleteBotTier(id: String, tierId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/bots/\(id)/tiers/\(tierId)", method: "DELETE")
    }

    // MARK: - Bots Clients (/api/v1/bots-clients)
    func listBotsClients() async throws -> [BotsClientRecord] {
        try await request(endpoint: "/api/v1/bots-clients")
    }
    func getBotsClient(id: String) async throws -> BotsClientRecord {
        try await request(endpoint: "/api/v1/bots-clients/\(id)")
    }
    func createBotsClient(_ payload: BotsClientRequest) async throws -> BotsClientRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/bots-clients", method: "POST", body: body)
    }
    func updateBotsClient(id: String, _ payload: BotsClientRequest) async throws -> BotsClientRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/bots-clients/\(id)", method: "PUT", body: body)
    }
    func deleteBotsClient(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/bots-clients/\(id)", method: "DELETE")
    }
    func setBotsClientStatus(id: String, status: String) async throws -> BotsClientRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/bots-clients/\(id)/status", method: "PUT", body: body)
    }

    // MARK: - Project Teams (/api/v1/project-teams) + members
    func listProjectTeams() async throws -> [ProjectTeamRecord] {
        try await request(endpoint: "/api/v1/project-teams")
    }
    func getProjectTeam(id: String) async throws -> ProjectTeamRecord {
        try await request(endpoint: "/api/v1/project-teams/\(id)")
    }
    func createProjectTeam(_ payload: ProjectTeamRequest) async throws -> ProjectTeamRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/project-teams", method: "POST", body: body)
    }
    func updateProjectTeam(id: String, _ payload: ProjectTeamRequest) async throws -> ProjectTeamRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/project-teams/\(id)", method: "PUT", body: body)
    }
    func deleteProjectTeam(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/project-teams/\(id)", method: "DELETE")
    }
    func setProjectTeamStatus(id: String, status: String) async throws -> ProjectTeamRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/project-teams/\(id)/status", method: "PUT", body: body)
    }
    func getProjectTeamMembers(id: String) async throws -> [ProjectTeamMemberRecord] {
        try await request(endpoint: "/api/v1/project-teams/\(id)/members")
    }
    func addProjectTeamMember(id: String, _ payload: AddProjectTeamMemberRequest) async throws -> ProjectTeamMemberRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/project-teams/\(id)/members", method: "POST", body: body)
    }
    func removeProjectTeamMember(id: String, memberId: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/project-teams/\(id)/members/\(memberId)", method: "DELETE")
    }

    // MARK: - Liquidity Sources (/api/v1/liquidity-sources) + priority/health/stats
    func listLiquiditySources() async throws -> [LiquiditySourceRecord] {
        try await request(endpoint: "/api/v1/liquidity-sources")
    }
    func getLiquiditySource(id: String) async throws -> LiquiditySourceRecord {
        try await request(endpoint: "/api/v1/liquidity-sources/\(id)")
    }
    func createLiquiditySource(_ payload: LiquiditySourceRequest) async throws -> LiquiditySourceRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/liquidity-sources", method: "POST", body: body)
    }
    func updateLiquiditySource(id: String, _ payload: LiquiditySourceRequest) async throws -> LiquiditySourceRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/liquidity-sources/\(id)", method: "PUT", body: body)
    }
    func deleteLiquiditySource(id: String) async throws {
        let _: EmptyResponse = try await request(endpoint: "/api/v1/liquidity-sources/\(id)", method: "DELETE")
    }
    func setLiquiditySourceStatus(id: String, status: String) async throws -> LiquiditySourceRecord {
        let body = try JSONSerialization.data(withJSONObject: ["status": status])
        return try await request(endpoint: "/api/v1/liquidity-sources/\(id)/status", method: "PUT", body: body)
    }
    func setLiquiditySourcePriority(id: String, _ payload: SetLiquiditySourcePriorityRequest) async throws -> LiquiditySourceRecord {
        let body = try JSONEncoder().encode(payload)
        return try await request(endpoint: "/api/v1/liquidity-sources/\(id)/priority", method: "PUT", body: body)
    }
    func liquiditySourceHealthCheck(id: String) async throws -> LiquiditySourceRecord {
        try await request(endpoint: "/api/v1/liquidity-sources/\(id)/health-check", method: "POST")
    }
    func getLiquiditySourceStats() async throws -> DomainStats {
        try await request(endpoint: "/api/v1/liquidity-sources/stats")
    }

    // MARK: - Generic Request
    private func request<T: Decodable>(_ endpoint: String, method: String = "GET", body: Data? = nil) async throws -> T {
        guard let url = URL(string: "\(baseURL)\(endpoint)") else {
            throw APIError(code: "INVALID_URL", message: "Invalid URL")
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        // JWT bearer auth (real token from AuthService keychain; never mocked).
        if let token = AuthService.shared.token, !token.isEmpty {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        
        if let body = body {
            request.httpBody = body
        }
        
        let (data, response) = try await URLSession.shared.data(for: request)
        
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError(code: "INVALID_RESPONSE", message: "Invalid response")
        }
        
        guard (200...299).contains(httpResponse.statusCode) else {
            throw APIError(code: "HTTP_\(httpResponse.statusCode)", message: "HTTP error")
        }
        
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        
        return try decoder.decode(T.self, from: data)
    }
}

struct APIError: Error {
    let code: String
    let message: String
}

struct EmptyResponse: Codable {}

struct User: Codable {
    let id: String
    let email: String
    let name: String
    let kycStatus: String
    let createdAt: Date
}

struct AdminTransaction: Codable {
    let id: String
    let hash: String
    let from: String
    let to: String
    let amount: String
    let chain: String
    let type: String
    let status: String
    let createdAt: Date
    var processedAt: Date?
}

struct AdminAnalytics: Codable {
    let totalUsers: Int
    let totalVolumeUSD: Double
    let totalTransactions: Int
    let pendingKYC: Int
    let systemHealth: Double
    let activeUsers24h: Int
    let volume24h: Double
}

struct AnalyticsData: Codable {
    let date: Date
    let value: Double
    let change: Double
}

struct SystemStatus: Codable {
    let uptime: Double
    let services: [ServiceStatus]
    let database: DatabaseStatus
    let cache: CacheStatus
}

struct ServiceStatus: Codable {
    let name: String
    let status: String
    let uptime: Double
    let lastCheck: Date
}

struct DatabaseStatus: Codable {
    let postgres: ComponentStatus
    let redis: ComponentStatus
}

struct ComponentStatus: Codable {
    let status: String
    let connections: Int
    let latency: Double
}

struct CacheStatus: Codable {
    let status: String
    let hitRate: Double
    let memory: Int64
}

struct FeeConfig: Codable {
    var tradingFee: Double
    var withdrawalFee: Double
    var depositFee: Double
    var networkFee: Double
}

struct Token: Codable {
    let address: String
    let name: String
    let symbol: String
    let decimals: Int
    let isListed: Bool
    let marketCap: Double
    let volume24h: Double
}

struct AdminUser: Codable {
    let id: String
    let email: String
    let name: String
    let role: String
    let permissions: [String]
    let createdAt: Date
    var lastLogin: Date?
}

struct CreateAdminUserRequest: Codable {
    let email: String
    let name: String
    let role: String
    let permissions: [String]
}
