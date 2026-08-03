package com.tigeradmin

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.ViewModel

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            val viewModel: AdminViewModel = androidx.lifecycle.viewModel.compose.rememberViewModel()
            val isDarkMode by viewModel.isDarkMode.collectAsState()
            
            AdminTheme(darkTheme = isDarkMode) {
                AdminMainScreen(viewModel = viewModel)
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AdminMainScreen(viewModel: AdminViewModel) {
    var selectedTab by remember { mutableIntStateOf(0) }
    val isDarkMode by viewModel.isDarkMode.collectAsState()
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { 
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("🔧", fontSize = 24.sp)
                        Text(" Admin Panel", fontWeight = FontWeight.Bold)
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.toggleDarkMode() }) {
                        Text(if (isDarkMode) "☀️" else "🌙")
                    }
                }
            )
        },
        bottomBar = {
            NavigationBar {
                NavigationBarItem(
                    icon = { Text("📊") },
                    label = { Text("Dashboard") },
                    selected = selectedTab == 0,
                    onClick = { selectedTab = 0 }
                )
                NavigationBarItem(
                    icon = { Text("👥") },
                    label = { Text("Users") },
                    selected = selectedTab == 1,
                    onClick = { selectedTab = 1 }
                )
                NavigationBarItem(
                    icon = { Text("📜") },
                    label = { Text("Transactions") },
                    selected = selectedTab == 2,
                    onClick = { selectedTab = 2 }
                )
                NavigationBarItem(
                    icon = { Text("⚙️") },
                    label = { Text("System") },
                    selected = selectedTab == 3,
                    onClick = { selectedTab = 3 }
                )
            }
        }
    ) { padding ->
        when (selectedTab) {
            0 -> AdminDashboardScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
            1 -> UsersScreen(modifier = Modifier.padding(padding))
            2 -> TransactionsAdminScreen(modifier = Modifier.padding(padding))
            3 -> SystemScreen(modifier = Modifier.padding(padding))
        }
    }
}

@Composable
fun AdminDashboardScreen(viewModel: AdminViewModel, modifier: Modifier = Modifier) {
    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text("Dashboard", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            AdminStatCard(title = "Total Users", value = "12,450", icon = "👥", modifier = Modifier.weight(1f))
            AdminStatCard(title = "Total Volume", value = "$45.2M", icon = "💰", modifier = Modifier.weight(1f))
        }
        
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            AdminStatCard(title = "Pending KYC", value = "89", icon = "⏳", modifier = Modifier.weight(1f))
            AdminStatCard(title = "System Health", value = "99.9%", icon = "❤️", modifier = Modifier.weight(1f))
        }
        
        Text("Quick Actions", fontWeight = FontWeight.Bold)
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            AdminActionButton("👤", "Users", Modifier.weight(1f)) { }
            AdminActionButton("📜", "Transactions", Modifier.weight(1f)) { }
            AdminActionButton("⚙️", "Settings", Modifier.weight(1f)) { }
            AdminActionButton("📊", "Analytics", Modifier.weight(1f)) { }
        }
        
        Text("Recent Activity", fontWeight = FontWeight.Bold)
        for (i in 0..4) {
            AdminActivityRow()
        }
    }
}

@Composable
fun AdminStatCard(title: String, value: String, icon: String, modifier: Modifier = Modifier) {
    Card(modifier = modifier) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(icon, fontSize = 20.sp)
                Spacer(Modifier.weight(1f))
            }
            Spacer(modifier = Modifier.height(8.dp))
            Text(value, fontSize = 24.sp, fontWeight = FontWeight.Bold)
            Text(title, fontSize = 12.sp, color = Color.Gray)
        }
    }
}

@Composable
fun AdminActionButton(icon: String, label: String, modifier: Modifier = Modifier, onClick: () -> Unit) {
    Card(modifier = modifier, onClick = onClick) {
        Column(
            modifier = Modifier.padding(12.dp).fillMaxWidth(),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Text(icon, fontSize = 24.sp)
            Text(label, fontSize = 10.sp)
        }
    }
}

