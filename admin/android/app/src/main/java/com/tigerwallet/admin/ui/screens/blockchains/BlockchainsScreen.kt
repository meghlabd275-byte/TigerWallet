package com.tigerwallet.admin.ui.screens.blockchains

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BlockchainsScreen(onNavigateBack: () -> Unit) {
    val blockchains = listOf(Pair("Ethereum", "ETH"), Pair("Bitcoin", "BTC"), Pair("BNB Smart Chain", "BNB"), Pair("Solana", "SOL"), Pair("Polygon", "MATIC"))
    Scaffold(topBar = { TopAppBar(title = { Text("Blockchains") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }, actions = { IconButton(onClick = {}) { Icon(Icons.Default.Add, contentDescription = "Add") } }) }) { padding ->
        LazyColumn(modifier = Modifier.fillMaxSize().padding(padding), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            items(blockchains) { (name, symbol) ->
                Card(modifier = Modifier.fillMaxWidth()) {
                    ListItem(headlineContent = { Text(name) }, supportingContent = { Text(symbol) }, leadingContent = { Icon(Icons.Default.Language, contentDescription = null) }, trailingContent = { AssistChip(onClick = {}, label = { Text("Active") }) })
                }
            }
        }
    }
}
