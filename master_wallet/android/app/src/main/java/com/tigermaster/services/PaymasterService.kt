package com.tigermaster.services

import android.content.Context
import android.util.Base64
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import org.web3j.crypto.ECKeyPair
import org.web3j.crypto.Hash
import org.web3j.crypto.Sign
import java.math.BigInteger
import java.net.HttpURLConnection
import java.net.URL

/**
 * MasterWallet Paymaster Service (Android)
 * ERC-4337 Paymaster support for gasless transactions.
 *
 * Gas prices are sourced from the canonical backend (GET /api/v1/gas) — never
 * hardcoded. Client-side sponsorship signing requires a paymaster signer key;
 * when one is not available, sponsorship is delegated to the backend
 * (POST /api/aa/paymaster/sponsor) or fails closed rather than emitting a
 * fake all-zero signature.
 */
class PaymasterService(private val context: Context) {

    class PaymasterException(message: String) : Exception(message)

    companion object {
        private const val BASE_URL = "http://localhost:8450"
        private const val PREFS_NAME = "paymaster_prefs"
        private const val DEFAULT_ENTRY_POINT = "0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3"
    }

    private var authToken: String? = null

    fun setAuthToken(token: String?) {
        authToken = token
    }

    private val masterKey: MasterKey by lazy {
        MasterKey.Builder(context)
            .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
            .build()
    }

    private val encryptedPrefs by lazy {
        EncryptedSharedPreferences.create(
            context,
            PREFS_NAME,
            masterKey,
            EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
            EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        )
    }

    private var paymasterAddress: String = ""
    private var isInitialized: Boolean = false
    private var gasPriceCache: Map<String, GasPrices> = emptyMap()

