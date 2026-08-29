package com.tigermaster

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import org.json.JSONObject

/**
 * MasterWallet feature screens. Every screen reads ONLY the StateFlows fed by
 * the canonical backend (:8450) via MasterWalletViewModel.loadFeature — no
 * mock data, no fabricated rows. Theme: all composables inherit the wrapping
 * MasterWalletTheme (light/dark) from MainActivity.
 */

data class FeatureEntry(val id: String, val label: String, val icon: String)

val MASTER_FEATURES = listOf(
    FeatureEntry("subwallets", "Sub-Wallets", "🗂️"),
    FeatureEntry("send", "Send", "💸"),
    FeatureEntry("ops", "Auto-Sign Ops", "🛠️"),
    FeatureEntry("treasury", "Treasury", "🏦"),
    FeatureEntry("multisig", "Multisig", "🔐"),
    FeatureEntry("autosign", "Auto-Sign", "🔑"),
    FeatureEntry("fees", "Fees", "💸"),
    FeatureEntry("policies", "Policies", "📏"),
    FeatureEntry("users", "Users", "👥"),
    FeatureEntry("chains", "Chains", "⛓️"),
    FeatureEntry("tokens", "Tokens", "🪙"),
    FeatureEntry("flags", "Feature Flags", "🚩"),
    FeatureEntry("webhooks", "Webhooks & Alerts", "🔔"),
    FeatureEntry("audit", "Audit Log", "🧾"),
    FeatureEntry("analytics", "Analytics", "📈"),
    FeatureEntry("passkeys", "Passkeys", "🪪"),
    FeatureEntry("withdraw", "Withdraw", "📤"),
)

@Composable
fun MoreScreen(viewModel: MasterWalletViewModel, modifier: Modifier = Modifier) {
    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Text("All Features", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(12.dp))
        MASTER_FEATURES.chunked(2).forEach { rowItems ->
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                rowItems.forEach { f ->
                    Card(
                        modifier = Modifier.weight(1f).padding(vertical = 6.dp),
                        onClick = { viewModel.openFeature(f.id); viewModel.loadFeature(f.id) }
                    ) {
                        Column(Modifier.padding(16.dp)) {
                            Text(f.icon, fontSize = 22.sp)
                            Text(f.label, fontWeight = FontWeight.Bold, fontSize = 14.sp)
                        }
                    }
                }
                if (rowItems.size == 1) Spacer(Modifier.weight(1f))
            }
        }
    }
}

@Composable
fun FeatureHostScreen(viewModel: MasterWalletViewModel, feature: String, modifier: Modifier = Modifier) {
    val entry = MASTER_FEATURES.firstOrNull { it.id == feature }
    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextButton(onClick = { viewModel.closeFeature() }) { Text("← Back") }
            Text("${entry?.icon ?: ""} ${entry?.label ?: feature}", fontSize = 20.sp, fontWeight = FontWeight.Bold)
        }
        Spacer(Modifier.height(8.dp))
        when (feature) {
            "subwallets" -> SubWalletsScreen(viewModel)
            "send" -> SendScreen(viewModel)
            "ops" -> AutoSignOpsScreen(viewModel)
            "treasury" -> TreasuryScreen(viewModel)
            "multisig" -> MultisigScreen(viewModel)
            "autosign" -> AutoSignScreen(viewModel)
            "fees" -> FeesScreen(viewModel)
            "policies" -> PoliciesScreen(viewModel)
            "users" -> UsersScreen(viewModel)
            "chains" -> ChainsScreen(viewModel)
            "tokens" -> TokensScreen(viewModel)
            "flags" -> FlagsScreen(viewModel)
            "webhooks" -> WebhooksScreen(viewModel)
            "audit" -> AuditScreen(viewModel)
            "analytics" -> AnalyticsScreen(viewModel)
            "passkeys" -> PasskeysScreen(viewModel)
            "withdraw" -> WithdrawScreen(viewModel)
        }
    }
}

// ---- Shared building blocks ------------------------------------------------

private fun JSONObject.str(vararg keys: String): String {
    for (k in keys) {
        val v = opt(k)
        if (v != null && v != JSONObject.NULL && v.toString().isNotBlank()) return v.toString()
    }
    return ""
}

