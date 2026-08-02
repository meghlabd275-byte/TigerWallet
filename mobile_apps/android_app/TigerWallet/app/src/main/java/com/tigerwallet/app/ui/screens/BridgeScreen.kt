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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp

// Bridge Screen - Cross-chain bridging
class BridgeScreen : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                BridgeScreenContent()
            }
        }
    }
}

@OptIn(ExperimentalMaterialApi::class)
@Composable
fun BridgeScreenContent() {
    var fromChain by remember { mutableStateOf("Ethereum") }
    var toChain by remember { mutableStateOf("Polygon") }
    var token by remember { mutableStateOf("ETH") }
    var amount by remember { mutableStateOf("") }
    var isBridging by remember { mutableStateOf(false) }
    
    val chains = listOf("Ethereum", "BNB Chain", "Polygon", "Arbitrum", "Optimism", "Avalanche", "Solana", "Base", "Linea", "ZKSync")
    val tokens = listOf("ETH", "USDT", "USDC", "BNB", "MATIC", "AVAX", "SOL")
    val context = LocalContext.current

    Scaffold(
        topBar = {
            TopAppBar(title = { Text("Bridge") })
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
            // From Chain
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("From", style = MaterialTheme.typography.subtitle2)
                    Spacer(modifier = Modifier.height(8.dp))
                    
                    // Chain Selector
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        chains.take(4).forEach { chain ->
                            FilterChip(
                                selected = fromChain == chain,
                                onClick = { fromChain = chain },
                                label = { Text(chain.take(8)) }
                            )
                        }
                    }
                    
                    Spacer(modifier = Modifier.height(12.dp))
                    
                    // Token Selector
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        tokens.forEach { t ->
                            FilterChip(
                                selected = token == t,
                                onClick = { token = t },
                                label = { Text(t) }
                            )
                        }
                    }
                }
            }

            // Swap Button
            IconButton(
                onClick = {
                    val temp = fromChain
                    fromChain = toChain
                    toChain = temp
                },
                modifier = Modifier.align(Alignment.CenterHorizontally)
            ) {
                Icon(
                    androidx.compose.material.icons.Icons.Default.SwapVert,
                    contentDescription = "Swap",
                    tint = MaterialTheme.colors.primary,
                    modifier = Modifier.size(32.dp)
                )
            }

            // To Chain
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("To", style = MaterialTheme.typography.subtitle2)
                    Spacer(modifier = Modifier.height(8.dp))
                    
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        chains.take(4).forEach { chain ->
                            FilterChip(
                                selected = toChain == chain,
                                onClick = { toChain = chain },
                                label = { Text(chain.take(8)) }
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
                    OutlinedTextField(
                        value = amount,
                        onValueChange = { amount = it },
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = { Text("0.0") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        singleLine = true,
                        trailingIcon = { Text(token) }
                    )
                }
            }

            // Bridge Button
            Button(
                onClick = {
                    if (amount.isEmpty()) {
                        Toast.makeText(context, "Enter amount", Toast.LENGTH_SHORT).show()
                        return@Button
                    }
                    isBridging = true
                    android.os.Handler(android.os.Looper.getMainLooper()).postDelayed({
                        isBridging = false
                        Toast.makeText(context, "Bridge initiated!", Toast.LENGTH_SHORT).show()
                    }, 2000)
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                enabled = !isBridging
            ) {
                if (isBridging) {
                    CircularProgressIndicator(modifier = Modifier.size(24.dp), color = Color.White)
                } else {
                    Icon(androidx.compose.material.icons.Icons.Default.Bridge, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Bridge", style = MaterialTheme.typography.title6)
                }
            }

            // Info
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Estimated Time")
                        Text("~10-30 minutes", color = Color.Gray)
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Fee")
                        Text("~0.1%", color = Color.Gray)
                    }
                }
            }
        }
    }
}
