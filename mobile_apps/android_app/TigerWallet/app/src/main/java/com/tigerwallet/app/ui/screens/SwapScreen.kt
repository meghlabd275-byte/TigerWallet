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

// Swap Screen - DEX swapping functionality
class SwapScreen : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                SwapScreenContent()
            }
        }
    }
}

@OptIn(ExperimentalMaterialApi::class)
@Composable
fun SwapScreenContent() {
    var fromToken by remember { mutableStateOf("ETH") }
    var toToken by remember { mutableStateOf("USDT") }
    var fromAmount by remember { mutableStateOf("") }
    var toAmount by remember { mutableStateOf("") }
    var slippage by remember { mutableStateOf(0.5f) }
    var isSwapping by remember { mutableStateOf(false) }
    
    val tokens = listOf("ETH", "USDT", "USDC", "BNB", "MATIC", "SOL", "TRX", "BTC")
    val context = LocalContext.current

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Swap") },
                actions = {
                    IconButton(onClick = { /* Settings */ }) {
                        Icon(androidx.compose.material.icons.Icons.Default.Settings, contentDescription = "Settings")
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
            // From Token
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("You Pay", style = MaterialTheme.typography.subtitle2)
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        // Token Picker
                        var expanded by remember { mutableStateOf(false) }
                        ExposedDropdownMenuBox(
                            expanded = expanded,
                            onExpandedChange = { expanded = it }
                        ) {
                            OutlinedTextField(
                                value = fromToken,
                                onValueChange = {},
                                modifier = Modifier
                                    .menuAnchor()
                                    .width(120.dp),
                                readOnly = true,
                                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) }
                            )
                            ExposedDropdownMenu(
                                expanded = expanded,
                                onDismissRequest = { expanded = false }
                            ) {
                                tokens.forEach { token ->
                                    DropdownMenuItem(
                                        text = { Text(token) },
                                        onClick = {
                                            fromToken = token
                                            expanded = false
                                        }
                                    )
                                }
                            }
                        }
                        
                        // Amount
                        OutlinedTextField(
                            value = fromAmount,
                            onValueChange = { fromAmount = it },
                            modifier = Modifier.weight(1f).padding(start = 8.dp),
                            placeholder = { Text("0.0") },
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                            singleLine = true,
                            textStyle = MaterialTheme.typography.h6
                        )
                    }
                    
                    // Balance & Max
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Balance: 0.0", style = MaterialTheme.typography.caption, color = MaterialTheme.colors.onSurfaceVariant)
                        TextButton(onClick = { fromAmount = "1.0" }) {
                            Text("MAX")
                        }
                    }
                }
            }

            // Swap Button
            IconButton(
                onClick = {
                    val temp = fromToken
                    fromToken = toToken
                    toToken = temp
                }
            ) {
                Icon(
                    androidx.compose.material.icons.Icons.Default.SwapVert,
                    contentDescription = "Swap",
                    tint = MaterialTheme.colors.primary,
                    modifier = Modifier.size(32.dp)
                )
            }

            // To Token
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("You Receive", style = MaterialTheme.typography.subtitle2)
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        // Token Picker
                        var expanded by remember { mutableStateOf(false) }
                        ExposedDropdownMenuBox(
                            expanded = expanded,
                            onExpandedChange = { expanded = it }
                        ) {
                            OutlinedTextField(
                                value = toToken,
                                onValueChange = {},
                                modifier = Modifier
                                    .menuAnchor()
                                    .width(120.dp),
                                readOnly = true,
                                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) }
                            )
                            ExposedDropdownMenu(
                                expanded = expanded,
                                onDismissRequest = { expanded = false }
                            ) {
                                tokens.forEach { token ->
                                    DropdownMenuItem(
                                        text = { Text(token) },
                                        onClick = {
                                            toToken = token
                                            expanded = false
                                        }
                                    )
                                }
                            }
                        }
                        
                        // Amount
                        OutlinedTextField(
                            value = toAmount,
                            onValueChange = { toAmount = it },
                            modifier = Modifier.weight(1f).padding(start = 8.dp),
                            placeholder = { Text("0.0") },
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                            singleLine = true,
                            textStyle = MaterialTheme.typography.h6
                        )
                    }
                }
            }

            // Exchange Rate
            if (fromAmount.isNotEmpty() && toAmount.isNotEmpty()) {
                val rate = (toAmount.toDoubleOrNull() ?: 0.0) / (fromAmount.toDoubleOrNull() ?: 1.0)
                Text(
                    "1 $fromToken = ${String.format("%.6f", rate)} $toToken",
                    style = MaterialTheme.typography.caption,
                    color = MaterialTheme.colors.onSurfaceVariant
                )
            }

            // Slippage
            OutlinedCard {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("Slippage Tolerance")
                    TextButton(onClick = { /* Show slippage settings */ }) {
                        Text("${slippage}%")
                    }
                }
            }

            // Swap Button
            Button(
                onClick = {
                    if (fromAmount.isEmpty() || toAmount.isEmpty()) {
                        Toast.makeText(context, "Please enter amounts", Toast.LENGTH_SHORT).show()
                        return@Button
                    }
                    
                    isSwapping = true
                    android.os.Handler(android.os.Looper.getMainLooper()).postDelayed({
                        isSwapping = false
                        Toast.makeText(context, "Swap completed!", Toast.LENGTH_SHORT).show()
                        fromAmount = ""
                        toAmount = ""
                    }, 2000)
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                enabled = !isSwapping
            ) {
                if (isSwapping) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(24.dp),
                        color = androidx.compose.ui.graphics.Color.White
                    )
                } else {
                    Text("Swap", style = MaterialTheme.typography.title6)
                }
            }
        }
    }
}