@Composable
private fun JsonRowCard(title: String, subtitle: String, actions: (@Composable RowScope.() -> Unit)? = null) {
    Card(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
        Row(Modifier.padding(12.dp).fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(title, fontWeight = FontWeight.Bold, fontSize = 14.sp)
                if (subtitle.isNotBlank()) Text(subtitle, fontSize = 11.sp, color = Color.Gray)
            }
            actions?.invoke(this)
        }
    }
}

@Composable
private fun FieldRow(fields: List<Pair<String, (String) -> Unit>>, values: List<String>) {
    fields.forEachIndexed { i, (hint, onChange) ->
        OutlinedTextField(
            value = values[i],
            onValueChange = onChange,
            label = { Text(hint) },
            modifier = Modifier.fillMaxWidth().padding(vertical = 2.dp),
            singleLine = true
        )
    }
}

@Composable
private fun ErrorText(viewModel: MasterWalletViewModel) {
    val err by viewModel.error.collectAsState()
    if (!err.isNullOrBlank()) Text(err!!, color = MaterialTheme.colorScheme.error, fontSize = 12.sp)
}

// ---- Sub-Wallets ------------------------------------------------------------

@Composable
fun SubWalletsScreen(viewModel: MasterWalletViewModel) {
    val subs by viewModel.subWalletList.collectAsState()
    var sid by remember { mutableStateOf("") }
    var to by remember { mutableStateOf("") }
    var amount by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        subs.forEach { s ->
            JsonRowCard(
                "${s.str("label", "name").ifBlank { "Sub-wallet" }} — ${s.str("status").ifBlank { "active" }}",
                "${s.str("address", "id")} ${s.str("balance")}"
            ) {
                TextButton(onClick = { sid = s.str("id") }) { Text("Use") }
            }
        }
        if (subs.isEmpty()) Text("No sub-wallets.", fontSize = 12.sp, color = Color.Gray)
        Text("Transfer from sub-wallet", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        FieldRow(listOf(
            "Sub-wallet ID" to { v: String -> sid = v },
            "To address" to { v: String -> to = v },
            "Amount" to { v: String -> amount = v },
            "Wallet password" to { v: String -> password = v }
        ), listOf(sid, to, amount, password))
        Button(onClick = { viewModel.transferFromSubWallet(sid, to, amount, password) }, modifier = Modifier.fillMaxWidth()) { Text("Transfer") }
        ErrorText(viewModel)
    }
}

// ---- Send (sign + broadcast) --------------------------------------------------

@Composable
fun SendScreen(viewModel: MasterWalletViewModel) {
    var to by remember { mutableStateOf("") }
    var amount by remember { mutableStateOf("") }
    var token by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var result by remember { mutableStateOf<String?>(null) }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        FieldRow(listOf(
            "To address" to { v: String -> to = v },
            "Amount (e.g. 0.5)" to { v: String -> amount = v },
            "Token contract (empty = native)" to { v: String -> token = v },
            "Wallet password" to { v: String -> password = v }
        ), listOf(to, amount, token, password))
        Button(onClick = {
            viewModel.sendSigned(to, amount, token, password) { result = it; if (it.startsWith("Transaction")) password = "" }
        }, modifier = Modifier.fillMaxWidth()) { Text("Sign & broadcast") }
        result?.let { Text(it, fontSize = 12.sp, modifier = Modifier.padding(top = 8.dp)) }
    }
}

// ---- Auto-Sign Ops ------------------------------------------------------------

