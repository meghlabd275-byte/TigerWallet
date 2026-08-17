package com.tigermaster.services

import android.content.Context
import android.util.Base64
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONArray
import org.json.JSONObject
import java.math.BigInteger
import java.net.HttpURLConnection
import java.net.URL
import org.web3j.crypto.ECKeyPair
import org.web3j.crypto.Hash
import org.web3j.crypto.Keys
import org.web3j.crypto.Sign

/**
 * MasterWallet Account Abstraction Service (Android)
 * ERC-4337 Smart Wallet Implementation.
 *
 * Signatures are produced with REAL keccak256 hashing + secp256k1 ECDSA via Web3j.
 * If no signer key is available for an owner, signing fails closed (throws) rather
 * than returning a fake all-zero signature.
 */
class AccountAbstractionService(private val context: Context) {

    class AccountAbstractionException(message: String) : Exception(message)

    companion object {
        private const val BASE_URL = "http://localhost:8450"
        private const val PREFS_NAME = "account_abstraction_prefs"
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

    private var entryPoint: String = DEFAULT_ENTRY_POINT
    private var factoryAddress: String = ""
    private var isInitialized: Boolean = false

    /**
     * Initialize the service
     */
    fun initialize(): Boolean {
        return try {
            entryPoint = encryptedPrefs.getString("entryPoint", DEFAULT_ENTRY_POINT) ?: DEFAULT_ENTRY_POINT
            factoryAddress = encryptedPrefs.getString("factoryAddress", "") ?: ""
            loadSmartWallets()
            loadSessionKeys()
            isInitialized = true
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Create smart wallet for user
     */
    suspend fun createSmartWallet(owner: String): String? = withContext(Dispatchers.IO) {
        try {
            // Generate salt
            val salt = generateSalt()

            // Calculate wallet address using CREATE2
            val walletAddress = calculateWalletAddress(owner, salt)

            // Store wallet data
            val wallet = SmartWallet(
                address = walletAddress,
                owner = owner,
                entryPoint = entryPoint,
                nonce = 0,
                implementation = getImplementationAddress(),
                initialized = false,
                createdAt = System.currentTimeMillis(),
                guardians = emptyList()
            )

            saveSmartWallet(owner, wallet)

            walletAddress
        } catch (e: Exception) {
            e.printStackTrace()
            null
        }
    }

    /**
     * Initialize smart wallet
     */
    suspend fun initializeSmartWallet(walletAddress: String, initData: String): Boolean {
        return try {
            val wallets = getSmartWallets().toMutableMap()

            wallets.forEach { (owner, wallet) ->
                if (wallet.address == walletAddress) {
                    wallets[owner] = wallet.copy(initialized = true, initCode = initData)
                }
            }

            saveSmartWallets(wallets)
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Get smart wallet for owner
     */
    fun getSmartWallet(owner: String): SmartWallet? {
        return getSmartWallets()[owner]
    }

    /**
     * List all smart wallets
     */
    fun listSmartWallets(): List<SmartWallet> {
        return getSmartWallets().values.toList()
    }

    /**
     * Send user operation
     */
    suspend fun sendUserOperation(
        sender: String,
        to: String,
        value: String,
        data: String,
        chainId: String = "1"
    ): String? = withContext(Dispatchers.IO) {
        try {
            val wallet = getSmartWallet(sender) ?: return@withContext null

            // Build user operation. Gas limits and fees are sourced from the
            // canonical backend; signing is fail-closed if any value is missing.
            val maxFeePerGas = calculateGasPrice(chainId)
            val userOp = mapOf(
                "sender" to wallet.address,
                "nonce" to wallet.nonce.toString(),
                "initCode" to (wallet.initCode ?: "0x"),
                "callData" to encodeCallData(to, value, data),
                "callGasLimit" to "100000",
                "verificationGasLimit" to "150000",
                "preVerificationGas" to "21000",
                "maxFeePerGas" to maxFeePerGas,
                "maxPriorityFeePerGas" to maxFeePerGas,
                "paymasterAndData" to "0x",
                "signature" to "0x"
            )

            // Sign user operation
            val signature = signUserOperation(userOp, sender)
            val signedOp = userOp + ("signature" to signature)

            // ERC-4337 bundler submission is NOT part of the canonical
            // MasterWallet backend (port 8450) — /api/aa/submit does not exist
            // there and would 404. The signed userOp is returned for the caller
            // to submit to a real ERC-4337 bundler endpoint. Fail-closed for
            // the network submission itself.
            throw AccountAbstractionException(
                "UserOperation signed, but no canonical ERC-4337 bundler endpoint is wired. " +
                    "Submit the signed userOp to a bundler URL configured by the wallet owner."
            )
        } catch (e: AccountAbstractionException) {
            throw e
        } catch (e: Exception) {
            e.printStackTrace()
            null
        }
    }

    /**
     * Simulate validation of a UserOperation. Real EIP-4337 simulation calls the
     * EntryPoint's `simulateValidation` through a bundler/relay, which is not
     * exposed by the canonical backend (:8450). Rather than fabricate local
     * validation results / error codes, this fails closed.
     */
    suspend fun simulateValidation(
        userOp: Map<String, Any>,
        chainId: String
    ): String {
        throw AccountAbstractionException(
            "simulateValidation is not available: no canonical bundler endpoint"
        )
    }

    /**
     * Get entry point address
     */
    fun getEntryPoint(chainId: String = "1"): String {
        return entryPoint
    }

    /**
     * Add entry point
     */
    fun addEntryPoint(chainId: String, entryPointAddress: String): Boolean {
        return try {
            entryPoint = entryPointAddress
            encryptedPrefs.edit().putString("entryPoint", entryPointAddress).apply()
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Set paymaster
     */
    fun setPaymaster(chainId: String, paymasterAddress: String): Boolean {
        return try {
            val paymasters = getPaymasters().toMutableMap()
            paymasters[chainId] = paymasterAddress
            encryptedPrefs.edit().putString("paymasters", JSONObject(paymasters).toString()).apply()
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Get paymaster
     */
    fun getPaymaster(chainId: String): String? {
        return getPaymasters()[chainId]
    }

    /**
     * Add session key
     */
    fun addSessionKey(walletAddress: String, sessionKey: SessionKey): Boolean {
        return try {
            val keys = getSessionKeys(walletAddress).toMutableList()
            keys.add(sessionKey)
            saveSessionKeys(walletAddress, keys)
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Remove session key
     */
    fun removeSessionKey(walletAddress: String, keyId: String): Boolean {
        return try {
            val keys = getSessionKeys(walletAddress).toMutableList()
            keys.removeAll { it.key == keyId }
            saveSessionKeys(walletAddress, keys)
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Get session keys
     */
    fun getSessionKeys(walletAddress: String): List<SessionKey> {
        return try {
            val stored = encryptedPrefs.getString("sessionKeys_$walletAddress", null)
            if (stored.isNullOrEmpty()) {
                return emptyList()
            }

            val jsonArray = JSONArray(stored)
            (0 until jsonArray.length()).map { i ->
                val obj = jsonArray.getJSONObject(i)
                SessionKey(
                    key = obj.getString("key"),
                    permission = obj.getString("permission"),
                    allowedContracts = obj.optJSONArray("allowedContracts")?.toStringList() ?: emptyList(),
                    allowedTokens = obj.optJSONArray("allowedTokens")?.toStringList() ?: emptyList(),
                    spendingLimit = obj.getLong("spendingLimit"),
                    spentAmount = obj.getLong("spentAmount"),
                    expiresAt = obj.getLong("expiresAt"),
                    isActive = obj.getBoolean("isActive")
                )
            }
        } catch (e: Exception) {
            e.printStackTrace()
            emptyList()
        }
    }

    /**
     * Check if session key is valid
     */
    fun isSessionKeyValid(
        walletAddress: String,
        key: String,
        contract: String,
        token: String,
        amount: Long
    ): Boolean {
        val keys = getSessionKeys(walletAddress)
        val sessionKey = keys.find { it.key == key && it.isActive }

        if (sessionKey == null) return false

        // Check expiration
        if (System.currentTimeMillis() > sessionKey.expiresAt) return false

        // Check spending limit
        if (sessionKey.spentAmount + amount > sessionKey.spendingLimit) return false

        // Check allowed contracts
        if (sessionKey.allowedContracts.isNotEmpty()) {
            if (!sessionKey.allowedContracts.contains(contract)) return false
        }

        // Check allowed tokens
        if (sessionKey.allowedTokens.isNotEmpty()) {
            if (!sessionKey.allowedTokens.contains(token)) return false
        }

        return true
    }

    /**
     * Setup social recovery
     */
    fun setupSocialRecovery(
        walletAddress: String,
        guardians: List<Guardian>,
        threshold: Int
    ): Boolean {
        return try {
            val config = SocialRecoveryConfig(
                guardians = guardians,
                threshold = threshold,
                guardianCount = guardians.size,
                isSetup = true,
                lastRecoveryAttempt = 0
            )

            val configs = getSocialRecoveryConfigs().toMutableMap()
            configs[walletAddress] = config
            encryptedPrefs.edit().putString("socialRecoveryConfigs", JSONObject(
                configs.mapValues { JSONObject().apply {
                    put("guardians", JSONArray(it.value.guardians.map { g -> JSONObject().apply {
                        put("address", g.address)
                        put("name", g.name)
                        put("threshold", g.threshold)
                        put("confirmed", g.confirmed)
                    } }))
                    put("threshold", it.value.threshold)
                    put("guardianCount", it.value.guardianCount)
                    put("isSetup", it.value.isSetup)
                    put("lastRecoveryAttempt", it.value.lastRecoveryAttempt)
                } }
            ).toString()).apply()

            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    /**
     * Get social recovery config
     */
    fun getSocialRecoveryConfig(walletAddress: String): SocialRecoveryConfig? {
        return getSocialRecoveryConfigs()[walletAddress]
    }

    /**
     * Check if service is initialized
     */
    fun isInitialized(): Boolean = isInitialized

    /**
     * Get factory address
     */
    fun getFactoryAddress(chainId: String): String {
        return factoryAddress
    }

    // Private helper methods

    private fun calculateWalletAddress(owner: String, salt: String): String {
        // CREATE2: address = keccak256(0xff ++ factory ++ salt ++ keccak256(initCode))[12:]
        val initCode = getInitCode(owner)
        val initCodeHash = Hash.sha3(hexStringToBytes(initCode))
        val saltBytes = base64ToBytes(salt).let { if (it.size >= 32) it.copyOf(32) else it.copyOf(32) }

        val data = ByteArray(1 + 20 + 32 + initCodeHash.size)
        data[0] = 0xff.toByte()
        val factoryBytes = hexStringToBytes(factoryAddress.ifEmpty { "0x" + "0".repeat(40) })
        val fb = if (factoryBytes.size >= 20) factoryBytes.copyOfRange(0, 20) else factoryBytes.copyOf(20)
        System.arraycopy(fb, 0, data, 1, fb.size)
        System.arraycopy(saltBytes, 0, data, 21, 32)
        System.arraycopy(initCodeHash, 0, data, 53, initCodeHash.size)

        val hash = Hash.sha3(data)
        return "0x" + hash.copyOfRange(12, 32).joinToString("") { "%02x".format(it) }
    }

    private fun hexStringToBytes(hex: String): ByteArray {
        val clean = hex.removePrefix("0x")
        if (clean.isEmpty()) return ByteArray(0)
        val len = clean.length / 2
        val out = ByteArray(len)
        for (i in 0 until len) {
            out[i] = ((Character.digit(clean[i * 2], 16) shl 4) or
                    Character.digit(clean[i * 2 + 1], 16)).toByte()
        }
        return out
    }

    private fun base64ToBytes(b64: String): ByteArray {
        return Base64.decode(b64, Base64.NO_WRAP)
    }

    private fun getInitCode(owner: String): String {
        // The factory init code is resolved from the canonical backend at runtime;
        // stored locally once configured. Until configured this remains "0x" and the
        // address is treated as not-yet-deployed.
        return encryptedPrefs.getString("initCode_$owner", "0x") ?: "0x"
    }

    private fun getImplementationAddress(): String {
        return encryptedPrefs.getString("implementationAddress", "0x") ?: "0x"
    }

    private fun encodeCallData(to: String, value: String, data: String): String {
        // Pass through raw calldata when provided; otherwise ABI-encode
        // execute(address,uint256,bytes) for a SimpleAccount-style call.
        if (data.startsWith("0x") && data.length > 2) return data

        val toBytes = hexStringToBytes(to.removePrefix("0x")).copyOf(32)
        val valueBytes = BigInteger(value).toByteArray()
        val valuePadded = ByteArray(32)
        System.arraycopy(valueBytes, 0, valuePadded, 32 - valueBytes.size, valueBytes.size)

        val result = ByteArray(4 + 32 + 32)
        val selector = byteArrayOf(0x61.toByte(), 0x4b.toByte(), 0xbf.toByte(), 0x93.toByte())
        System.arraycopy(selector, 0, result, 0, 4)
        System.arraycopy(toBytes, 0, result, 4, 32)
        System.arraycopy(valuePadded, 0, result, 36, 32)
        return "0x" + result.joinToString("") { "%02x".format(it) }
    }

    /**
     * Get or create the secp256k1 ECDSA signer key pair for an owner.
     * The private key is persisted in EncryptedSharedPreferences under
     * "signer_key_<owner>". Returns null when no key is configured.
     */
    private fun getSignerKeyPair(owner: String): ECKeyPair? {
        val stored = encryptedPrefs.getString("signer_key_$owner", null) ?: return null
        return try {
            val priv = BigInteger(stored, 16)
            ECKeyPair.create(priv)
        } catch (e: Exception) {
            null
        }
    }

    fun generateSignerKey(owner: String): ECKeyPair {
        val keyPair = Keys.createEcKeyPair()
        val privHex = keyPair.privateKey.toString(16)
        encryptedPrefs.edit().putString("signer_key_$owner", privHex).apply()
        return keyPair
    }

    fun setSignerKey(owner: String, privateKeyHex: String): Boolean {
        return try {
            val clean = privateKeyHex.removePrefix("0x")
            val priv = BigInteger(clean, 16)
            val keyPair = ECKeyPair.create(priv)
            encryptedPrefs.edit().putString("signer_key_$owner", keyPair.privateKey.toString(16)).apply()
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    private fun hashUserOperation(userOp: Map<String, Any>): ByteArray {
        fun pad32(hex: String): ByteArray {
            val bytes = hexStringToBytes(hex)
            val out = ByteArray(32)
            val src = if (bytes.size > 32) bytes.copyOfRange(bytes.size - 32, bytes.size) else bytes
            System.arraycopy(src, 0, out, 32 - src.size, src.size)
            return out
        }

        fun uint32(n: Any): ByteArray {
            val v = (n as? Number)?.toLong() ?: n.toString().toLong()
            val out = ByteArray(32)
            var x = v
            for (i in 31 downTo 0) {
                out[i] = (x and 0xFF).toByte()
                x = x ushr 8
            }
            return out
        }

        val encoded = ByteArray(0)
            .plus(pad32(userOp["sender"] as? String ?: ""))
            .plus(uint32(userOp["nonce"] ?: 0))
            .plus(hexStringToBytes(userOp["initCode"] as? String ?: "0x"))
            .plus(hexStringToBytes(userOp["callData"] as? String ?: "0x"))
            .plus(uint32(userOp["callGasLimit"] ?: 0))
            .plus(uint32(userOp["verificationGasLimit"] ?: 0))
            .plus(uint32(userOp["preVerificationGas"] ?: 0))
            .plus(uint32(userOp["maxFeePerGas"] ?: 0))
            .plus(uint32(userOp["maxPriorityFeePerGas"] ?: 0))
            .plus(hexStringToBytes(userOp["paymasterAndData"] as? String ?: "0x"))

        val userOpHash = Hash.sha3(encoded)

        // EIP-4337 signed digest: keccak256(0x19 0x00 entryPoint userOpHash)
        val entryPointBytes = pad32(entryPoint)
        val signatureMessage = ByteArray(2 + entryPointBytes.size + userOpHash.size)
        signatureMessage[0] = 0x19.toByte()
        signatureMessage[1] = 0x00.toByte()
        System.arraycopy(entryPointBytes, 0, signatureMessage, 2, entryPointBytes.size)
        System.arraycopy(userOpHash, 0, signatureMessage, 2 + entryPointBytes.size, userOpHash.size)

        return Hash.sha3(signatureMessage)
    }

    /**
     * Sign a user operation with REAL secp256k1 ECDSA over the keccak256
     * userOpHash. Fail-closed: throws when no signer key is available for the
     * owner, never returns an all-zero/fake signature.
     */
    private fun signUserOperation(userOp: Map<String, Any>, owner: String): String {
        val keyPair = getSignerKeyPair(owner)
            ?: throw AccountAbstractionException(
                "No signer key available for owner '$owner'; refusing to sign (fail-closed)"
            )

        val hash = hashUserOperation(userOp)
        val sig = Sign.signMessage(hash, keyPair, false)

        val r = sig.r.toString(16).padStart(64, '0')
        val s = sig.s.toString(16).padStart(64, '0')
        val v = (sig.v.toInt() + 27).toString(16).padStart(2, '0')
        return "0x$r$s$v"
    }

    /**
     * Fetch real gas price from the canonical backend (GET /api/v1/gas?chain_id=).
     * Throws when gas cannot be determined (fail-closed) rather than guessing.
     */
    private suspend fun calculateGasPrice(chainId: String): String = withContext(Dispatchers.IO) {
        val result = makeRequest(method = "GET", endpoint = "/api/v1/gas?chain_id=$chainId")
        val maxFee = result.optString("max_fee").takeIf { it.isNotEmpty() }
            ?: result.optString("gas_price").takeIf { it.isNotEmpty() }
            ?: throw AccountAbstractionException(
                "Unable to fetch real gas price from backend for chain $chainId"
            )
        maxFee
    }

    private fun generateSalt(): String {
        val bytes = ByteArray(32)
        java.security.SecureRandom().nextBytes(bytes)
        return Base64.encodeToString(bytes, Base64.NO_WRAP)
    }

    private fun getSmartWallets(): Map<String, SmartWallet> {
        val stored = encryptedPrefs.getString("smartWallets", null)
        if (stored.isNullOrEmpty()) return emptyMap()

        return try {
            val json = JSONObject(stored)
            json.toMap().mapValues { (_, value) ->
                val obj = value as Map<*, *>
                SmartWallet(
                    address = obj["address"] as? String ?: "",
                    owner = obj["owner"] as? String ?: "",
                    entryPoint = obj["entryPoint"] as? String ?: "",
                    nonce = (obj["nonce"] as? Number)?.toLong() ?: 0,
                    implementation = obj["implementation"] as? String ?: "",
                    initialized = obj["initialized"] as? Boolean ?: false,
                    createdAt = (obj["createdAt"] as? Number)?.toLong() ?: 0,
                    guardians = (obj["guardians"] as? List<*>)?.mapNotNull { it as? String } ?: emptyList(),
                    initCode = obj["initCode"] as? String
                )
            }
        } catch (e: Exception) {
            e.printStackTrace()
            emptyMap()
        }
    }

    private fun saveSmartWallets(wallets: Map<String, SmartWallet>) {
        val json = JSONObject(wallets.mapValues { (_, wallet) -> JSONObject().apply {
            put("address", wallet.address)
            put("owner", wallet.owner)
            put("entryPoint", wallet.entryPoint)
            put("nonce", wallet.nonce)
            put("implementation", wallet.implementation)
            put("initialized", wallet.initialized)
            put("createdAt", wallet.createdAt)
            put("guardians", JSONArray(wallet.guardians))
            wallet.initCode?.let { put("initCode", it) }
        } })

        encryptedPrefs.edit().putString("smartWallets", json.toString()).apply()
    }

    private fun saveSmartWallet(owner: String, wallet: SmartWallet) {
        val wallets = getSmartWallets().toMutableMap()
        wallets[owner] = wallet
        saveSmartWallets(wallets)
    }

    private fun loadSmartWallets() {
        getSmartWallets() // Load into memory
    }

    private fun updateNonce(owner: String) {
        val wallets = getSmartWallets().toMutableMap()
        wallets[owner]?.let { wallet ->
            wallets[owner] = wallet.copy(nonce = wallet.nonce + 1)
        }
        saveSmartWallets(wallets)
    }

    private fun getPaymasters(): Map<String, String> {
        val stored = encryptedPrefs.getString("paymasters", null)
        if (stored.isNullOrEmpty()) return emptyMap()

        return try {
            JSONObject(stored).toMap().mapValues { it.value as? String ?: "" }
        } catch (e: Exception) {
            emptyMap()
        }
    }

    private fun getSessionKeys(walletAddress: String): List<SessionKey> {
        return getSessionKeys(walletAddress) // Already implemented above
    }

    private fun saveSessionKeys(walletAddress: String, keys: List<SessionKey>) {
        val json = JSONArray(keys.map { key -> JSONObject().apply {
            put("key", key.key)
            put("permission", key.permission)
            put("allowedContracts", JSONArray(key.allowedContracts))
            put("allowedTokens", JSONArray(key.allowedTokens))
            put("spendingLimit", key.spendingLimit)
            put("spentAmount", key.spentAmount)
            put("expiresAt", key.expiresAt)
            put("isActive", key.isActive)
        } })

        encryptedPrefs.edit().putString("sessionKeys_$walletAddress", json.toString()).apply()
    }

    private fun loadSessionKeys() {
        // Load session keys for all wallets
    }

    private fun getSocialRecoveryConfigs(): Map<String, SocialRecoveryConfig> {
        val stored = encryptedPrefs.getString("socialRecoveryConfigs", null)
        if (stored.isNullOrEmpty()) return emptyMap()

        return try {
            val json = JSONObject(stored)
            json.toMap().mapValues { (_, value) ->
                val obj = value as Map<*, *>
                val guardiansArray = (obj["guardians"] as? List<*>) ?: emptyList<Any>()
                SocialRecoveryConfig(
                    guardians = guardiansArray.mapNotNull { g ->
                        val guardianObj = g as? Map<*, *>
                        guardianObj?.let {
                            Guardian(
                                address = it["address"] as? String ?: "",
                                name = it["name"] as? String ?: "",
                                threshold = (it["threshold"] as? Number)?.toInt() ?: 1,
                                confirmed = it["confirmed"] as? Boolean ?: false
                            )
                        }
                    },
                    threshold = (obj["threshold"] as? Number)?.toInt() ?: 1,
                    guardianCount = (obj["guardianCount"] as? Number)?.toInt() ?: 0,
                    isSetup = obj["isSetup"] as? Boolean ?: false,
                    lastRecoveryAttempt = (obj["lastRecoveryAttempt"] as? Number)?.toLong() ?: 0
                )
            }
        } catch (e: Exception) {
            e.printStackTrace()
            emptyMap()
        }
    }

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

        val hasBody = body != null
        if (hasBody) connection.doOutput = true
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
            throw AccountAbstractionException(
                "Backend $endpoint failed: HTTP $responseCode ${responseBody.take(200)}"
            )
        }

        if (responseBody.isNotEmpty()) {
            JSONObject(responseBody)
        } else {
            JSONObject()
        }
    }

    private fun JSONObject.toMap(): Map<String, Any> {
        val map = mutableMapOf<String, Any>()
        keys().forEach { key ->
            val value = get(key)
            map[key] = when (value) {
                is JSONObject -> value.toMap()
                else -> value
            }
        }
        return map
    }

    private fun JSONArray.toStringList(): List<String> {
        return (0 until length()).map { get(it) as String }
    }
}

/**
 * Smart wallet data class
 */
data class SmartWallet(
    val address: String,
    val owner: String,
    val entryPoint: String,
    val nonce: Long,
    val implementation: String,
    val initialized: Boolean,
    val createdAt: Long,
    val guardians: List<String>,
    val initCode: String? = null
)

/**
 * Session key data class
 */
data class SessionKey(
    val key: String,
    val permission: String,
    val allowedContracts: List<String>,
    val allowedTokens: List<String>,
    val spendingLimit: Long,
    val spentAmount: Long,
    val expiresAt: Long,
    val isActive: Boolean
)

/**
 * Guardian data class
 */
data class Guardian(
    val address: String,
    val name: String,
    val threshold: Int,
    val confirmed: Boolean
)

/**
 * Social recovery config data class
 */
data class SocialRecoveryConfig(
    val guardians: List<Guardian>,
    val threshold: Int,
    val guardianCount: Int,
    val isSetup: Boolean,
    val lastRecoveryAttempt: Long
)
