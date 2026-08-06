package com.tigerwallet.admin.ui.screens.tickets

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
fun TicketsScreen(onNavigateBack: () -> Unit) {
    val tickets = listOf(Pair("Cannot withdraw funds", "Withdrawal"), Pair("KYC pending", "KYC"), Pair("Account locked", "Account"), Pair("Transaction issue", "Transaction"))
    Scaffold(topBar = { TopAppBar(title = { Text("Support Tickets") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }, actions = { IconButton(onClick = {}) { Icon(Icons.Default.Add, contentDescription = "Add") } }) }) { padding ->
        LazyColumn(modifier = Modifier.fillMaxSize().padding(padding), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            items(tickets) { (title, type) ->
                Card(modifier = Modifier.fillMaxWidth()) {
                    ListItem(headlineContent = { Text(title) }, supportingContent = { Text(type) }, leadingContent = { Icon(Icons.Default.Support, contentDescription = null) }, trailingContent = { AssistChip(onClick = {}, label = { Text("Open") }) })
                }
            }
        }
    }
}