@Composable
fun AutoSignOpsScreen(viewModel: MasterWalletViewModel) {
    var chkType by remember { mutableStateOf("") }
    var chkValue by remember { mutableStateOf("") }
    var chkResult by remember { mutableStateOf<String?>(null) }
    var mnemonic by remember { mutableStateOf("") }
    var chainId by remember { mutableStateOf("1") }
    var chainType by remember { mutableStateOf("evm") }
    var txType by remember { mutableStateOf("send") }
    var to by remember { mutableStateOf("") }
    var value by remember { mutableStateOf("") }
    var tokenAddr by remember { mutableStateOf("") }
    var rpTo by remember { mutableStateOf("") }
    var rpAmount by remember { mutableStateOf("") }
    var rpPassword by remember { mutableStateOf("") }
    var rpWid by remember { mutableStateOf("") }
    var result by remember { mutableStateOf<String?>(null) }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text("Check auto-sign policy", fontWeight = FontWeight.Bold)
        FieldRow(listOf("Tx type" to { v: String -> chkType = v }, "Value" to { v: String -> chkValue = v }), listOf(chkType, chkValue))
        Button(onClick = { viewModel.checkAutoSignPolicyNow(chkType, chkValue) { chkResult = it } }, modifier = Modifier.fillMaxWidth()) { Text("Check") }
        chkResult?.let { Text(it, fontSize = 12.sp, modifier = Modifier.padding(top = 4.dp)) }

        Text("Auto-sign transaction (24-word seed)", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 12.dp))
        FieldRow(listOf(
            "24-word mnemonic" to { v: String -> mnemonic = v },
            "Chain ID" to { v: String -> chainId = v },
            "Chain type" to { v: String -> chainType = v },
            "Tx type" to { v: String -> txType = v },
            "To address" to { v: String -> to = v },
            "Value" to { v: String -> value = v },
            "Token contract (optional)" to { v: String -> tokenAddr = v }
        ), listOf(mnemonic, chainId, chainType, txType, to, value, tokenAddr))
        Row {
            Button(onClick = {
                viewModel.autoSignTransactionNow(mnemonic, chainId.toLongOrNull() ?: 1, chainType, txType, to, value, tokenAddr) {
                    result = it; if (it.startsWith("Transaction")) mnemonic = ""
                }
            }, modifier = Modifier.weight(1f)) { Text("Auto-sign tx") }
            Spacer(Modifier.width(8.dp))
            OutlinedButton(onClick = {
                viewModel.userWalletAutoSign(mnemonic, chainId.toLongOrNull() ?: 1, chainType, txType) {
                    result = it; mnemonic = ""
                }
            }, modifier = Modifier.weight(1f)) { Text("UW auto-sign") }
        }
        result?.let { Text(it, fontSize = 12.sp, modifier = Modifier.padding(top = 4.dp)) }

        Text("Revenue payout (SuperAdmin co-sign required)", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 12.dp))
        FieldRow(listOf(
            "Destination address" to { v: String -> rpTo = v },
            "Amount" to { v: String -> rpAmount = v },
            "Wallet password" to { v: String -> rpPassword = v },
            "Withdrawal ID (co-signed)" to { v: String -> rpWid = v }
        ), listOf(rpTo, rpAmount, rpPassword, rpWid))
        Button(onClick = {
            viewModel.revenuePayout(rpTo, rpAmount, rpPassword, rpWid) { result = it; rpPassword = "" }
        }, modifier = Modifier.fillMaxWidth()) { Text("Execute payout") }
        ErrorText(viewModel)
    }
}

// ---- Treasury ---------------------------------------------------------------

