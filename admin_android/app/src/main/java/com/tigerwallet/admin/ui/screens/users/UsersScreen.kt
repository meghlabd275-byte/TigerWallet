/**
 * TigerWallet Admin - Users Screen
 */

package com.tigerwallet.admin.ui.screens.users

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
fun UsersScreen(
    onNavigateBack: () -> Unit,
    onNavigateToUserDetail: (String) -> Unit
) {
    val users = (1..20).map { "User $it" }
    var selectedTab by remember { mutableStateOf("All") }
    val tabs = listOf("All", "Active", "Suspended", "Banned")

    Scaffold(
        topBar = {
            TopAppBar(title = { Text("Users") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } })
        }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            // Search
            OutlinedTextField(
                value = "", onValueChange = {},
                modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
                placeholder = { Text("Search users...") },
                leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) }
            )

            // Tabs
            ScrollableTabRow(selectedTabIndex = tabs.indexOf(selectedTab)) {
                tabs.forEach { tab ->
                    Tab(selected = selectedTab == tab, onClick = { selectedTab = tab }, text = { Text(tab) })
                }
            }

            // Users List
            LazyColumn(modifier = Modifier.fillMaxSize(), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(users) { user ->
                    Card(modifier = Modifier.fillMaxWidth().clickable { onNavigateToUserDetail(user) }) {
                        Row(modifier = Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
                            Icon(Icons.Default.Person, contentDescription = null, modifier = Modifier.size(40.dp))
                            Spacer(modifier = Modifier.width(12.dp))
                            Column(modifier = Modifier.weight(1f)) {
                                Text(user, style = MaterialTheme.typography.titleMedium)
                                Text("user@example.com", style = MaterialTheme.typography.bodySmall)
                            }
                            AssistChip(onClick = {}, label = { Text("Active") })
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UserDetailScreen(userId: String, onNavigateBack: () -> Unit) {
    Scaffold(
        topBar = {
            TopAppBar(title = { Text("User: $userId") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } })
        }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp)) {
            Text("User Details", style = MaterialTheme.typography.headlineSmall)
            Spacer(modifier = Modifier.height(16.dp))
            Text("ID: $userId")
            Text("Email: user@example.com")
            Text("Status: Active")
            Spacer(modifier = Modifier.height(24.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedButton(onClick = {}) { Text("Ban") }
                OutlinedButton(onClick = {}) { Text("Suspend") }
                OutlinedButton(onClick = {}) { Text("Edit") }
            }
        }
    }
}
