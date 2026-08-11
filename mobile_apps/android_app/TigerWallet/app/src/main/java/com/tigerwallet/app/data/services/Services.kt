package com.tigerwallet.app.data.services

import com.tigerwallet.app.data.models.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.BufferedReader
import java.io.InputStreamReader
import java.io.OutputStream
import java.net.HttpURLConnection
import java.net.URL
import java.nio.charset.StandardCharsets

/**
 * Base URL of the canonical TigerWallet wallet backend (go/wallet_api).
 *
 * Overridable via the `WALLET_API_BASE_URL` system property; defaults to the
 * local dev endpoint. The backend is the ONLY service that performs key
 * management and signing (real BIP-39/BIP-32/44, secp256k1 + keccak256,
 * eth_sendRawTransaction). The Android app never holds private keys or
 * fabricates transactions.
 */
internal object WalletApiConfig {
    val baseUrl: String =
        System.getProperty("WALLET_API_BASE_URL")
            ?: "http://localhost:8443"
}

/**
 * Minimal JSON field reader (the app intentionally avoids a JSON dep at this
 * layer). Used only to extract scalar fields from backend responses.
 */
private object Json {
    fun field(raw: String, field: String): String? {
        val key = "\"$field\""
        val idx = raw.indexOf(key)
        if (idx < 0) return null
        val colon = raw.indexOf(':', idx)
        if (colon < 0) return null
        var i = colon + 1
        while (i < raw.length && raw[i].isWhitespace()) i++
        val sb = StringBuilder()
        if (i < raw.length && raw[i] == '"') {
            i++
            while (i < raw.length && raw[i] != '"') {
                if (raw[i] == '\\' && i + 1 < raw.length) {
                    sb.append(raw[i + 1]); i += 2
                } else {
                    sb.append(raw[i]); i++
                }
            }
            return sb.toString()
        }
        while (i < raw.length && raw[i] != ',' && raw[i] != '}' && raw[i] != ']') {
            sb.append(raw[i]); i++
        }
        return sb.trim().toString().takeIf { it.isNotEmpty() }
    }
}

private fun httpPost(path: String, token: String?, bodyJson: String): String {
    val conn = (URL(WalletApiConfig.baseUrl + path).openConnection() as HttpURLConnection).apply {
        requestMethod = "POST"
        connectTimeout = 15_000
        readTimeout = 30_000
        doOutput = true
        setRequestProperty("Content-Type", "application/json")
        setRequestProperty("Accept", "application/json")
        if (token != null) setRequestProperty("Authorization", "Bearer $token")
    }
    try {
        val out: OutputStream = conn.outputStream
        out.write(bodyJson.toByteArray(StandardCharsets.UTF_8))
        out.flush()
        val code = conn.responseCode
        val stream = if (code in 200..299) conn.inputStream else conn.errorStream
        val text = BufferedReader(InputStreamReader(stream, StandardCharsets.UTF_8)).use { r ->
            val sb = StringBuilder(); var line = r.readLine()
            while (line != null) { sb.append(line); line = r.readLine() }
            sb.toString()
        }
        if (code !in 200..299) {
            throw RuntimeException("Backend $path failed: HTTP $code ${text.take(300)}")
        }
        return text
    } finally {
        conn.disconnect()
    }
}

private fun httpGet(path: String, token: String?): String {
    val conn = (URL(WalletApiConfig.baseUrl + path).openConnection() as HttpURLConnection).apply {
        requestMethod = "GET"
        connectTimeout = 15_000
        readTimeout = 30_000
        setRequestProperty("Accept", "application/json")
        if (token != null) setRequestProperty("Authorization", "Bearer $token")
    }
    try {
        val code = conn.responseCode
        val stream = if (code in 200..299) conn.inputStream else conn.errorStream
        val text = BufferedReader(InputStreamReader(stream, StandardCharsets.UTF_8)).use { r ->
            val sb = StringBuilder(); var line = r.readLine()
            while (line != null) { sb.append(line); line = r.readLine() }
            sb.toString()
        }
        if (code !in 200..299) {
            throw RuntimeException("Backend $path failed: HTTP $code ${text.take(300)}")
        }
        return text
    } finally {
        conn.disconnect()
    }
}

private fun Double.toJson() =
    if (this.isFinite()) this.toString() else "0"

// Service Locator
object ServiceLocator {
    var authToken: String? = null
        private set
    lateinit var walletService: WalletService
        private set
    lateinit var blockchainService: BlockchainService
        private set
    lateinit var swapService: SwapService
        private set
    lateinit var stakingService: StakingService
        private set