@Composable
fun TreasuryScreen(viewModel: MasterWalletViewModel) {
    val treasury by viewModel.treasury.collectAsState()
    val txs by viewModel.treasuryTxs.collectAsState()
    var to by remember { mutableStateOf("") }
    var amount by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var sweepId by remember { mutableStateOf("") }
    var sweepPw by remember { mutableStateOf("") }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Card(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
            Column(Modifier.padding(12.dp)) {
                Text("Treasury Overview", fontWeight = FontWeight.Bold)
                if (treasury == null) Text("No treasury data.", fontSize = 12.sp, color = Color.Gray)
                else {
                    Text("Address: ${treasury!!.str("address", "treasury_address")}", fontSize = 12.sp)
                    Text("Balance: ${treasury!!.str("balance", "balance_wei", "total_balance")}", fontSize = 12.sp)
                }
            }
        }
        Text("Transfer", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        FieldRow(listOf("To" to { v: String -> to = v }, "Amount" to { v: String -> amount = v }, "Password" to { v: String -> password = v }), listOf(to, amount, password))
        Button(onClick = { viewModel.treasuryTransfer(to, amount, password) }, modifier = Modifier.fillMaxWidth()) { Text("Transfer") }
        Text("Sweep sub-wallet", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        FieldRow(listOf("Sub-wallet ID" to { v: String -> sweepId = v }, "Password" to { v: String -> sweepPw = v }), listOf(sweepId, sweepPw))
        Button(onClick = { viewModel.treasurySweep(sweepId, sweepPw) }, modifier = Modifier.fillMaxWidth()) { Text("Sweep") }
        Text("Treasury Transactions", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        txs.forEach { t ->
            JsonRowCard("${t.str("tx_type")} — ${t.str("amount")}", "${t.str("to_address", "to")} · ${t.str("status")}")
        }
        ErrorText(viewModel)
    }
}

// ---- Multisig ---------------------------------------------------------------

@Composable
fun MultisigScreen(viewModel: MasterWalletViewModel) {
    val wallets by viewModel.multisigWallets.collectAsState()
    val txs by viewModel.multisigTxs.collectAsState()
    var name by remember { mutableStateOf("") }
    var owners by remember { mutableStateOf("") }
    var threshold by remember { mutableStateOf("") }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text("Create Multisig Wallet", fontWeight = FontWeight.Bold)
        FieldRow(listOf("Name" to { v: String -> name = v }, "Owners (comma-separated)" to { v: String -> owners = v }, "Threshold" to { v: String -> threshold = v }), listOf(name, owners, threshold))
        Button(onClick = { viewModel.createMultisig(name, owners, threshold.toIntOrNull() ?: 0) }, modifier = Modifier.fillMaxWidth()) { Text("Create") }
        Text("Multisig Wallets", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        wallets.forEach { w ->
            JsonRowCard(
                "${w.str("name")} (${w.str("threshold")} sig)",
                w.str("address", "id")
            ) {
                TextButton(onClick = { viewModel.loadMultisigTxs(w.str("id")) }) { Text("Txs") }
            }
        }
        if (txs.isNotEmpty()) {
            Text("Multisig Transactions", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
            txs.forEach { t ->
                JsonRowCard("${t.str("to")} — ${t.str("value", "amount")}", t.str("status")) {
                    TextButton(onClick = { viewModel.signMultisig(t.str("id")) }) { Text("Sign") }
                    TextButton(onClick = { viewModel.executeMultisig(t.str("id")) }) { Text("Exec") }
                }
            }
        }
        ErrorText(viewModel)
    }
}

// ---- Auto-Sign --------------------------------------------------------------

@Composable
fun AutoSignScreen(viewModel: MasterWalletViewModel) {
    val rules by viewModel.autoSignRules.collectAsState()
    val logs by viewModel.autoSignLogs.collectAsState()
    val policy by viewModel.autoSignPolicy.collectAsState()
    var name by remember { mutableStateOf("") }
    var maxAmount by remember { mutableStateOf("") }
    var chain by remember { mutableStateOf("") }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Card(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
            Column(Modifier.padding(12.dp)) {
                Text("Daemon Policy", fontWeight = FontWeight.Bold)
                val enabled = policy?.optBoolean("enabled") == true
                Text(if (enabled) "Auto-signer ENABLED" else "Auto-signer disabled", fontSize = 12.sp)
                Text("Max auto value: ${policy?.str("max_auto_value_wei") ?: "—"} wei", fontSize = 12.sp)
                Row {
                    TextButton(onClick = { viewModel.setAutoSignPolicyEnabled(true) }) { Text("Enable") }
                    TextButton(onClick = { viewModel.setAutoSignPolicyEnabled(false) }) { Text("Disable") }
                }
            }
        }
        Text("Add Rule", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        FieldRow(listOf("Rule name" to { v: String -> name = v }, "Max amount" to { v: String -> maxAmount = v }, "Chain" to { v: String -> chain = v }), listOf(name, maxAmount, chain))
        Button(onClick = { viewModel.createAutoSignRule(name, maxAmount, chain, true) }, modifier = Modifier.fillMaxWidth()) { Text("Add rule") }
        Text("Rules", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        rules.forEach { r ->
            JsonRowCard(r.name, "${r.chain} · max ${r.maxAmount} · ${if (r.enabled) "active" else "off"}") {
                TextButton(onClick = { viewModel.deleteAutoSignRule(r.id) }) { Text("Delete", color = MaterialTheme.colorScheme.error) }
            }
        }
        Text("Auto-Sign Logs", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        logs.forEach { l ->
            JsonRowCard(l.str("action", "status"), "${l.str("tx_hash")} ${l.str("created_at")}")
        }
        ErrorText(viewModel)
    }
}

// ---- Fees -------------------------------------------------------------------

@Composable
fun FeesScreen(viewModel: MasterWalletViewModel) {
    val fees by viewModel.fees.collectAsState()
    var feeType by remember { mutableStateOf("") }
    var pct by remember { mutableStateOf("") }
    var fixed by remember { mutableStateOf("") }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text("Add Fee", fontWeight = FontWeight.Bold)
        FieldRow(listOf("Fee type" to { v: String -> feeType = v }, "Fee percentage" to { v: String -> pct = v }, "Fee fixed (wei)" to { v: String -> fixed = v }), listOf(feeType, pct, fixed))
        Button(onClick = { viewModel.createFee(feeType, pct.toDoubleOrNull() ?: 0.0, fixed) }, modifier = Modifier.fillMaxWidth()) { Text("Add") }
        Spacer(Modifier.height(8.dp))
        fees.forEach { f ->
            val active = f.optBoolean("is_active")
            JsonRowCard(
                "${f.str("fee_type")} — ${f.str("fee_percentage")}%",
                if (f.str("fee_fixed").isNotBlank()) "fixed ${f.str("fee_fixed")}" else ""
            ) {
                TextButton(onClick = { viewModel.toggleFee(f.str("id"), !active) }) { Text(if (active) "Off" else "On") }
                TextButton(onClick = { viewModel.deleteFee(f.str("id")) }) { Text("Del", color = MaterialTheme.colorScheme.error) }
            }
        }
        ErrorText(viewModel)
    }
}

// ---- Policies ---------------------------------------------------------------

@Composable
fun PoliciesScreen(viewModel: MasterWalletViewModel) {
    val policies by viewModel.policies.collectAsState()
    var name by remember { mutableStateOf("") }
    var type by remember { mutableStateOf("") }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text("Add Policy", fontWeight = FontWeight.Bold)
        FieldRow(listOf("Policy name" to { v: String -> name = v }, "Policy type" to { v: String -> type = v }), listOf(name, type))
        Button(onClick = { viewModel.createPolicy(name, type) }, modifier = Modifier.fillMaxWidth()) { Text("Add") }
        Spacer(Modifier.height(8.dp))
        policies.forEach { p ->
            JsonRowCard("${p.str("name")} (${p.str("policy_type")})", "priority ${p.str("priority")} · ${if (p.optBoolean("is_active")) "active" else "off"}") {
                TextButton(onClick = { viewModel.deletePolicy(p.str("id")) }) { Text("Del", color = MaterialTheme.colorScheme.error) }
            }
        }
        ErrorText(viewModel)
    }
}

// ---- Users ------------------------------------------------------------------

@Composable
fun UsersScreen(viewModel: MasterWalletViewModel) {
    val users by viewModel.users.collectAsState()
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var name by remember { mutableStateOf("") }
    var role by remember { mutableStateOf("") }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text("Add User", fontWeight = FontWeight.Bold)
        FieldRow(listOf("Email" to { v: String -> email = v }, "Password (min 8)" to { v: String -> password = v }, "Name" to { v: String -> name = v }, "Role" to { v: String -> role = v }), listOf(email, password, name, role))
        Button(onClick = { viewModel.createUser(email, password, name, role) }, modifier = Modifier.fillMaxWidth()) { Text("Add") }
        Spacer(Modifier.height(8.dp))
        users.forEach { u ->
            JsonRowCard("${u.name} (${u.role})", u.email) {
                TextButton(onClick = { viewModel.deleteUser(u.id) }) { Text("Del", color = MaterialTheme.colorScheme.error) }
            }
        }
        ErrorText(viewModel)
    }
}

// ---- Chains -----------------------------------------------------------------

@Composable
fun ChainsScreen(viewModel: MasterWalletViewModel) {
    val evm by viewModel.evmChains.collectAsState()
    val nonEvm by viewModel.nonEvmChains.collectAsState()
    var eId by remember { mutableStateOf("") }
    var eName by remember { mutableStateOf("") }
    var eRpc by remember { mutableStateOf("") }
    var eSym by remember { mutableStateOf("") }
    var nId by remember { mutableStateOf("") }
    var nName by remember { mutableStateOf("") }
    var nType by remember { mutableStateOf("") }
    var nRpc by remember { mutableStateOf("") }
    var nPath by remember { mutableStateOf("") }
    var editingEvm by remember { mutableStateOf<Long?>(null) }
    var editingNonEvm by remember { mutableStateOf<Long?>(null) }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text(if (editingEvm != null) "Edit EVM Chain" else "Add EVM Chain", fontWeight = FontWeight.Bold)
        FieldRow(listOf("Chain ID" to { v: String -> eId = v }, "Name" to { v: String -> eName = v }, "RPC URL" to { v: String -> eRpc = v }, "Symbol" to { v: String -> eSym = v }), listOf(eId, eName, eRpc, eSym))
        Button(onClick = {
            val editId = editingEvm
            if (editId != null) { viewModel.updateEvmChain(editId, eName, eRpc, eSym); editingEvm = null }
            else viewModel.addEvmChain(eId.toLongOrNull() ?: 0, eName, eRpc, eSym)
        }, modifier = Modifier.fillMaxWidth()) { Text(if (editingEvm != null) "Save EVM chain" else "Add EVM chain") }
        evm.forEach { c ->
            JsonRowCard("${c.str("name")} (${c.str("chain_id")})", c.str("rpc_url")) {
                TextButton(onClick = {
                    eId = c.str("chain_id"); eName = c.str("name"); eRpc = c.str("rpc_url"); eSym = c.str("symbol")
                    editingEvm = c.str("chain_id").toLongOrNull()
                }) { Text("Edit") }
                TextButton(onClick = { viewModel.removeEvmChain(c.str("chain_id").toLongOrNull() ?: 0) }) { Text("Del", color = MaterialTheme.colorScheme.error) }
            }
        }
        Text(if (editingNonEvm != null) "Edit Non-EVM Chain" else "Add Non-EVM Chain", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        FieldRow(listOf(
            "Chain ID (SLIP-44)" to { v: String -> nId = v },
            "Name" to { v: String -> nName = v },
            "Chain type" to { v: String -> nType = v },
            "RPC / node URL" to { v: String -> nRpc = v },
            "Derivation path" to { v: String -> nPath = v }
        ), listOf(nId, nName, nType, nRpc, nPath))
        Button(onClick = {
            val editId = editingNonEvm
            if (editId != null) { viewModel.updateNonEvmChain(editId, nName, nType, nRpc, nPath); editingNonEvm = null }
            else viewModel.addNonEvmChain(nId.toLongOrNull() ?: 0, nName, nType, nRpc, nPath)
        }, modifier = Modifier.fillMaxWidth()) { Text(if (editingNonEvm != null) "Save non-EVM chain" else "Add non-EVM chain") }
        nonEvm.forEach { c ->
            JsonRowCard("${c.str("name")} (${c.str("chain_type")})", "id ${c.str("chain_id")} · ${c.str("rpc_url")}") {
                TextButton(onClick = {
                    nId = c.str("chain_id"); nName = c.str("name"); nType = c.str("chain_type"); nRpc = c.str("rpc_url"); nPath = c.str("derivation_path")
                    editingNonEvm = c.str("chain_id").toLongOrNull()
                }) { Text("Edit") }
                TextButton(onClick = { viewModel.removeNonEvmChain(c.str("chain_id").toLongOrNull() ?: 0) }) { Text("Del", color = MaterialTheme.colorScheme.error) }
            }
        }
        ErrorText(viewModel)
    }
}

// ---- Tokens -----------------------------------------------------------------

@Composable
fun TokensScreen(viewModel: MasterWalletViewModel) {
    val tokens by viewModel.userTokens.collectAsState()
    var chainId by remember { mutableStateOf("") }
    var symbol by remember { mutableStateOf("") }
    var name by remember { mutableStateOf("") }
    var address by remember { mutableStateOf("") }
    var decimals by remember { mutableStateOf("18") }
    var editingToken by remember { mutableStateOf<Long?>(null) }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text(if (editingToken != null) "Edit Token" else "Add Token", fontWeight = FontWeight.Bold)
        FieldRow(listOf(
            "Chain ID" to { v: String -> chainId = v },
            "Symbol" to { v: String -> symbol = v },
            "Name" to { v: String -> name = v },
            "Contract address" to { v: String -> address = v },
            "Decimals" to { v: String -> decimals = v }
        ), listOf(chainId, symbol, name, address, decimals))
        Button(onClick = {
            val editId = editingToken
            if (editId != null) { viewModel.updateUserToken(editId, symbol, name, address, decimals.toIntOrNull() ?: 18); editingToken = null }
            else viewModel.addUserToken(chainId.toLongOrNull() ?: 0, symbol, name, address, decimals.toIntOrNull() ?: 18)
        }, modifier = Modifier.fillMaxWidth()) { Text(if (editingToken != null) "Save token" else "Add token") }
        Spacer(Modifier.height(8.dp))
        tokens.forEach { t ->
            JsonRowCard("${t.str("symbol")} — ${t.str("name")}", "chain ${t.str("chain_id")} · ${t.str("contract_address").ifBlank { "native" }}") {
                TextButton(onClick = {
                    chainId = t.str("chain_id"); symbol = t.str("symbol"); name = t.str("name"); address = t.str("contract_address"); decimals = t.str("decimals").ifBlank { "18" }
                    editingToken = t.str("id").toLongOrNull()
                }) { Text("Edit") }
                TextButton(onClick = { viewModel.removeUserToken(t.str("id").toLongOrNull() ?: 0) }) { Text("Del", color = MaterialTheme.colorScheme.error) }
            }
        }
        ErrorText(viewModel)
    }
}

// ---- Feature Flags ----------------------------------------------------------

@Composable
fun FlagsScreen(viewModel: MasterWalletViewModel) {
    val flags by viewModel.featureFlags.collectAsState()
    var key by remember { mutableStateOf("") }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text("Add Flag", fontWeight = FontWeight.Bold)
        FieldRow(listOf("Flag key" to { v: String -> key = v }), listOf(key))
        Button(onClick = { viewModel.addFeatureFlag(key) }, modifier = Modifier.fillMaxWidth()) { Text("Add") }
        Spacer(Modifier.height(8.dp))
        flags.forEach { f ->
            val enabled = f.optBoolean("is_enabled")
            JsonRowCard(f.str("flag_key") + if (enabled) " ✅" else " ⛔", f.str("description", "flag_value")) {
                TextButton(onClick = { viewModel.toggleFeatureFlag(f.str("id").toLongOrNull() ?: 0, !enabled) }) { Text(if (enabled) "Off" else "On") }
                TextButton(onClick = { viewModel.removeFeatureFlag(f.str("id").toLongOrNull() ?: 0) }) { Text("Del", color = MaterialTheme.colorScheme.error) }
            }
        }
        ErrorText(viewModel)
    }
}