@Composable
fun AdminActivityRow() {
    Card(modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.padding(12.dp).fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("👤", fontSize = 20.sp)
                Spacer(modifier = Modifier.width(8.dp))
                Column {
                    Text("New user verified", fontSize = 14.sp)
                    Text("user@example.com", fontSize = 10.sp, color = Color.Gray)
                }
            }
            Text("2 min ago", fontSize = 10.sp, color = Color.Gray)
        }
    }
}

@Composable
fun UsersScreen(modifier: Modifier = Modifier) {
    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Text("Users", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        for (i in 0..10) {
            Card(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                Row(
                    modifier = Modifier.padding(12.dp).fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Column {
                        Text("user$i@example.com", fontWeight = FontWeight.Bold)
                        Text(if (i % 3 == 0) "Pending" else "Verified", fontSize = 10.sp, 
                            color = if (i % 3 == 0) Color(0xFFFF9800) else Color(0xFF4CAF50))
                    }
                    Text("→", fontSize = 20.sp)
                }
            }
        }
    }
}

@Composable
fun TransactionsAdminScreen(modifier: Modifier = Modifier) {
    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Text("Transactions", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        for (i in 0..15) {
            Card(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                Row(
                    modifier = Modifier.padding(12.dp).fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Column {
                        Text(if (i % 2 == 0) "Transfer" else "Swap", fontWeight = FontWeight.Bold)
                        Text("0x${i.toString().padStart(16, '0')}...", fontSize = 10.sp, color = Color.Gray)
                    }
                    Column(horizontalAlignment = Alignment.End) {
                        Text("$${(100..50000).random()}", fontWeight = FontWeight.Bold)
                        Text(if (i % 4 == 0) "Pending" else "Confirmed", fontSize = 10.sp,
                            color = if (i % 4 == 0) Color(0xFFFF9800) else Color(0xFF4CAF50))
                    }
                }
            }
        }
    }
}

@Composable
fun SystemScreen(modifier: Modifier = Modifier) {
    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text("System Status", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Services", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))
                SystemServiceRow("API Gateway", "Running", "99.99%")
                SystemServiceRow("Wallet Service", "Running", "99.95%")
                SystemServiceRow("Transaction Engine", "Running", "99.99%")
                SystemServiceRow("Price Feed", "Running", "99.90%")
            }
        }
        
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Database", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))
                SystemServiceRow("PostgreSQL", "Running", "99.99%")
                SystemServiceRow("Redis Cache", "Running", "99.95%")
            }
        }
        
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Network", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))
                SystemServiceRow("Ethereum RPC", "Running", "99.80%")
                SystemServiceRow("BSC RPC", "Running", "99.85%")
            }
        }
    }
}

@Composable
fun SystemServiceRow(name: String, status: String, uptime: String) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(if (status == "Running") "✅" else "❌")
            Spacer(modifier = Modifier.width(8.dp))
            Text(name)
        }
        Text(uptime, color = Color.Gray)
    }
}

// ViewModel
class AdminViewModel : ViewModel() {
    private val _isDarkMode = MutableStateFlow(false)
    val isDarkMode: StateFlow<Boolean> = _isDarkMode.asStateFlow()
    
    private val _users = MutableStateFlow<List<String>>(emptyList())
    val users: StateFlow<List<String>> = _users.asStateFlow()
    
    fun toggleDarkMode() {
        _isDarkMode.value = !_isDarkMode.value
    }
}

// Theme
@Composable
fun AdminTheme(darkTheme: Boolean, content: @Composable () -> Unit) {
    val colors = if (darkTheme) {
        darkColorScheme()
    } else {
        lightColorScheme()
    }
    
    MaterialTheme(colorScheme = colors, content = content)
}