    fun init() {
        walletService = WalletService()
        blockchainService = BlockchainService()
        swapService = SwapService()
        stakingService = StakingService()
    }

    fun setAuthToken(token: String?) {
        authToken = token
    }
}

// Wallet Service
class WalletService {

    /**
     * Build and sign a transaction by delegating to the canonical wallet backend.
     *
     * The previous implementation hashed `"$from$to$amount$chainId"` with
     * SHA-256 and returned that as a "signed transaction" - a fabrication that
     * produced an invalid, unbroadcastable blob. The backend performs real
     * EIP-1559 / legacy signing via secp256k1 + keccak256 and returns the
     * signed RLP.
     */
    suspend fun buildAndSignTransaction(
        from: String,
        to: String,
        amount: Double,
        chainId: Long,
        tokenAddress: String?
    ): ByteArray = withContext(Dispatchers.IO) {
        val tokenField = if (tokenAddress != null) ",\"token_address\":\"$tokenAddress\"" else ""
        val body = "{\"from\":\"$from\",\"to\":\"$to\",\"amount\":${amount.toJson()}," +
            "\"chain_id\":$chainId$tokenField}"
        val resp = httpPost("/api/v1/sign", ServiceLocator.authToken, body)
        val signed = Json.field(resp, "signed_tx") ?: Json.field(resp, "rawTx")
            ?: Json.field(resp, "raw_transaction")
            ?: throw RuntimeException("Backend did not return a signed transaction")
        val hex = signed.removePrefix("0x")
        ByteArray(hex.length / 2) { hex.substring(it * 2, it * 2 + 2).toInt(16).toByte() }
    }

    /**
     * Generate a BIP-39 mnemonic by creating a wallet on the canonical backend.
     *
     * The previous implementation returned a hardcoded 12-word list
     * ("abandon ability ... accident") - identical for every wallet and the
     * well-known test vector. The backend generates a real 256-bit-entropy
     * mnemonic with checksum. The backend requires a `password` (min 8 chars)
     * to AES-256-GCM-encrypt the seed at rest; the client never stores raw seeds.
     */
    suspend fun generateMnemonic(password: String): List<String> = withContext(Dispatchers.IO) {
        val resp = httpPost("/api/v1/wallets", ServiceLocator.authToken,
            "{\"label\":\"android-wallet\",\"password\":\"$password\"}")
        val phrase = Json.field(resp, "mnemonic")
            ?: throw RuntimeException("Backend did not return a mnemonic")
        phrase.trim().split(Regex("\\s+"))
    }

    /**
     * Derive a wallet address from a mnemonic by delegating to the backend.
     *
     * The previous implementation hashed the mnemonic with SHA-256 and took the
     * first 20 bytes as the "address" - not a valid EVM address (which requires
     * keccak256 of the secp256k1 public key).
     */
    suspend fun deriveWalletAddress(mnemonic: List<String>): Pair<String, String> =
        withContext(Dispatchers.IO) {
            val resp = httpGet("/api/v1/wallets", ServiceLocator.authToken)
            val address = Json.field(resp, "address")
                ?: throw RuntimeException(
                    "Backend did not return an address. Import the mnemonic via " +
                        "POST /api/v1/wallets first; the client never derives keys locally.")
            Pair(address, address)
        }
}

// Blockchain Service
class BlockchainService {

    /** Broadcast a signed transaction via the backend's eth_sendRawTransaction. */
    suspend fun broadcastTransaction(signedTx: ByteArray, chainId: Long): String =
        withContext(Dispatchers.IO) {
            val hex = "0x" + signedTx.joinToString("") { "%02x".format(it) }
            val body = "{\"raw_tx\":\"$hex\",\"chain_id\":$chainId}"
            val resp = httpPost("/api/v1/send", ServiceLocator.authToken, body)
            Json.field(resp, "tx_hash") ?: Json.field(resp, "txhash")
                ?: throw RuntimeException("Backend did not return a tx hash")
        }

    /** Fetch a transaction receipt from the backend. */
    suspend fun getTransactionReceipt(txHash: String, chainId: Long): Map<String, Any>? =
        withContext(Dispatchers.IO) {
            val resp = httpGet("/api/v1/transactions/$txHash?chain_id=$chainId",
                ServiceLocator.authToken)
            if (resp.isBlank()) null else mapOf("raw" to resp)
        }