// ---- Webhooks & Notifications ------------------------------------------------

@Composable
fun WebhooksScreen(viewModel: MasterWalletViewModel) {
    val webhooks by viewModel.webhooks.collectAsState()
    val notifications by viewModel.notifications.collectAsState()
    var wName by remember { mutableStateOf("") }
    var wUrl by remember { mutableStateOf("") }
    var wEvents by remember { mutableStateOf("") }
    var nType by remember { mutableStateOf("") }
    var nTitle by remember { mutableStateOf("") }
    var nMsg by remember { mutableStateOf("") }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text("Add Webhook", fontWeight = FontWeight.Bold)
        FieldRow(listOf("Name" to { v: String -> wName = v }, "URL" to { v: String -> wUrl = v }, "Events (comma-separated)" to { v: String -> wEvents = v }), listOf(wName, wUrl, wEvents))
        Button(onClick = { viewModel.createWebhook(wName, wUrl, wEvents) }, modifier = Modifier.fillMaxWidth()) { Text("Add webhook") }
        webhooks.forEach { w ->
            JsonRowCard(w.str("name"), w.str("url")) {
                TextButton(onClick = { viewModel.deleteWebhook(w.str("id")) }) { Text("Del", color = MaterialTheme.colorScheme.error) }
            }
        }
        Text("Send Notification", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        FieldRow(listOf("Type" to { v: String -> nType = v }, "Title" to { v: String -> nTitle = v }, "Message" to { v: String -> nMsg = v }), listOf(nType, nTitle, nMsg))
        Button(onClick = { viewModel.createNotification(nType, nTitle, nMsg) }, modifier = Modifier.fillMaxWidth()) { Text("Send") }
        notifications.forEach { n ->
            JsonRowCard(n.str("title", "notification_type"), "${n.str("message")} · ${n.str("created_at")}")
        }
        ErrorText(viewModel)
    }
}

