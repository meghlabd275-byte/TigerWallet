package com.tigermaster

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

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            val viewModel: MasterWalletViewModel = androidx.lifecycle.viewModel.compose.rememberViewModel()
            val isDarkMode by viewModel.isDarkMode.collectAsState()
            
            MasterWalletTheme(darkTheme = isDarkMode) {
                MasterMainScreen(viewModel = viewModel)
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MasterMainScreen(viewModel: MasterWalletViewModel) {
    var selectedTab by remember { mutableIntStateOf(0) }
    val isDarkMode by viewModel.isDarkMode.collectAsState()
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { 
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("🏦", fontSize = 24.sp)
                        Text(" MasterWallet", fontWeight = FontWeight.Bold)
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
                    icon = { Text("💼") },
                    label = { Text("Wallets") },
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
                    label = { Text("Settings") },
                    selected = selectedTab == 3,
                    onClick = { selectedTab = 3 }
                )
            }
        }
    ) { padding ->
        when (selectedTab) {
            0 -> DashboardScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
            1 -> WalletsScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
            2 -> TransactionsScreen(modifier = Modifier.padding(padding))
            3 -> SettingsScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
        }
    }
}

@Composable
fun DashboardScreen(viewModel: MasterWalletViewModel, modifier: Modifier = Modifier) {
    val wallet by viewModel.masterWallet.collectAsState()
    
    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text("Dashboard", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            MasterStatCard(
                title = "Wallets",
                value = "${wallet?.subWalletCount ?: 0}",
                icon = "💼",
                modifier = Modifier.weight(1f)
            )
            MasterStatCard(
                title = "Volume",
                value = "$${wallet?.totalVolume ?: "0"}",
                icon = "💰",
                modifier = Modifier.weight(1f)
            )
        }
        
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            MasterStatCard(
                title = "Users",
                value = "${wallet?.userCount ?: 0}",
                icon = "👥",
                modifier = Modifier.weight(1f)
            )
            MasterStatCard(
                title = "Pending",
                value = "${wallet?.pendingTx ?: 0}",
                icon = "⏳",
                modifier = Modifier.weight(1f)
            )
        }
        
        Text("Quick Actions", fontWeight = FontWeight.Bold)
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            MasterActionButton("➕", "Create Wallet", Modifier.weight(1f)) { }
            MasterActionButton("👤", "Add User", Modifier.weight(1f)) { }
            MasterActionButton("🔑", "Auto Sign", Modifier.weight(1f)) { }
            MasterActionButton("📊", "Analytics", Modifier.weight(1f)) { }
        }
        
        Text("Recent Activity", fontWeight = FontWeight.Bold)
        for (i in 0..3) {
            MasterActivityRow()
        }
    }
}

@Composable
fun MasterStatCard(title: String, value: String, icon: String, modifier: Modifier = Modifier) {
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
fun MasterActionButton(icon: String, label: String, modifier: Modifier = Modifier, onClick: () -> Unit) {
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
fun MasterActivityRow() {
    Card(modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.padding(12.dp).fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("📤", fontSize = 20.sp)
                Spacer(modifier = Modifier.width(8.dp))
                Column {
                    Text("Transaction Sent", fontSize = 14.sp)
                    Text("0x742d...12eB3", fontSize = 10.sp, color = Color.Gray)
                }
            }
            Text("+$5,000", color = Color(0xFF4CAF50), fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
fun WalletsScreen(viewModel: MasterWalletViewModel, modifier: Modifier = Modifier) {
    val wallets by viewModel.subWallets.collectAsState()
    
    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text("Sub-Wallets", fontSize = 24.sp, fontWeight = FontWeight.Bold)
            Button(onClick = { }) {
                Text("➕ Add")
            }
        }
        
        wallets.forEach { wallet ->
            Card(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                Row(
                    modifier = Modifier.padding(16.dp).fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Column {
                        Text(wallet.name, fontWeight = FontWeight.Bold)
                        Text(wallet.address, fontSize = 10.sp, color = Color.Gray)
                    }
                    Column(horizontalAlignment = Alignment.End) {
                        Text("$${wallet.balance}", fontWeight = FontWeight.Bold)
                        Text(wallet.status, fontSize = 10.sp, color = if (wallet.status == "Active") Color(0xFF4CAF50) else Color.Gray)
                    }
                }
            }
        }
    }
}

@Composable
fun TransactionsScreen(modifier: Modifier = Modifier) {
    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Text("Transactions", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        for (i in 0..5) {
            Card(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                Row(
                    modifier = Modifier.padding(12.dp).fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Column {
                        Text("ETH Transfer", fontWeight = FontWeight.Bold)
                        Text("To: 0x742d...12eB3", fontSize = 10.sp, color = Color.Gray)
                    }
                    Column(horizontalAlignment = Alignment.End) {
                        Text("-2.5 ETH", color = Color.Red)
                        Text("Confirmed", fontSize = 10.sp, color = Color(0xFF4CAF50))
                    }
                }
            }
        }
    }
}

@Composable
fun SettingsScreen(viewModel: MasterWalletViewModel, modifier: Modifier = Modifier) {
    val isDarkMode by viewModel.isDarkMode.collectAsState()
    
    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text("Settings", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Appearance", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text("Dark Mode")
                    Switch(checked = isDarkMode, onCheckedChange = { viewModel.toggleDarkMode() })
                }
            }
        }
        
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Security", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))
                Text("Auto-Sign Rules")
                Text("User Permissions")
                Text("API Keys")
            }
        }
        
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Network", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))
                Text("Default Chain: Ethereum")
            }
        }
        
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("About", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))
                Text("Version: 1.0.0")
            }
        }
    }
}

// Data classes
data class SubWalletData(
    val name: String,
    val address: String,
    val balance: String,
    val status: String
)

// Theme
@Composable
fun MasterWalletTheme(darkTheme: Boolean, content: @Composable () -> Unit) {
    val colors = if (darkTheme) {
        darkColorScheme()
    } else {
        lightColorScheme()
    }
    
    MaterialTheme(colorScheme = colors, content = content)
}
