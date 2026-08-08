package com.tigerwallet.admin.ui.screens.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(onNavigateBack: () -> Unit, onNavigateToProfile: () -> Unit) {
    Scaffold(topBar = { TopAppBar(title = { Text("Settings") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }) }) { padding ->
        LazyColumn(modifier = Modifier.fillMaxSize().padding(padding), contentPadding = PaddingValues(16.dp)) {
            item { Text("Account", style = MaterialTheme.typography.titleMedium) }
            item { Card(modifier = Modifier.fillMaxWidth()) { Column { SettingsItem(Icons.Default.Person, "Profile", "Manage your profile", onNavigateToProfile); Divider(); SettingsItem(Icons.Default.Lock, "Security", "Password and 2FA", {}); Divider(); SettingsItem(Icons.Default.Notifications, "Notifications", "Push and email", {}) } } }
            item { Spacer(modifier = Modifier.height(16.dp)) }
            item { Text("Appearance", style = MaterialTheme.typography.titleMedium) }
            item { Card(modifier = Modifier.fillMaxWidth()) { Column { SettingsItem(Icons.Default.DarkMode, "Theme", "Dark/Light mode", {}); Divider(); SettingsItem(Icons.Default.Language, "Language", "English", {}) } } }
            item { Spacer(modifier = Modifier.height(16.dp)) }
            item { Text("System", style = MaterialTheme.typography.titleMedium) }
            item { Card(modifier = Modifier.fillMaxWidth()) { Column { SettingsItem(Icons.Default.Storage, "Backup", "Database backups", {}); Divider(); SettingsItem(Icons.Default.Webhook, "Webhooks", "Manage webhooks", {}); Divider(); SettingsItem(Icons.Default.Key, "API Keys", "Manage API keys", {}) } } }
            item { Spacer(modifier = Modifier.height(24.dp)) }
            item { Button(onClick = {}, modifier = Modifier.fillMaxWidth(), colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error)) { Text("Sign Out") } }
        }
    }
}

@Composable
fun SettingsItem(icon: ImageVector, title: String, subtitle: String, onClick: () -> Unit) {
    ListItem(headlineContent = { Text(title) }, supportingContent = { Text(subtitle) }, leadingContent = { Icon(icon, contentDescription = null) }, modifier = Modifier.clickable(onClick = onClick), trailingContent = { Icon(Icons.Default.ChevronRight, contentDescription = null) })
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileScreen(onNavigateBack: () -> Unit) {
    Scaffold(topBar = { TopAppBar(title = { Text("Profile") }, navigationIcon = { IconButton(onClick = onNavigateBack) { Icon(Icons.Default.ArrowBack, contentDescription = "Back") } }) }) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            Icon(Icons.Default.Person, contentDescription = null, modifier = Modifier.size(80.dp))
            Spacer(modifier = Modifier.height(16.dp))
            OutlinedTextField(value = "Admin User", onValueChange = {}, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
            Spacer(modifier = Modifier.height(12.dp))
            OutlinedTextField(value = "admin@tigerwallet.com", onValueChange = {}, label = { Text("Email") }, modifier = Modifier.fillMaxWidth())
            Spacer(modifier = Modifier.height(24.dp))
            Button(onClick = {}, modifier = Modifier.fillMaxWidth()) { Text("Save Changes") }
        }
    }
}