// ---- Audit ------------------------------------------------------------------

@Composable
fun AuditScreen(viewModel: MasterWalletViewModel) {
    val logs by viewModel.auditLogs.collectAsState()
    Column {
        if (logs.isEmpty()) Text("No audit events.", fontSize = 12.sp, color = Color.Gray)
        LazyColumn {
            items(logs) { a ->
                JsonRowCard(a.str("action", "event"), "${a.str("actor", "user_id")} · ${a.str("details", "description")} · ${a.str("created_at", "timestamp")}")
            }
        }
    }
}

// ---- Analytics ----------------------------------------------------------------

@Composable
fun AnalyticsScreen(viewModel: MasterWalletViewModel) {
    val analytics by viewModel.analytics.collectAsState()
    Column(Modifier.verticalScroll(rememberScrollState())) {
        if (analytics.isEmpty()) Text("No analytics returned.", fontSize = 12.sp, color = Color.Gray)
        analytics.forEach { (k, v) ->
            JsonRowCard(k, v)
        }
    }
}

// ---- Passkeys -----------------------------------------------------------------

@Composable
fun PasskeysScreen(viewModel: MasterWalletViewModel) {
    val passkeys by viewModel.passkeys.collectAsState()
    val context = androidx.compose.ui.platform.LocalContext.current
    var label by remember { mutableStateOf("") }
    var result by remember { mutableStateOf<String?>(null) }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text("Register Passkey", fontWeight = FontWeight.Bold)
        FieldRow(listOf("Label (e.g. this phone)" to { v: String -> label = v }), listOf(label))
        Button(onClick = {
            viewModel.registerPasskey(context, label) { result = it; if (it.startsWith("Passkey")) label = "" }
        }, modifier = Modifier.fillMaxWidth()) { Text("Register passkey") }
        result?.let { Text(it, fontSize = 12.sp, modifier = Modifier.padding(top = 4.dp)) }
        Text("Registered Passkeys", fontWeight = FontWeight.Bold, modifier = Modifier.padding(top = 8.dp))
        if (passkeys.isEmpty()) Text("No passkeys registered.", fontSize = 12.sp, color = Color.Gray)
        passkeys.forEach { p ->
            JsonRowCard(p.str("label").ifBlank { "Passkey" }, "${p.str("credential_id", "id").take(24)} · ${p.str("created_at")}") {
                TextButton(onClick = { viewModel.deletePasskey(p.str("credential_id", "id")) }) { Text("Del", color = MaterialTheme.colorScheme.error) }
            }
        }
        ErrorText(viewModel)
    }
}

