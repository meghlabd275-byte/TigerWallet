package com.tigerwallet.admin.ui.screens.tokens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TokensScreen(onNavigateBack: () -> Unit) {
    val tokens = listOf(Pair("Bitcoin", "BTC"), Pair("Ethereum", "ETH"), Pair("Tether", "USDT"), Pair("BNB", "BNB"), Pair("Solana", "SOL"))
    Scaffold(topBar = { TopAppBar(title = { Text("Tokens") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }, actions = { IconButton(onClick = {}) { Icon(Icons.Default.Add, contentDescription = "Add") } }) }) { padding ->
        LazyColumn(modifier = Modifier.fillMaxSize().padding(padding), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            items(tokens) { (name, symbol) ->
                Card(modifier = Modifier.fillMaxWidth()) {
                    ListItem(headlineContent = { Text(name) }, supportingContent = { Text(symbol) }, leadingContent = { Icon(Icons.Default.Token, contentDescription = null) }, trailingContent = { Text("$45,234") })
                }
            }
        }
    }
}
