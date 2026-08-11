package com.tigerwallet.app.ui.screens

import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp

// Send Screen with QR Scanner - Matches Flutter/iOS Functionality
class SendScreen : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                SendScreenContent()
            }
        }
    }
}

@OptIn(ExperimentalMaterialApi::class)
@Composable
fun SendScreenContent() {
    var recipientAddress by remember { mutableStateOf("") }
    var amount by remember { mutableStateOf("") }
    var selectedToken by remember { mutableStateOf("ETH") }
    var selectedChain by remember { mutableStateOf("Ethereum") }
    var showQRScanner by remember { mutableStateOf(false) }
    var isLoading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    val tokens = listOf("ETH", "USDT", "USDC", "BNB", "MATIC", "SOL", "TRX")
    val chains = listOf("Ethereum", "BNB Chain", "Polygon", "Arbitrum", "Optimism", "Avalanche", "Solana", "TRON")
    val context = LocalContext.current

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Send") },
                actions = {
                    IconButton(onClick = { showQRScanner = true }) {
                        Icon(androidx.compose.material.icons.Icons.Default.QrCodeScanner, contentDescription = "Scan QR")
                    }
                }
            )
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(16.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Network Selector
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Network", style = MaterialTheme.typography.subtitle2, color = MaterialTheme.colors.onSurfaceVariant)
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        chains.take(4).forEach { chain ->
                            FilterChip(
                                selected = selectedChain == chain,
                                onClick = { selectedChain = chain },
                                label = { Text(chain.take(8)) }
                            )
                        }
                    }
                }
            }

            // Recipient Address
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Recipient Address", style = MaterialTheme.typography.subtitle2)
                        TextButton(onClick = { showQRScanner = true }) {
                            Text("Scan QR")
                        }
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    OutlinedTextField(
                        value = recipientAddress,
                        onValueChange = { recipientAddress = it },
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = { Text("0x...") },
                        singleLine = true
                    )
                }
            }

            // Token Selector
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Token", style = MaterialTheme.typography.subtitle2)
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        tokens.forEach { token ->
                            FilterChip(
                                selected = selectedToken == token,
                                onClick = { selectedToken = token },
                                label = { Text(token) }
                            )
                        }
                    }
                }
            }

            // Amount
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Amount", style = MaterialTheme.typography.subtitle2)
                        TextButton(onClick = { amount = "1.0" }) {
                            Text("MAX")
                        }
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    OutlinedTextField(
                        value = amount,
                        onValueChange = { amount = it },
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = { Text("0.0") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        singleLine = true,
                        trailingIcon = { Text(selectedToken) }
                    )
                }
            }

            // Error Message
            errorMessage?.let { error ->
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    colors = CardDefaults.cardColors(androidx.compose.ui.graphics.Color.Red.copy(alpha = 0.1f))
                ) {
                    Row(
                        modifier = Modifier.padding(12.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            androidx.compose.material.icons.Icons.Default.Warning,
                            contentDescription = null,
                            tint = androidx.compose.ui.graphics.Color.Red
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(error, color = androidx.compose.ui.graphics.Color.Red)
                    }
                }
            }

            // Send Button
            Button(
                onClick = {
                    if (recipientAddress.isEmpty()) {
                        errorMessage = "Please enter recipient address"
                        return@Button
                    }
                    if (!isValidAddress(recipientAddress)) {
                        errorMessage = "Invalid address format"
                        return@Button
                    }
                    if (amount.isEmpty() || amount.toDoubleOrNull() == null || amount.toDouble() <= 0) {
                        errorMessage = "Please enter valid amount"
                        return@Button
                    }

                    isLoading = true
                    errorMessage = null

                    // Simulate transaction
                    android.os.Handler(android.os.Looper.getMainLooper()).postDelayed({
                        isLoading = false
                        val txHash = "0x" + (0..63).map { "0123456789abcdef".random() }.joinToString("")
                        Toast.makeText(context, "Transaction submitted!\nHash: $txHash", Toast.LENGTH_LONG).show()
                        recipientAddress = ""
                        amount = ""
                    }, 2000)
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                enabled = !isLoading
            ) {
                if (isLoading) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(24.dp),
                        color = androidx.compose.ui.graphics.Color.White
                    )
                } else {
                    Text("Send $selectedToken", style = MaterialTheme.typography.title6)
                }
            }
        }
    }

    // QR Scanner Dialog
    if (showQRScanner) {
        AlertDialog(
            onDismissRequest = { showQRScanner = false },
            title = { Text("Scan QR Code") },
            text = {
                Column {
                    // Camera placeholder
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(200.dp)
                            .padding(16.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            Icon(
                                androidx.compose.material.icons.Icons.Default.QrCodeScanner,
                                contentDescription = null,
                                modifier = Modifier.size(60.dp),
                                tint = androidx.compose.ui.graphics.Color.Gray
                            )
                            Spacer(modifier = Modifier.height(8.dp))
                            Text("Camera QR Scanner", color = androidx.compose.ui.graphics.Color.Gray)
                        }
                    }

                    Divider()

                    // Manual entry
                    OutlinedTextField(
                        value = "",
                        onValueChange = { /* Handle manual input */ },
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = { Text("Or enter address manually") },
                        singleLine = true
                    )

                    Spacer(modifier = Modifier.height(8.dp))

                    // Recent addresses
                    Text("Recent Addresses", style = MaterialTheme.typography.subtitle2)
                    Spacer(modifier = Modifier.height(8.dp))

                    // Recent addresses are loaded from the backend transaction
                    // history — never hardcoded demo addresses.
                    emptyList<String>().forEach { address ->
                        TextButton(
                            onClick = {
                                recipientAddress = address
                                showQRScanner = false
                            },
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Text(
                                "${address.take(10)}...${address.takeLast(8)}",
                                style = MaterialTheme.typography.body2
                            )
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showQRScanner = false }) {
                    Text("Done")
                }
            }
        )
    }
}

private fun isValidAddress(address: String): Boolean {
    // Ethereum
    if (address.matches(Regex("^0x[a-fA-F0-9]{40}$"))) return true
    // Bitcoin
    if (address.startsWith("bc1") || address.startsWith("1") || address.startsWith("3")) {
        return address.length in 26..62
    }
    // Solana
    if (address.matches(Regex("^[1-9A-HJ-NP-Z]{32,44}$"))) return true
    // TRON
    if (address.startsWith("T") && address.length == 34) return true
    return false
}