// ---- Withdraw -----------------------------------------------------------------

@Composable
fun WithdrawScreen(viewModel: MasterWalletViewModel) {
    var to by remember { mutableStateOf("") }
    var amountWei by remember { mutableStateOf("") }
    var currency by remember { mutableStateOf("") }
    var chainId by remember { mutableStateOf("1") }
    var result by remember { mutableStateOf<String?>(null) }

    Column(Modifier.verticalScroll(rememberScrollState())) {
        Text(
            "Funds never move without TigerWallet SuperAdmin two-party co-sign. This only files the request.",
            fontSize = 12.sp, color = Color.Gray
        )
        FieldRow(listOf(
            "Destination address" to { v: String -> to = v },
            "Amount (wei)" to { v: String -> amountWei = v },
            "Currency (e.g. ETH)" to { v: String -> currency = v },
            "Chain ID" to { v: String -> chainId = v }
        ), listOf(to, amountWei, currency, chainId))
        Button(onClick = {
            viewModel.requestWithdrawal(to, amountWei, currency, chainId.toLongOrNull() ?: 1) { result = it }
        }, modifier = Modifier.fillMaxWidth()) { Text("Request withdrawal") }
        result?.let { Text(it, fontSize = 12.sp, modifier = Modifier.padding(top = 8.dp)) }
    }
}