    /**
     * Initialize the paymaster service
     */
    fun initialize(): Boolean {
        return try {
            // Load configuration
            paymasterAddress = encryptedPrefs.getString("paymasterAddress", "") ?: ""

            // Start gas price monitoring
            isInitialized = true
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Validate user operation
     */
    suspend fun validateUserOperation(
        sender: String,
        nonce: Long,
        initCode: String,
        callData: String,
        callGasLimit: Long,
        verificationGasLimit: Long,
        preVerificationGas: Long,
        maxFeePerGas: Long,
        maxPriorityFeePerGas: Long,
        chainId: String
    ): String {
        // Validate sender
        if (sender.isEmpty()) {
            return "AA10: sender not specified"
        }

        // Validate nonce
        if (nonce == Long.MAX_VALUE) {
            return "AA11: nonce too large"
        }

        // Validate gas limits
        if (callGasLimit > 5000000L || verificationGasLimit > 5000000L) {
            return "AA13: gas limit too high"
        }

        // Check sponsorship policy
        val canSponsor = canSponsor(sender, chainId)
        if (!canSponsor.first) {
            return "AA23: not sponsored"
        }

        return "0" // Success
    }

    /**
     * Sponsor user operation - generate paymasterAndData
     */
    suspend fun sponsorUserOperation(
        userOp: Map<String, Any>,
        chainId: String
    ): String? {
        try {
            // Validate first
            val validationResult = validateUserOperation(
                sender = userOp["sender"] as? String ?: "",
                nonce = (userOp["nonce"] as? Number)?.toLong() ?: 0,
                initCode = userOp["initCode"] as? String ?: "",
                callData = userOp["callData"] as? String ?: "",
                callGasLimit = (userOp["callGasLimit"] as? Number)?.toLong() ?: 0,
                verificationGasLimit = (userOp["verificationGasLimit"] as? Number)?.toLong() ?: 0,
                preVerificationGas = (userOp["preVerificationGas"] as? Number)?.toLong() ?: 0,
                maxFeePerGas = (userOp["maxFeePerGas"] as? Number)?.toLong() ?: 0,
                maxPriorityFeePerGas = (userOp["maxPriorityFeePerGas"] as? Number)?.toLong() ?: 0,
                chainId = chainId
            )

            if (validationResult != "0") {
                return null
            }

            // Build paymasterAndData
            val paymasterAndData = buildPaymasterAndData(userOp, chainId)

            // Increment daily usage
            incrementDailyUsage()

            return paymasterAndData
        } catch (e: Exception) {
            e.printStackTrace()
            return null
        }
    }

    /**
     * Build paymasterAndData according to ERC-4337. Sponsorship signing that
     * can be done client-side (paymaster signer key configured) is real; when
     * no signer key is configured the operation is fail-closed (the canonical
     * MasterWallet backend on :8450 has no /api/aa/paymaster/sponsor route —
     * ERC-4337 paymaster sponsorship is not part of the canonical contract).
     */
    private suspend fun buildPaymasterAndData(userOp: Map<String, Any>, chainId: String): String {
        val validUntil = 0 // Always valid

        val signerKey = getPaymasterSignerKey()
        if (signerKey == null) {
            throw PaymasterException(
                "No paymaster signer key configured and the canonical MasterWallet " +
                    "backend does not host a paymaster sponsor endpoint; refusing to " +
                    "sponsor (fail-closed)"
            )
        }

        val hash = hashPaymasterData(userOp, chainId, validUntil)
        val signature = signMessage(hash, signerKey)

        if (paymasterAddress.isEmpty()) {
            throw PaymasterException(
                "Paymaster address not configured; refusing to sponsor (fail-closed)"
            )
        }
        val validUntilHex = String.format("%08x", validUntil)
        return paymasterAddress + validUntilHex + signature
    }

    private fun getPaymasterSignerKey(): ECKeyPair? {
        val stored = encryptedPrefs.getString("paymasterSignerKey", null) ?: return null
        return try {
            val priv = BigInteger(stored, 16)
            ECKeyPair.create(priv)
        } catch (e: Exception) {
            null
        }
    }

    fun setPaymasterSignerKey(privateKeyHex: String): Boolean {
        return try {
            val clean = privateKeyHex.removePrefix("0x")
            val priv = BigInteger(clean, 16)
            val keyPair = ECKeyPair.create(priv)
            encryptedPrefs.edit().putString("paymasterSignerKey", keyPair.privateKey.toString(16)).apply()
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Hash paymaster data with keccak256 (Web3j), not SHA-256.
     */
    private fun hashPaymasterData(userOp: Map<String, Any>, chainId: String, validUntil: Int): ByteArray {
        val data = buildString {
            append(paymasterAddress)
            append(String.format("%08x", validUntil))
            append(userOp["sender"] ?: "")
            append(userOp["nonce"] ?: "0")
            append(chainId)
        }
        return Hash.sha3(data.toByteArray())
    }

    /**
     * Sign a message with REAL secp256k1 ECDSA via Web3j.
     */
    private fun signMessage(data: ByteArray, keyPair: ECKeyPair): String {
        val sig = Sign.signMessage(data, keyPair, false)
        val r = sig.r.toString(16).padStart(64, '0')
        val s = sig.s.toString(16).padStart(64, '0')
        val v = (sig.v.toInt() + 27).toString(16).padStart(2, '0')
        return "0x$r$s$v"
    }

    /**
     * Check if transaction can be sponsored
     */
    fun canSponsor(sender: String, chainId: String): Pair<Boolean, String> {
        val policy = getPolicy()

        if (!policy.enabled) {
            return Pair(false, "Sponsorship disabled")
        }

        // Check daily limit
        if (policy.dailyUsed >= policy.maxDailySponsored) {
            return Pair(false, "Daily limit reached")
        }

        // Check sender whitelist
        if (policy.requireWhitelist && policy.allowedSenders.isNotEmpty()) {
            if (!policy.allowedSenders.contains(sender)) {
                return Pair(false, "Sender not whitelisted")
            }
        }

        // Check blocked senders
        if (policy.blockedSenders.contains(sender)) {
            return Pair(false, "Sender blocked")
        }

        return Pair(true, "")
    }

    /**
     * Get current gas prices
     */
    suspend fun getGasPrices(chainId: String): GasPrices? {
        // Check cache first
        gasPriceCache[chainId]?.let { cached ->
            val age = System.currentTimeMillis() - cached.timestamp
            if (age < 15000) { // 15 seconds cache
                return cached
            }
        }

        // Fetch fresh prices
        return fetchGasPrices(chainId)
    }

    /**
     * Fetch real gas prices from the canonical backend (GET /api/v1/gas).
     * Never returns hardcoded/simulated values; on failure returns null.
     */
    private suspend fun fetchGasPrices(chainId: String): GasPrices? = withContext(Dispatchers.IO) {
        try {
            val result = makeRequest("GET", "/api/v1/gas?chain_id=$chainId")
            val gasPrice = result.optLong("gas_price", -1)
            val maxFee = result.optLong("max_fee", -1)
            val priorityFee = result.optLong("priority_fee", -1)
            if (gasPrice <= 0 && maxFee <= 0) return@withContext null

            val resolvedMax = if (maxFee > 0) maxFee else gasPrice
            val resolvedPriority = if (priorityFee > 0) priorityFee else (resolvedMax / 10L)
            val resolvedBase = if (gasPrice > 0) gasPrice else resolvedMax
            GasPrices(
                baseFeePerGas = resolvedBase,
                maxFeePerGas = resolvedMax,
                maxPriorityFeePerGas = resolvedPriority,
                suggestedMaxFeePerGas = resolvedMax,
                suggestedMaxPriorityFeePerGas = resolvedPriority,
                timestamp = System.currentTimeMillis()
            ).also { prices ->
                gasPriceCache = gasPriceCache + (chainId to prices)
            }
        } catch (e: Exception) {
            e.printStackTrace()
            null
        }
    }

    /**
     * Calculate post-op gas
     */
    fun calculatePostOpGas(actualGasUsed: Long): Long {
        val baseGas = 21000L
        val perUserOpGas = 21000L
        val perCalldataByte = 16L

        // Estimate based on typical call data
        val estimatedCallDataGas = 1000L * perCalldataByte

        return baseGas + perUserOpGas + estimatedCallDataGas + (actualGasUsed / 10)
    }

    /**
     * Calculate fee for user operation
     */
    suspend fun calculateFee(userOp: Map<String, Any>, chainId: String): Long {
        val gasPrices = getGasPrices(chainId) ?: return 0L

        val callGasLimit = (userOp["callGasLimit"] as? Number)?.toLong() ?: 100000L
        val verificationGasLimit = (userOp["verificationGasLimit"] as? Number)?.toLong() ?: 150000L
        val preVerificationGas = (userOp["preVerificationGas"] as? Number)?.toLong() ?: 21000L

        val totalGas = callGasLimit + verificationGasLimit + preVerificationGas
        val policy = getPolicy()

        val baseFee = totalGas * gasPrices.maxFeePerGas
        val markup = (baseFee * policy.markupPercent / 100).toLong()

        return baseFee + markup
    }

    /**
     * Get sponsorship policy
     */
    fun getPolicy(): SponsorshipPolicy {
        val stored = encryptedPrefs.getString("sponsorshipPolicy", null)
        return if (stored != null) {
            try {
                val json = JSONObject(stored)
                SponsorshipPolicy(
                    id = json.optString("id", "default"),
                    enabled = json.optBoolean("enabled", true),
                    maxDailySponsored = json.optInt("maxDailySponsored", 1000).toLong(),
                    dailyUsed = json.optInt("dailyUsed", 0).toLong(),
                    minTransactionValue = json.optString("minTransactionValue", "0"),
                    maxTransactionValue = json.optString("maxTransactionValue", "1000000000000000000"),
                    allowedSenders = json.optJSONArray("allowedSenders")?.toStringList() ?: emptyList(),
                    blockedSenders = json.optJSONArray("blockedSenders")?.toStringList() ?: emptyList(),
                    requireWhitelist = json.optBoolean("requireWhitelist", false),
                    markupPercent = json.optDouble("markupPercent", 10.0)
                )
            } catch (e: Exception) {
                getDefaultPolicy()
            }
        } else {
            getDefaultPolicy().also { savePolicy(it) }
        }
    }

    /**
     * Get default policy
     */
    private fun getDefaultPolicy(): SponsorshipPolicy {
        return SponsorshipPolicy(
            id = "default",
            enabled = true,
            maxDailySponsored = 1000L,
            dailyUsed = 0L,
            minTransactionValue = "0",
            maxTransactionValue = "1000000000000000000",
            allowedSenders = emptyList(),
            blockedSenders = emptyList(),
            requireWhitelist = false,
            markupPercent = 10.0
        )
    }

    /**
     * Save policy
     */
    private fun savePolicy(policy: SponsorshipPolicy) {
        val json = JSONObject().apply {
            put("id", policy.id)
            put("enabled", policy.enabled)
            put("maxDailySponsored", policy.maxDailySponsored)
            put("dailyUsed", policy.dailyUsed)
            put("minTransactionValue", policy.minTransactionValue)
            put("maxTransactionValue", policy.maxTransactionValue)
            put("allowedSenders", JSONArray(policy.allowedSenders))
            put("blockedSenders", JSONArray(policy.blockedSenders))
            put("requireWhitelist", policy.requireWhitelist)
            put("markupPercent", policy.markupPercent)
        }

        encryptedPrefs.edit().putString("sponsorshipPolicy", json.toString()).apply()
    }

    /**
     * Increment daily usage
     */
    private fun incrementDailyUsage() {
        val policy = getPolicy()
        val updatedPolicy = policy.copy(dailyUsed = policy.dailyUsed + 1)
        savePolicy(updatedPolicy)
    }

    /**
     * Reset daily usage (called daily)
     */
    fun resetDailyUsage() {
        val policy = getPolicy()
        val updatedPolicy = policy.copy(dailyUsed = 0L)
        savePolicy(updatedPolicy)
    }

    /**
     * Create new policy
     */
    fun createPolicy(policy: SponsorshipPolicy): Boolean {
        return try {
            savePolicy(policy)
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Get paymaster balance. The canonical MasterWallet backend (:8450) does not
     * host an /api/aa/paymaster/balance route (ERC-4337 paymaster balance is read
     * directly from the EntryPoint contract on-chain, not via this backend).
     * Fail-closed: returns empty to signal no value rather than fabricating "0".
     */
    suspend fun getPaymasterBalance(chainId: String): String {
        throw PaymasterException(
            "Paymaster balance is not available via the canonical MasterWallet backend; " +
                "read it from the EntryPoint contract on-chain (fail-closed)"
        )
    }

    /**
     * Fund paymaster. The canonical MasterWallet backend (:8450) does not host
     * an /api/aa/paymaster/fund route. Funding the paymaster is an on-chain
     * deposit to the EntryPoint; this client-side shortcut is fail-closed.
     */
    suspend fun fundPaymaster(chainId: String, amount: String): Boolean {
        throw PaymasterException(
            "Paymaster funding is not supported via the canonical MasterWallet backend; " +
                "deposit to the EntryPoint contract on-chain (fail-closed)"
        )
    }

    /**
     * Get statistics
     */
    fun getStats(): PaymasterStats {
        val policy = getPolicy()

        return PaymasterStats(
            totalSponsored = policy.dailyUsed,
            dailyLimit = policy.maxDailySponsored,
            availableToday = policy.maxDailySponsored - policy.dailyUsed,
            markupPercent = policy.markupPercent
        )
    }

    /**
     * Check if service is initialized
     */
    fun isInitialized(): Boolean = isInitialized

    /**
     * Get entry point address
     */
    fun getEntryPoint(): String = DEFAULT_ENTRY_POINT

    // Private helper

    private suspend fun makeRequest(
        method: String,
        endpoint: String,
        body: Map<String, Any>? = null
    ): JSONObject = withContext(Dispatchers.IO) {
        val url = URL("$BASE_URL$endpoint")
        val connection = url.openConnection() as HttpURLConnection

        connection.requestMethod = method
        connection.setRequestProperty("Content-Type", "application/json")
        connection.connectTimeout = 15000
        connection.readTimeout = 15000
        authToken?.takeIf { it.isNotEmpty() }?.let {
            connection.setRequestProperty("Authorization", "Bearer $it")
        }

        if (body != null) connection.doOutput = true
        connection.connect()

        body?.let {
            val bodyBytes = JSONObject(it).toString().toByteArray()
            connection.outputStream.use { os -> os.write(bodyBytes) }
        }

        val responseCode = connection.responseCode
        val responseBody = if (responseCode in 200..299) {
            connection.inputStream?.bufferedReader()?.readText() ?: ""
        } else {
            connection.errorStream?.bufferedReader()?.readText() ?: ""
        }

        connection.disconnect()

        if (responseCode !in 200..299) {
            throw PaymasterException(
                "Backend $endpoint failed: HTTP $responseCode ${responseBody.take(200)}"
            )
        }

        if (responseBody.isNotEmpty()) {
            JSONObject(responseBody)
        } else {
            JSONObject()
        }
    }

    private fun JSONArray.toStringList(): List<String> {
        return (0 until length()).map { get(it) as String }
    }
}

/**
 * Gas prices data class
 */
data class GasPrices(
    val baseFeePerGas: Long,
    val maxFeePerGas: Long,
    val maxPriorityFeePerGas: Long,
    val suggestedMaxFeePerGas: Long,
    val suggestedMaxPriorityFeePerGas: Long,
    val timestamp: Long
)

/**
 * Sponsorship policy data class
 */
data class SponsorshipPolicy(
    val id: String,
    val enabled: Boolean,
    val maxDailySponsored: Long,
    val dailyUsed: Long,
    val minTransactionValue: String,
    val maxTransactionValue: String,
    val allowedSenders: List<String>,
    val blockedSenders: List<String>,
    val requireWhitelist: Boolean,
    val markupPercent: Double
)

/**
 * Paymaster statistics
 */
data class PaymasterStats(
    val totalSponsored: Long,
    val dailyLimit: Long,
    val availableToday: Long,
    val markupPercent: Double
)
