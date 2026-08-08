package com.tigerwallet.admin.ui.screens.whitelabels

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
fun WhiteLabelsScreen(onNavigateBack: () -> Unit) {
    val whiteLabels = listOf(Pair("Company A", "a.tigerwallet.com"), Pair("Company B", "b.tigerwallet.com"), Pair("Company C", "c.tigerwallet.com"))
    Scaffold(topBar = { TopAppBar(title = { Text("White Labels") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }, actions = { IconButton(onClick = {}) { Icon(Icons.Default.Add, contentDescription = "Add") } }) }) { padding ->
        LazyColumn(modifier = Modifier.fillMaxSize().padding(padding), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            items(whiteLabels) { (company, domain) ->
                Card(modifier = Modifier.fillMaxWidth()) {
                    ListItem(headlineContent = { Text(company) }, supportingContent = { Text(domain) }, leadingContent = { Icon(Icons.Default.Business, contentDescription = null) }, trailingContent = { AssistChip(onClick = {}, label = { Text("Active") }) })
                }
            }
        }
    }
}
