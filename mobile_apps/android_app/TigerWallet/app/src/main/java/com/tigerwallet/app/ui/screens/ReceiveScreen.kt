package com.tigerwallet.app.ui.screens

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp

// Receive Screen - Displays QR Code for receiving crypto
class ReceiveScreen : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                ReceiveScreenContent()
            }
        }
    }
}

@OptIn(ExperimentalMaterialApi::class)
@Composable
fun ReceiveScreenContent() {
    var selectedChain by remember { mutableStateOf("Ethereum") }
    var showCopied by remember { mutableStateOf(false) }
    
    var walletAddress by remember { mutableStateOf("") }
    val chains = listOf("Ethereum", "BNB Chain", "Polygon", "Arbitrum", "Optimism", "Avalanche", "Solana", "TRON")
    val context = LocalContext.current

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Receive") }
            )
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(24.dp)
        ) {
            // Chain Selector
            OutlinedCard(
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Select Network", style = MaterialTheme.typography.subtitle2)
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

            // QR Code Display
            Card(
                modifier = Modifier.fillMaxWidth(),
                elevation = CardDefaults.elevation(8.dp)
            ) {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(24.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    // QR Code placeholder
                    Box(
                        modifier = Modifier
                            .size(200.dp)
                            .padding(16.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        // QR Code icon placeholder
                        Icon(
                            androidx.compose.material.icons.Icons.Default.QrCode,
                            contentDescription = null,
                            modifier = Modifier.size(150.dp),
                            tint = MaterialTheme.colors.primary
                        )
                    }

                    Text(
                        selectedChain,
                        style = MaterialTheme.typography.h6
                    )

                    Text(
                        "Scan to send ${getTokenSymbol(selectedChain)} to this address",
                        style = MaterialTheme.typography.caption,
                        color = MaterialTheme.colors.onSurfaceVariant,
                        textAlign = TextAlign.Center
                    )
                }
            }

            // Address Section
            OutlinedCard(
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(
                        "Your Address",
                        style = MaterialTheme.typography.subtitle2,
                        color = MaterialTheme.colors.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            walletAddress,
                            style = MaterialTheme.typography.body2,
                            modifier = Modifier.weight(1f)
                        )
                        IconButton(
                            onClick = {
                                val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                                val clip = ClipData.newPlainText("Wallet Address", walletAddress)
                                clipboard.setPrimaryClip(clip)
                                showCopied = true
                                android.os.Handler(android.os.Looper.getMainLooper()).postDelayed({
                                    showCopied = false
                                }, 2000)
                            }
                        ) {
                            Icon(
                                if (showCopied) androidx.compose.material.icons.Icons.Default.Check
                                else androidx.compose.material.icons.Icons.Default.ContentCopy,
                                contentDescription = "Copy",
                                tint = if (showCopied) androidx.compose.ui.graphics.Color.Green 
                                       else MaterialTheme.colors.primary
                            )
                        }
                    }
                }
            }

            // Share Button
            Button(
                onClick = {
                    val shareIntent = android.content.Intent().apply {
                        action = android.content.Intent.ACTION_SEND
                        putExtra(android.content.Intent.EXTRA_TEXT, walletAddress)
                        type = "text/plain"
                    }
                    context.startActivity(android.content.Intent.createChooser(shareIntent, "Share Address"))
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp)
            ) {
                Icon(androidx.compose.material.icons.Icons.Default.Share, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("Share Address")
            }
        }
    }
}

private fun getTokenSymbol(chain: String): String {
    return when (chain) {
        "Ethereum", "Arbitrum", "Optimism" -> "ETH"
        "BNB Chain" -> "BNB"
        "Polygon" -> "MATIC"
        "Avalanche" -> "AVAX"
        "Solana" -> "SOL"
        "TRON" -> "TRX"
        else -> "ETH"
    }
}
