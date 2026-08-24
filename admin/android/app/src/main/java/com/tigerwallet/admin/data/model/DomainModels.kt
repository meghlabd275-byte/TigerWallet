package com.tigerwallet.admin.data.model

import com.google.gson.annotations.SerializedName

/**
 * Domain models for the 12 admin families driven by the admin/go backend (port 9093).
 * These back the futures / options / copy-trading / convert / onramp / offramp /
 * p2p-clients / p2p-merchants / partners / rewards / marketing / roles admin screens.
 */

data class StatusUpdateRequest(
    @SerializedName("status") val status: String
)

data class RejectRequest(
    @SerializedName("reason") val reason: String
)

data class AssignRoleRequest(
    @SerializedName("roleId") val roleId: String
)

data class FuturesRecord(
    @SerializedName("id") val id: String,
    @SerializedName("symbol") val symbol: String? = null,
    @SerializedName("name") val name: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("leverage") val leverage: Int? = null,
    @SerializedName("margin") val margin: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class FuturesRequest(
    @SerializedName("symbol") val symbol: String,
    @SerializedName("name") val name: String,
    @SerializedName("leverage") val leverage: Int? = null,
    @SerializedName("margin") val margin: String? = null
)

data class OptionsRecord(
    @SerializedName("id") val id: String,
    @SerializedName("symbol") val symbol: String? = null,
    @SerializedName("name") val name: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("strike") val strike: String? = null,
    @SerializedName("expiry") val expiry: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class OptionsRequest(
    @SerializedName("symbol") val symbol: String,
    @SerializedName("name") val name: String,
    @SerializedName("strike") val strike: String? = null,
    @SerializedName("expiry") val expiry: String? = null
)

data class CopyTradingRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("trader") val trader: String? = null,
    @SerializedName("followers") val followers: Int? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class CopyTradingRequest(
    @SerializedName("name") val name: String,
    @SerializedName("trader") val trader: String? = null
)

data class ConvertRecord(
    @SerializedName("id") val id: String,
    @SerializedName("from_asset") val fromAsset: String? = null,
    @SerializedName("to_asset") val toAsset: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("amount") val amount: String? = null,
    @SerializedName("rate") val rate: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class ConvertRequest(
    @SerializedName("from_asset") val fromAsset: String,
    @SerializedName("to_asset") val toAsset: String,
    @SerializedName("amount") val amount: String? = null,
    @SerializedName("rate") val rate: String? = null
)

data class OnRampRecord(
    @SerializedName("id") val id: String,
    @SerializedName("user") val user: String? = null,
    @SerializedName("asset") val asset: String? = null,
    @SerializedName("amount") val amount: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("provider") val provider: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class OnRampRequest(
    @SerializedName("asset") val asset: String,
    @SerializedName("amount") val amount: String,
    @SerializedName("provider") val provider: String? = null
)

data class OffRampRecord(
    @SerializedName("id") val id: String,
    @SerializedName("user") val user: String? = null,
    @SerializedName("asset") val asset: String? = null,
    @SerializedName("amount") val amount: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("provider") val provider: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class OffRampRequest(
    @SerializedName("asset") val asset: String,
    @SerializedName("amount") val amount: String,
    @SerializedName("provider") val provider: String? = null
)

data class P2PClientRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("email") val email: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class P2PClientRequest(
    @SerializedName("name") val name: String,
    @SerializedName("email") val email: String? = null
)

data class P2PMerchantRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("email") val email: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("verified") val verified: Boolean? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class P2PMerchantRequest(
    @SerializedName("name") val name: String,
    @SerializedName("email") val email: String? = null
)

data class PartnerRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("type") val type: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("approved") val approved: Boolean? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class PartnerRequest(
    @SerializedName("name") val name: String,
    @SerializedName("type") val type: String? = null
)

data class RewardRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("type") val type: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("amount") val amount: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class RewardRequest(
    @SerializedName("name") val name: String,
    @SerializedName("type") val type: String? = null,
    @SerializedName("amount") val amount: String? = null
)

data class MarketingRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("campaign") val campaign: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class MarketingRequest(
    @SerializedName("name") val name: String,
    @SerializedName("campaign") val campaign: String? = null
)

data class RoleRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
    @SerializedName("description") val description: String? = null,
    @SerializedName("permissions") val permissions: List<String>? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class RoleRequest(
    @SerializedName("name") val name: String,
    @SerializedName("description") val description: String? = null,
    @SerializedName("permissions") val permissions: List<String>? = null
)

data class PermissionRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String,
    @SerializedName("description") val description: String? = null,
    @SerializedName("resource") val resource: String? = null,
    @SerializedName("action") val action: String? = null,
    @SerializedName("created_at") val createdAt: String? = null
)

data class PermissionRequest(
    @SerializedName("name") val name: String,
    @SerializedName("description") val description: String? = null,
    @SerializedName("resource") val resource: String? = null,
    @SerializedName("action") val action: String? = null
)

// ============================================================================
// New admin domains (bots, bots-clients, project-teams, liquidity-sources)
// backed by the admin/go service on port 9093. Each domain exposes CRUD +
// setStatus, plus the domain-specific sub-resources documented in
// admin/cpp/include/admin_bots.hpp.
// ============================================================================

// ---------- Bots ----------
data class BotRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("bot_type") val botType: String? = null,
    @SerializedName("user_id") val userId: String? = null,
    @SerializedName("leverage") val leverage: Int? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class BotRequest(
    @SerializedName("name") val name: String,
    @SerializedName("bot_type") val botType: String? = null,
    @SerializedName("leverage") val leverage: Int? = null
)

data class BotTierRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("level") val level: Int? = null,
    @SerializedName("status") val status: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class BotTierRequest(
    @SerializedName("name") val name: String,
    @SerializedName("level") val level: Int? = null,
    @SerializedName("status") val status: String? = null
)

// ---------- Bots Clients ----------
data class BotsClientRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("email") val email: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("client_id") val clientId: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class BotsClientRequest(
    @SerializedName("name") val name: String,
    @SerializedName("email") val email: String? = null,
    @SerializedName("client_id") val clientId: String? = null
)

// ---------- Project Teams ----------
data class ProjectTeamRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("description") val description: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class ProjectTeamRequest(
    @SerializedName("name") val name: String,
    @SerializedName("description") val description: String? = null
)

data class ProjectTeamMemberRecord(
    @SerializedName("id") val id: String,
    @SerializedName("team_id") val teamId: String? = null,
    @SerializedName("user_id") val userId: String? = null,
    @SerializedName("email") val email: String? = null,
    @SerializedName("role") val role: String? = null,
    @SerializedName("created_at") val createdAt: String? = null
)

data class AddProjectTeamMemberRequest(
    @SerializedName("user_id") val userId: String,
    @SerializedName("role") val role: String? = null
)

// ---------- Liquidity Sources ----------
data class LiquiditySourceRecord(
    @SerializedName("id") val id: String,
    @SerializedName("name") val name: String? = null,
    @SerializedName("status") val status: String,
    @SerializedName("type") val type: String? = null,
    @SerializedName("priority") val priority: Int? = null,
    @SerializedName("chain") val chain: String? = null,
    @SerializedName("created_at") val createdAt: String? = null,
    @SerializedName("updated_at") val updatedAt: String? = null
)

data class LiquiditySourceRequest(
    @SerializedName("name") val name: String,
    @SerializedName("type") val type: String? = null,
    @SerializedName("chain") val chain: String? = null,
    @SerializedName("priority") val priority: Int? = null
)

data class SetLiquiditySourcePriorityRequest(
    @SerializedName("priority") val priority: Int
)

// Generic stats wrapper: stats payloads vary per domain, so decode loosely.
data class DomainStatsResponse(
    @SerializedName("total") val total: Long? = null,
    @SerializedName("active") val active: Long? = null,
    @SerializedName("inactive") val inactive: Long? = null,
    @SerializedName("healthy") val healthy: Long? = null,
    @SerializedName("unhealthy") val unhealthy: Long? = null,
    @SerializedName("total_volume") val totalVolume: String? = null,
    @SerializedName("total_pnl") val totalPnl: String? = null
)

