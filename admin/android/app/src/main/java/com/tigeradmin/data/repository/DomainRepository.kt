package com.tigeradmin.data.repository

import com.tigeradmin.data.api.AdminApiService
import com.tigeradmin.data.model.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Generic CRUD + status/approve/reject repository for the admin domain families
 * driven by the admin/go backend (port 9093). Each repository wraps the Retrofit
 * AdminApiService calls and surfaces Result<T> with real loading/error states.
 */
abstract class DomainRepository<T, R>(protected val apiService: AdminApiService) {

    protected abstract suspend fun fetchList(): retrofit2.Response<List<T>>
    protected abstract suspend fun fetchOne(id: String): retrofit2.Response<T>
    protected abstract suspend fun createRecord(request: R): retrofit2.Response<T>
    protected abstract suspend fun updateRecord(id: String, request: R): retrofit2.Response<T>
    protected abstract suspend fun deleteRecord(id: String): retrofit2.Response<Unit>
    protected abstract suspend fun setStatus(id: String, status: String): retrofit2.Response<T>

    suspend fun list(): Result<List<T>> = withContext(Dispatchers.IO) {
        try {
            val response = fetchList()
            if (response.isSuccessful) Result.success(response.body() ?: emptyList())
            else Result.failure(Exception("Failed to load records (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun get(id: String): Result<T> = withContext(Dispatchers.IO) {
        try {
            val response = fetchOne(id)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to load record (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun create(request: R): Result<T> = withContext(Dispatchers.IO) {
        try {
            val response = createRecord(request)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to create record (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun update(id: String, request: R): Result<T> = withContext(Dispatchers.IO) {
        try {
            val response = updateRecord(id, request)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to update record (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun delete(id: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = deleteRecord(id)
            if (response.isSuccessful) Result.success(Unit)
            else Result.failure(Exception("Failed to delete record (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateStatus(id: String, status: String): Result<T> = withContext(Dispatchers.IO) {
        try {
            val response = setStatus(id, status)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to update status (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/** Approvable / rejectable repositories (onramp, offramp, p2p-merchants, partners). */
abstract class ApprovableRepository<T, R>(apiService: AdminApiService) : DomainRepository<T, R>(apiService) {

    protected abstract suspend fun approveRecord(id: String): retrofit2.Response<T>
    protected abstract suspend fun rejectRecord(id: String, reason: String): retrofit2.Response<T>

    suspend fun approve(id: String): Result<T> = withContext(Dispatchers.IO) {
        try {
            val response = approveRecord(id)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to approve (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun reject(id: String, reason: String): Result<T> = withContext(Dispatchers.IO) {
        try {
            val response = rejectRecord(id, reason)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to reject (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

class FuturesRepository(apiService: AdminApiService) : DomainRepository<FuturesRecord, FuturesRequest>(apiService) {
    override suspend fun fetchList() = apiService.getFutures()
    override suspend fun fetchOne(id: String) = apiService.getFuturesRecord(id)
    override suspend fun createRecord(request: FuturesRequest) = apiService.createFutures(request)
    override suspend fun updateRecord(id: String, request: FuturesRequest) = apiService.updateFutures(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteFutures(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setFuturesStatus(id, StatusUpdateRequest(status))
}

class OptionsRepository(apiService: AdminApiService) : DomainRepository<OptionsRecord, OptionsRequest>(apiService) {
    override suspend fun fetchList() = apiService.getOptions()
    override suspend fun fetchOne(id: String) = apiService.getOptionsRecord(id)
    override suspend fun createRecord(request: OptionsRequest) = apiService.createOptions(request)
    override suspend fun updateRecord(id: String, request: OptionsRequest) = apiService.updateOptions(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteOptions(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setOptionsStatus(id, StatusUpdateRequest(status))
}

class CopyTradingRepository(apiService: AdminApiService) : DomainRepository<CopyTradingRecord, CopyTradingRequest>(apiService) {
    override suspend fun fetchList() = apiService.getCopyTrading()
    override suspend fun fetchOne(id: String) = apiService.getCopyTradingRecord(id)
    override suspend fun createRecord(request: CopyTradingRequest) = apiService.createCopyTrading(request)
    override suspend fun updateRecord(id: String, request: CopyTradingRequest) = apiService.updateCopyTrading(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteCopyTrading(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setCopyTradingStatus(id, StatusUpdateRequest(status))
}

class ConvertRepository(apiService: AdminApiService) : DomainRepository<ConvertRecord, ConvertRequest>(apiService) {
    override suspend fun fetchList() = apiService.getConvert()
    override suspend fun fetchOne(id: String) = apiService.getConvertRecord(id)
    override suspend fun createRecord(request: ConvertRequest) = apiService.createConvert(request)
    override suspend fun updateRecord(id: String, request: ConvertRequest) = apiService.updateConvert(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteConvert(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setConvertStatus(id, StatusUpdateRequest(status))
}

class OnRampRepository(apiService: AdminApiService) : ApprovableRepository<OnRampRecord, OnRampRequest>(apiService) {
    override suspend fun fetchList() = apiService.getOnRamp()
    override suspend fun fetchOne(id: String) = apiService.getOnRampRecord(id)
    override suspend fun createRecord(request: OnRampRequest) = apiService.createOnRamp(request)
    override suspend fun updateRecord(id: String, request: OnRampRequest) = apiService.updateOnRamp(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteOnRamp(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setOnRampStatus(id, StatusUpdateRequest(status))
    override suspend fun approveRecord(id: String) = apiService.approveOnRamp(id)
    override suspend fun rejectRecord(id: String, reason: String) = apiService.rejectOnRamp(id, RejectRequest(reason))
}

class OffRampRepository(apiService: AdminApiService) : ApprovableRepository<OffRampRecord, OffRampRequest>(apiService) {
    override suspend fun fetchList() = apiService.getOffRamp()
    override suspend fun fetchOne(id: String) = apiService.getOffRampRecord(id)
    override suspend fun createRecord(request: OffRampRequest) = apiService.createOffRamp(request)
    override suspend fun updateRecord(id: String, request: OffRampRequest) = apiService.updateOffRamp(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteOffRamp(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setOffRampStatus(id, StatusUpdateRequest(status))
    override suspend fun approveRecord(id: String) = apiService.approveOffRamp(id)
    override suspend fun rejectRecord(id: String, reason: String) = apiService.rejectOffRamp(id, RejectRequest(reason))
}

class P2PClientRepository(apiService: AdminApiService) : DomainRepository<P2PClientRecord, P2PClientRequest>(apiService) {
    override suspend fun fetchList() = apiService.getP2PClients()
    override suspend fun fetchOne(id: String) = apiService.getP2PClientRecord(id)
    override suspend fun createRecord(request: P2PClientRequest) = apiService.createP2PClient(request)
    override suspend fun updateRecord(id: String, request: P2PClientRequest) = apiService.updateP2PClient(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteP2PClient(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setP2PClientStatus(id, StatusUpdateRequest(status))
}

class P2PMerchantRepository(apiService: AdminApiService) : ApprovableRepository<P2PMerchantRecord, P2PMerchantRequest>(apiService) {
    override suspend fun fetchList() = apiService.getP2PMerchants()
    override suspend fun fetchOne(id: String) = apiService.getP2PMerchantRecord(id)
    override suspend fun createRecord(request: P2PMerchantRequest) = apiService.createP2PMerchant(request)
    override suspend fun updateRecord(id: String, request: P2PMerchantRequest) = apiService.updateP2PMerchant(id, request)
    // p2p-merchants has no delete / status endpoints on admin/go — expose approve/reject + transactions.
    override suspend fun deleteRecord(id: String): retrofit2.Response<Unit> =
        throw UnsupportedOperationException("p2p-merchants does not support delete")
    override suspend fun setStatus(id: String, status: String): retrofit2.Response<P2PMerchantRecord> =
        throw UnsupportedOperationException("p2p-merchants does not support status")
    override suspend fun approveRecord(id: String) = apiService.approveP2PMerchant(id)
    override suspend fun rejectRecord(id: String, reason: String) = apiService.rejectP2PMerchant(id, RejectRequest(reason))

    suspend fun transactions(id: String): Result<List<Map<String, Any>>> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getP2PMerchantTransactions(id)
            if (response.isSuccessful) Result.success(response.body() ?: emptyList())
            else Result.failure(Exception("Failed to load transactions (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

class PartnerRepository(apiService: AdminApiService) : ApprovableRepository<PartnerRecord, PartnerRequest>(apiService) {
    override suspend fun fetchList() = apiService.getPartners()
    override suspend fun fetchOne(id: String) = apiService.getPartnerRecord(id)
    override suspend fun createRecord(request: PartnerRequest) = apiService.createPartner(request)
    override suspend fun updateRecord(id: String, request: PartnerRequest) = apiService.updatePartner(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deletePartner(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setPartnerStatus(id, StatusUpdateRequest(status))
    override suspend fun approveRecord(id: String) = apiService.approvePartner(id)
    override suspend fun rejectRecord(id: String, reason: String) = apiService.rejectPartner(id, RejectRequest(reason))
}

class RewardRepository(apiService: AdminApiService) : DomainRepository<RewardRecord, RewardRequest>(apiService) {
    override suspend fun fetchList() = apiService.getRewards()
    override suspend fun fetchOne(id: String) = apiService.getRewardRecord(id)
    override suspend fun createRecord(request: RewardRequest) = apiService.createReward(request)
    override suspend fun updateRecord(id: String, request: RewardRequest) = apiService.updateReward(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteReward(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setRewardStatus(id, StatusUpdateRequest(status))
}

class MarketingRepository(apiService: AdminApiService) : DomainRepository<MarketingRecord, MarketingRequest>(apiService) {
    override suspend fun fetchList() = apiService.getMarketing()
    override suspend fun fetchOne(id: String) = apiService.getMarketingRecord(id)
    override suspend fun createRecord(request: MarketingRequest) = apiService.createMarketing(request)
    override suspend fun updateRecord(id: String, request: MarketingRequest) = apiService.updateMarketing(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteMarketing(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setMarketingStatus(id, StatusUpdateRequest(status))
}

/**
 * RBAC repository: roles CRUD, permissions CRUD, and admin role/permission assignments.
 */
class RolesRepository(apiService: AdminApiService) : DomainRepository<RoleRecord, RoleRequest>(apiService) {
    override suspend fun fetchList() = apiService.getRoles()
    override suspend fun fetchOne(id: String) = apiService.getRole(id)
    override suspend fun createRecord(request: RoleRequest) = apiService.createRole(request)
    override suspend fun updateRecord(id: String, request: RoleRequest) = apiService.updateRole(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteRole(id)
    override suspend fun setStatus(id: String, status: String) = apiService.updateRole(id, RoleRequest(name = status))
}

class PermissionsRepository(apiService: AdminApiService) : DomainRepository<PermissionRecord, PermissionRequest>(apiService) {
    override suspend fun fetchList() = apiService.getPermissions()
    override suspend fun fetchOne(id: String) = apiService.getPermission(id)
    override suspend fun createRecord(request: PermissionRequest) = apiService.createPermission(request)
    override suspend fun updateRecord(id: String, request: PermissionRequest) = apiService.updatePermission(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deletePermission(id)
    override suspend fun setStatus(id: String, status: String) = apiService.updatePermission(id, PermissionRequest(name = status))
}

class AdminRbacRepository(private val apiService: AdminApiService) {

    suspend fun assignRole(adminId: String, roleId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.assignRole(adminId, AssignRoleRequest(roleId))
            if (response.isSuccessful) Result.success(Unit)
            else Result.failure(Exception("Failed to assign role (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun revokeRole(adminId: String, roleId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.revokeRole(adminId, roleId)
            if (response.isSuccessful) Result.success(Unit)
            else Result.failure(Exception("Failed to revoke role (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getEffectivePermissions(adminId: String): Result<List<PermissionRecord>> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getAdminPermissions(adminId)
            if (response.isSuccessful) Result.success(response.body() ?: emptyList())
            else Result.failure(Exception("Failed to load permissions (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

/**
 * New admin domain repositories (bots, bots-clients, project-teams, liquidity-sources)
 * backed by the admin/go service on port 9093. CRUD + status come from DomainRepository;
 * each adds its domain-specific sub-resource methods.
 */
class BotsDomainRepository(apiService: AdminApiService) : DomainRepository<BotRecord, BotRequest>(apiService) {
    override suspend fun fetchList() = apiService.getBotsDomain()
    override suspend fun fetchOne(id: String) = apiService.getBotRecord(id)
    override suspend fun createRecord(request: BotRequest) = apiService.createBotRecord(request)
    override suspend fun updateRecord(id: String, request: BotRequest) = apiService.updateBotRecord(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteBotRecord(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setBotStatus(id, StatusUpdateRequest(status))

    suspend fun getStats(): Result<DomainStatsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getBotDomainStats()
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to load bot stats (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getTiers(id: String): Result<List<BotTierRecord>> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getBotTiers(id)
            if (response.isSuccessful) Result.success(response.body() ?: emptyList())
            else Result.failure(Exception("Failed to load tiers (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createTier(id: String, request: BotTierRequest): Result<BotTierRecord> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.createBotTier(id, request)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to create tier (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateTier(id: String, tierId: String, request: BotTierRequest): Result<BotTierRecord> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.updateBotTier(id, tierId, request)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to update tier (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteTier(id: String, tierId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.deleteBotTier(id, tierId)
            if (response.isSuccessful) Result.success(Unit)
            else Result.failure(Exception("Failed to delete tier (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

class BotsClientsRepository(apiService: AdminApiService) : DomainRepository<BotsClientRecord, BotsClientRequest>(apiService) {
    override suspend fun fetchList() = apiService.getBotsClients()
    override suspend fun fetchOne(id: String) = apiService.getBotsClientRecord(id)
    override suspend fun createRecord(request: BotsClientRequest) = apiService.createBotsClient(request)
    override suspend fun updateRecord(id: String, request: BotsClientRequest) = apiService.updateBotsClient(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteBotsClient(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setBotsClientStatus(id, StatusUpdateRequest(status))
}

class ProjectTeamsRepository(apiService: AdminApiService) : DomainRepository<ProjectTeamRecord, ProjectTeamRequest>(apiService) {
    override suspend fun fetchList() = apiService.getProjectTeams()
    override suspend fun fetchOne(id: String) = apiService.getProjectTeamRecord(id)
    override suspend fun createRecord(request: ProjectTeamRequest) = apiService.createProjectTeam(request)
    override suspend fun updateRecord(id: String, request: ProjectTeamRequest) = apiService.updateProjectTeam(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteProjectTeam(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setProjectTeamStatus(id, StatusUpdateRequest(status))

    suspend fun getMembers(id: String): Result<List<ProjectTeamMemberRecord>> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getProjectTeamMembers(id)
            if (response.isSuccessful) Result.success(response.body() ?: emptyList())
            else Result.failure(Exception("Failed to load members (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun addMember(id: String, request: AddProjectTeamMemberRequest): Result<ProjectTeamMemberRecord> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.addProjectTeamMember(id, request)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to add member (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun removeMember(id: String, memberId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.removeProjectTeamMember(id, memberId)
            if (response.isSuccessful) Result.success(Unit)
            else Result.failure(Exception("Failed to remove member (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

class LiquiditySourcesRepository(apiService: AdminApiService) : DomainRepository<LiquiditySourceRecord, LiquiditySourceRequest>(apiService) {
    override suspend fun fetchList() = apiService.getLiquiditySources()
    override suspend fun fetchOne(id: String) = apiService.getLiquiditySourceRecord(id)
    override suspend fun createRecord(request: LiquiditySourceRequest) = apiService.createLiquiditySource(request)
    override suspend fun updateRecord(id: String, request: LiquiditySourceRequest) = apiService.updateLiquiditySource(id, request)
    override suspend fun deleteRecord(id: String) = apiService.deleteLiquiditySource(id)
    override suspend fun setStatus(id: String, status: String) = apiService.setLiquiditySourceStatus(id, StatusUpdateRequest(status))

    suspend fun setPriority(id: String, priority: Int): Result<LiquiditySourceRecord> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.setLiquiditySourcePriority(id, SetLiquiditySourcePriorityRequest(priority))
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to set priority (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun healthCheck(id: String): Result<LiquiditySourceRecord> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.liquiditySourceHealthCheck(id)
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to run health check (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getStats(): Result<DomainStatsResponse> = withContext(Dispatchers.IO) {
        try {
            val response = apiService.getLiquiditySourceStats()
            if (response.isSuccessful && response.body() != null) Result.success(response.body()!!)
            else Result.failure(Exception("Failed to load liquidity source stats (${response.code()})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
