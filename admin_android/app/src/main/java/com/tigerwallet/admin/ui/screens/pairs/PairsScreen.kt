package com.tigerwallet.admin.ui.screens.pairs

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
fun PairsScreen(onNavigateBack: () -> Unit) {
    val pairs = listOf(Pair("BTC/USDT", "$45,234"), Pair("ETH/USDT", "$2,456"), Pair("BNB/USDT", "$312"), Pair("SOL/USDT", "$98"))
    Scaffold(topBar = { TopAppBar(title = { Text("Trading Pairs") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }, actions = { IconButton(onClick = {}) { Icon(Icons.Default.Add, contentDescription = "Add") } }) }) { padding ->
        LazyColumn(modifier = Modifier.fillMaxSize().padding(padding), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            items(pairs) { (pair, price) ->
                Card(modifier = Modifier.fillMaxWidth()) {
                    ListItem(headlineContent = { Text(pair) }, supportingContent = { Text(price) }, leadingContent = { Icon(Icons.Default.SwapHoriz, contentDescription = null) }, trailingContent = { AssistChip(onClick = {}, label = { Text("Active") }) })
                }
            }
        }
    }
}
