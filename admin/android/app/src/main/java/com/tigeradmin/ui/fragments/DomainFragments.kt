package com.tigeradmin.ui.fragments

import com.tigeradmin.TigerAdminApplication
import com.tigeradmin.data.model.*
import com.tigeradmin.data.repository.*

/**
 * Concrete admin domain fragments for the 12 families backed by the admin/go
 * backend (port 9093). Each performs real API calls via the Retrofit-based
 * repositories and renders loading / error / empty states with light/dark theme.
 */
class FuturesFragment : DomainListFragment<FuturesRecord>() {
    private val repo = FuturesRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Futures"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: FuturesRecord) = item.name ?: item.symbol ?: "Futures #${item.id}"
    override fun rowSubtitle(item: FuturesRecord) =
        listOfNotNull(item.symbol, item.leverage?.let { "${it}x" }, item.margin).joinToString(" · ")
    override fun rowStatus(item: FuturesRecord) = item.status
    override fun primaryActionLabel() = "Toggle"
    override fun onPrimaryAction(item: FuturesRecord) {
        val next = if (item.status.equals("active", true)) "paused" else "active"
        viewLifecycleOwner.lifecycleScope.launch {
            repo.updateStatus(item.id, next).onSuccess { toast("Status set to $next"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class OptionsFragment : DomainListFragment<OptionsRecord>() {
    private val repo = OptionsRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Options"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: OptionsRecord) = item.name ?: item.symbol ?: "Options #${item.id}"
    override fun rowSubtitle(item: OptionsRecord) =
        listOfNotNull(item.symbol, item.strike, item.expiry).joinToString(" · ")
    override fun rowStatus(item: OptionsRecord) = item.status
    override fun primaryActionLabel() = "Toggle"
    override fun onPrimaryAction(item: OptionsRecord) {
        val next = if (item.status.equals("active", true)) "paused" else "active"
        viewLifecycleOwner.lifecycleScope.launch {
            repo.updateStatus(item.id, next).onSuccess { toast("Status set to $next"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class CopyTradingFragment : DomainListFragment<CopyTradingRecord>() {
    private val repo = CopyTradingRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Copy Trading"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: CopyTradingRecord) = item.name ?: "Strategy #${item.id}"
    override fun rowSubtitle(item: CopyTradingRecord) =
        listOfNotNull(item.trader, item.followers?.let { "$it followers" }).joinToString(" · ")
    override fun rowStatus(item: CopyTradingRecord) = item.status
    override fun primaryActionLabel() = "Toggle"
    override fun onPrimaryAction(item: CopyTradingRecord) {
        val next = if (item.status.equals("active", true)) "paused" else "active"
        viewLifecycleOwner.lifecycleScope.launch {
            repo.updateStatus(item.id, next).onSuccess { toast("Status set to $next"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class ConvertFragment : DomainListFragment<ConvertRecord>() {
    private val repo = ConvertRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Convert"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: ConvertRecord) =
        "${item.fromAsset ?: "?"} → ${item.toAsset ?: "?"}"
    override fun rowSubtitle(item: ConvertRecord) =
        listOfNotNull(item.amount, item.rate?.let { "rate $it" }).joinToString(" · ")
    override fun rowStatus(item: ConvertRecord) = item.status
    override fun primaryActionLabel() = "Toggle"
    override fun onPrimaryAction(item: ConvertRecord) {
        val next = if (item.status.equals("active", true)) "paused" else "active"
        viewLifecycleOwner.lifecycleScope.launch {
            repo.updateStatus(item.id, next).onSuccess { toast("Status set to $next"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class OnRampFragment : DomainListFragment<OnRampRecord>() {
    private val repo = OnRampRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "On-Ramp"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: OnRampRecord) = "${item.asset ?: "Asset"} ${item.amount ?: ""}"
    override fun rowSubtitle(item: OnRampRecord) =
        listOfNotNull(item.user, item.provider).joinToString(" · ")
    override fun rowStatus(item: OnRampRecord) = item.status
    override fun primaryActionLabel() = "Approve"
    override fun onPrimaryAction(item: OnRampRecord) {
        viewLifecycleOwner.lifecycleScope.launch {
            repo.approve(item.id).onSuccess { toast("Approved"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
    override fun secondaryActionLabel() = "Reject"
    override fun onSecondaryAction(item: OnRampRecord) {
        viewLifecycleOwner.lifecycleScope.launch {
            repo.reject(item.id, "Rejected by admin").onSuccess { toast("Rejected"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class OffRampFragment : DomainListFragment<OffRampRecord>() {
    private val repo = OffRampRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Off-Ramp"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: OffRampRecord) = "${item.asset ?: "Asset"} ${item.amount ?: ""}"
    override fun rowSubtitle(item: OffRampRecord) =
        listOfNotNull(item.user, item.provider).joinToString(" · ")
    override fun rowStatus(item: OffRampRecord) = item.status
    override fun primaryActionLabel() = "Approve"
    override fun onPrimaryAction(item: OffRampRecord) {
        viewLifecycleOwner.lifecycleScope.launch {
            repo.approve(item.id).onSuccess { toast("Approved"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
    override fun secondaryActionLabel() = "Reject"
    override fun onSecondaryAction(item: OffRampRecord) {
        viewLifecycleOwner.lifecycleScope.launch {
            repo.reject(item.id, "Rejected by admin").onSuccess { toast("Rejected"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class P2PClientsFragment : DomainListFragment<P2PClientRecord>() {
    private val repo = P2PClientRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "P2P Clients"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: P2PClientRecord) = item.name ?: "Client #${item.id}"
    override fun rowSubtitle(item: P2PClientRecord) = item.email ?: ""
    override fun rowStatus(item: P2PClientRecord) = item.status
    override fun primaryActionLabel() = "Toggle"
    override fun onPrimaryAction(item: P2PClientRecord) {
        val next = if (item.status.equals("active", true)) "suspended" else "active"
        viewLifecycleOwner.lifecycleScope.launch {
            repo.updateStatus(item.id, next).onSuccess { toast("Status set to $next"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class P2PMerchantsFragment : DomainListFragment<P2PMerchantRecord>() {
    private val repo = P2PMerchantRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "P2P Merchants"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: P2PMerchantRecord) = item.name ?: "Merchant #${item.id}"
    override fun rowSubtitle(item: P2PMerchantRecord) =
        listOfNotNull(item.email, item.verified?.let { if (it) "verified" else "unverified" }).joinToString(" · ")
    override fun rowStatus(item: P2PMerchantRecord) = item.status
    override fun primaryActionLabel() = "Approve"
    override fun onPrimaryAction(item: P2PMerchantRecord) {
        viewLifecycleOwner.lifecycleScope.launch {
            repo.approve(item.id).onSuccess { toast("Approved"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
    override fun secondaryActionLabel() = "Reject"
    override fun onSecondaryAction(item: P2PMerchantRecord) {
        viewLifecycleOwner.lifecycleScope.launch {
            repo.reject(item.id, "Rejected by admin").onSuccess { toast("Rejected"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class PartnersFragment : DomainListFragment<PartnerRecord>() {
    private val repo = PartnerRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Partners"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: PartnerRecord) = item.name ?: "Partner #${item.id}"
    override fun rowSubtitle(item: PartnerRecord) = item.type ?: ""
    override fun rowStatus(item: PartnerRecord) = item.status
    override fun primaryActionLabel() = "Approve"
    override fun onPrimaryAction(item: PartnerRecord) {
        viewLifecycleOwner.lifecycleScope.launch {
            repo.approve(item.id).onSuccess { toast("Approved"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
    override fun secondaryActionLabel() = "Reject"
    override fun onSecondaryAction(item: PartnerRecord) {
        viewLifecycleOwner.lifecycleScope.launch {
            repo.reject(item.id, "Rejected by admin").onSuccess { toast("Rejected"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class RewardsFragment : DomainListFragment<RewardRecord>() {
    private val repo = RewardRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Rewards"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: RewardRecord) = item.name ?: "Reward #${item.id}"
    override fun rowSubtitle(item: RewardRecord) =
        listOfNotNull(item.type, item.amount).joinToString(" · ")
    override fun rowStatus(item: RewardRecord) = item.status
    override fun primaryActionLabel() = "Toggle"
    override fun onPrimaryAction(item: RewardRecord) {
        val next = if (item.status.equals("active", true)) "paused" else "active"
        viewLifecycleOwner.lifecycleScope.launch {
            repo.updateStatus(item.id, next).onSuccess { toast("Status set to $next"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class MarketingFragment : DomainListFragment<MarketingRecord>() {
    private val repo = MarketingRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Marketing"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: MarketingRecord) = item.name ?: "Campaign #${item.id}"
    override fun rowSubtitle(item: MarketingRecord) = item.campaign ?: ""
    override fun rowStatus(item: MarketingRecord) = item.status
    override fun primaryActionLabel() = "Toggle"
    override fun onPrimaryAction(item: MarketingRecord) {
        val next = if (item.status.equals("active", true)) "paused" else "active"
        viewLifecycleOwner.lifecycleScope.launch {
            repo.updateStatus(item.id, next).onSuccess { toast("Status set to $next"); reload() }
                .onFailure { toast("Failed: ${it.message}") }
        }
    }
}

class RolesFragment : DomainListFragment<RoleRecord>() {
    private val repo = RolesRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Roles"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: RoleRecord) = item.name
    override fun rowSubtitle(item: RoleRecord) =
        listOfNotNull(item.description, item.permissions?.joinToString(",")?.let { "perms: $it" }).joinToString(" · ")
    override fun rowStatus(item: RoleRecord) = null
}

class PermissionsFragment : DomainListFragment<PermissionRecord>() {
    private val repo = PermissionsRepository(TigerAdminApplication.instance!!.adminApiService)
    override fun fragmentTitle() = "Permissions"
    override suspend fun loadRecords() = repo.list().getOrThrow()
    override fun rowTitle(item: PermissionRecord) = item.name
    override fun rowSubtitle(item: PermissionRecord) =
        listOfNotNull(item.resource, item.action, item.description).joinToString(" · ")
    override fun rowStatus(item: PermissionRecord) = null
}
