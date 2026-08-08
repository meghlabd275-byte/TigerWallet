package com.tigerwallet.admin.ui.screens.analytics

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AnalyticsScreen(onNavigateBack: () -> Unit) {
    Scaffold(topBar = { TopAppBar(title = { Text("Analytics") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }) }) { padding ->
        LazyColumn(modifier = Modifier.fillMaxSize().padding(padding), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            item { Text("Overview", style = MaterialTheme.typography.titleLarge) }
            item { Card(modifier = Modifier.fillMaxWidth()) { Box(modifier = Modifier.height(200.dp)) { Text("Chart placeholder", modifier = Modifier.padding(16.dp)) } } }
            item { Text("User Analytics", style = MaterialTheme.typography.titleMedium) }
            item { Card(modifier = Modifier.fillMaxWidth()) { Column(modifier = Modifier.padding(16.dp)) { Text("Total Users: 12,543"); Text("Active Users: 8,921"); Text("New Users: 1,234") } } }
            item { Text("Transaction Analytics", style = MaterialTheme.typography.titleMedium) }
            item { Card(modifier = Modifier.fillMaxWidth()) { Column(modifier = Modifier.padding(16.dp)) { Text("Total: 456,789"); Text("Volume: \$12.5M"); Text("Fees: \$234K") } } }
        }
    }
}
