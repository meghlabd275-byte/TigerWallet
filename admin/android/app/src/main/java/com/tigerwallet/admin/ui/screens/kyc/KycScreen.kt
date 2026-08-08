/**
 * TigerWallet Admin - KYC Screen
 */

package com.tigerwallet.admin.ui.screens.kyc

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
fun KycScreen(onNavigateBack: () -> Unit) {
    var selectedTab by remember { mutableStateOf(0) }
    Scaffold(
        topBar = { TopAppBar(title = { Text("KYC Verification") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }) }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            TabRow(selectedTabIndex = selectedTab) {
                Tab(selected = selectedTab == 0, onClick = { selectedTab = 0 }, text = { Text("Pending") })
                Tab(selected = selectedTab == 1, onClick = { selectedTab = 1 }, text = { Text("Approved") })
                Tab(selected = selectedTab == 2, onClick = { selectedTab = 2 }, text = { Text("Rejected") })
            }
            LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                items(10) { index ->
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Icon(Icons.Default.Person, contentDescription = null)
                                Spacer(modifier = Modifier.width(12.dp))
                                Column(modifier = Modifier.weight(1f)) {
                                    Text("KYC Request ${index + 1}", style = MaterialTheme.typography.titleMedium)
                                    Text("Level ${(index % 3) + 1}", style = MaterialTheme.typography.bodySmall)
                                }
                                AssistChip(onClick = {}, label = { Text("Pending") })
                            }
                            Spacer(modifier = Modifier.height(12.dp))
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                Button(onClick = {}, colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)) { Text("Approve") }
                                OutlinedButton(onClick = {}) { Text("Reject") }
                            }
                        }
                    }
                }
            }
        }
    }
}