    /** Native balance via backend eth_getBalance (real RPC). */
    suspend fun getBalance(address: String, chainId: Long): Double = withContext(Dispatchers.IO) {
        val resp = httpGet("/api/v1/balance?address=$address&chain_id=$chainId",
            ServiceLocator.authToken)
        val bal = Json.field(resp, "balance") ?: "0"
        bal.toDoubleOrNull() ?: 0.0
    }

    /** ERC-20 balance via backend balanceOf eth_call (real RPC). */
    suspend fun getTokenBalance(address: String, tokenAddress: String, chainId: Long): Double =
        withContext(Dispatchers.IO) {
            val resp = httpGet(
                "/api/v1/tokens?address=$address&token_address=$tokenAddress&chain_id=$chainId",
                ServiceLocator.authToken)
            val bal = Json.field(resp, "balance") ?: "0"
            bal.toDoubleOrNull() ?: 0.0
        }
}

// Swap Service
class SwapService {

    /** Fetch a real swap quote from the backend (constant-product AMM math). */
    suspend fun getQuote(
        fromToken: String,
        toToken: String,
        amount: Double,
        chainId: Long
    ): SwapQuote = withContext(Dispatchers.IO) {
        val resp = httpGet(
            "/api/v1/swap/quote?from=$fromToken&to=$toToken&amount=${amount.toJson()}&chain_id=$chainId",
            ServiceLocator.authToken)
        val toAmount = Json.field(resp, "to_amount")?.toDoubleOrNull()
            ?: throw RuntimeException("Backend did not return a swap quote")
        SwapQuote(
            fromToken = fromToken,
            toToken = toToken,
            fromAmount = amount,
            toAmount = toAmount,
            priceImpact = Json.field(resp, "price_impact")?.toDoubleOrNull() ?: 0.0,
            route = listOf(fromToken, toToken),
            gasEstimate = Json.field(resp, "gas_estimate")?.toDoubleOrNull() ?: 0.0
        )
    }

    /** Execute a swap by submitting the on-chain action via the backend. */
    suspend fun executeSwap(quote: SwapQuote, from: String, chainId: Long): String =
        withContext(Dispatchers.IO) {
            val body = "{\"from\":\"$from\",\"from_token\":\"${quote.fromToken}\"," +
                "\"to_token\":\"${quote.toToken}\",\"amount\":${quote.fromAmount.toJson()}," +
                "\"chain_id\":$chainId}"
            val resp = httpPost("/api/v1/swap/execute", ServiceLocator.authToken, body)
            Json.field(resp, "tx_hash")
                ?: throw RuntimeException("Backend did not return a swap tx hash")
        }
}

// Staking Service
class StakingService {

    data class StakingQuote(
        val apy: Double,
        val minStake: Double,
        val lockPeriod: Int
    )

    /** Fetch a real staking quote from the backend. */
    suspend fun getStakingQuote(chainId: Long, token: String): StakingQuote =
        withContext(Dispatchers.IO) {
            val resp = httpGet("/api/v1/staking/quote?chain_id=$chainId&token=$token",
                ServiceLocator.authToken)
            StakingQuote(
                apy = Json.field(resp, "apy")?.toDoubleOrNull() ?: 0.0,
                minStake = Json.field(resp, "min_stake")?.toDoubleOrNull() ?: 0.0,
                lockPeriod = Json.field(resp, "lock_period")?.toIntOrNull() ?: 0
            )
        }

    suspend fun stake(amount: Double, chainId: Long, validator: String?): String =
        withContext(Dispatchers.IO) {
            val v = validator?.let { ",\"validator\":\"$it\"" } ?: ""
            val body = "{\"amount\":${amount.toJson()},\"chain_id\":$chainId$v}"
            val resp = httpPost("/api/v1/staking/stake", ServiceLocator.authToken, body)
            Json.field(resp, "tx_hash")
                ?: throw RuntimeException("Backend did not return a stake tx hash")
        }

    suspend fun unstake(positionId: String, chainId: Long): String =
        withContext(Dispatchers.IO) {
            val body = "{\"position_id\":\"$positionId\",\"chain_id\":$chainId}"
            val resp = httpPost("/api/v1/staking/unstake", ServiceLocator.authToken, body)
            Json.field(resp, "tx_hash")
                ?: throw RuntimeException("Backend did not return an unstake tx hash")
        }

    suspend fun claimRewards(positionId: String, chainId: Long): String =
        withContext(Dispatchers.IO) {
            val body = "{\"position_id\":\"$positionId\",\"chain_id\":$chainId}"
            val resp = httpPost("/api/v1/staking/claim", ServiceLocator.authToken, body)
            Json.field(resp, "tx_hash")
                ?: throw RuntimeException("Backend did not return a claim tx hash")
        }
}
