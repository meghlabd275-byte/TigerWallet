package com.tigerwallet.admin.ui.screens.transactions

import androidx.compose.foundation.clickable
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
fun TransactionsScreen(onNavigateBack: () -> Unit, onNavigateToDetail: (String) -> Unit) {
    Scaffold(topBar = { TopAppBar(title = { Text("Transactions") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }) }) { padding ->
        LazyColumn(modifier = Modifier.fillMaxSize().padding(padding), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            items(20) { index ->
                Card(modifier = Modifier.fillMaxWidth().clickable { onNavigateToDetail(index.toString()) }) {
                    Row(modifier = Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.SwapHoriz, contentDescription = null, modifier = Modifier.size(40.dp))
                        Spacer(modifier = Modifier.width(12.dp))
                        Column(modifier = Modifier.weight(1f)) {
                            Text("Transaction #${index + 1}", style = MaterialTheme.typography.titleMedium)
                            Text("0x${index.toString().padStart(10, '0')}", style = MaterialTheme.typography.bodySmall)
                        }
                        Column(horizontalAlignment = Alignment.End) {
                            Text("\$${(index + 1) * 100}", style = MaterialTheme.typography.titleMedium)
                            Text("Completed", style = MaterialTheme.typography.bodySmall)
                        }
                    }
                }
            }
        }
    }
}
