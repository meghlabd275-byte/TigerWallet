package com.tigerwallet.app

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
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            val viewModel: WalletViewModel = androidx.lifecycle.viewmodel.compose.viewModel()
            val isDarkMode by viewModel.isDarkMode.collectAsState()
            
            TigerWalletTheme(darkTheme = isDarkMode) {
                MainScreen(viewModel = viewModel)
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen(viewModel: WalletViewModel) {
    var selectedTab by remember { mutableIntStateOf(0) }
    val isDarkMode by viewModel.isDarkMode.collectAsState()
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { 
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text("🐯", fontSize = 24.sp)
                        Text(" TigerWallet", fontWeight = FontWeight.Bold)
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
                    icon = { Text("💳") },
                    label = { Text("Wallet") },
                    selected = selectedTab == 0,
                    onClick = { selectedTab = 0 }
                )
                NavigationBarItem(
                    icon = { Text("📈") },
                    label = { Text("Trade") },
                    selected = selectedTab == 1,
                    onClick = { selectedTab = 1 }
                )
                NavigationBarItem(
                    icon = { Text("🌐") },
                    label = { Text("DApps") },
                    selected = selectedTab == 2,
                    onClick = { selectedTab = 2 }
                )
                NavigationBarItem(
                    icon = { Text("👤") },
                    label = { Text("Settings") },
                    selected = selectedTab == 3,
                    onClick = { selectedTab = 3 }
                )
            }
        }
    ) { padding ->
        when (selectedTab) {
            0 -> WalletScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
            1 -> TradeScreen(modifier = Modifier.padding(padding))
            2 -> DAppsScreen(modifier = Modifier.padding(padding))
            3 -> SettingsScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
        }
    }
}

@Composable
fun WalletScreen(viewModel: WalletViewModel, modifier: Modifier = Modifier) {
    val wallet by viewModel.wallet.collectAsState()
    val selectedChain by viewModel.selectedChain.collectAsState()
    val isLoading by viewModel.isLoading.collectAsState()
    
    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        // Balance Card
        Card(modifier = Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = Color(0xFFFFE5D9))) {
            Column(modifier = Modifier.padding(24.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Total Balance", color = Color.Gray)
                Text("$${wallet?.totalBalance ?: "0.00"}", fontSize = 40.sp, fontWeight = FontWeight.Bold)
                Text("${wallet?.nativeBalance ?: "0.0"} ${selectedChain.symbol}", color = Color.Gray)
            }
        }
        
        // Action Buttons
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            ActionButton("📤", "Send", Color(0xFFFF6B35), Modifier.weight(1f)) { }
            ActionButton("📥", "Receive", Color(0xFF4CAF50), Modifier.weight(1f)) { }
            ActionButton("🔄", "Swap", Color(0xFF2196F3), Modifier.weight(1f)) { }
            ActionButton("📊", "Portfolio", Color(0xFF9C27B0), Modifier.weight(1f)) { }
        }
        
        // Chain Selector
        Text("Chains", fontWeight = FontWeight.Bold)
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Chain.chains.forEach { chain ->
                ChainChip(
                    name = "${chain.icon} ${chain.name}",
                    active = selectedChain == chain,
                    onClick = { viewModel.selectChain(chain) }
                )
            }
        }
        
        // Token List
        if (isLoading) {
            Box(modifier = Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
        } else {
            wallet?.tokens?.forEach { token ->
                TokenRow(token = token)
            }
        }
    }
}

@Composable
fun TokenRow(token: Token) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.padding(16.dp).fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(token.symbol, fontWeight = FontWeight.Bold)
                Text(token.name, color = Color.Gray, fontSize = 12.sp)
            }
            Column(horizontalAlignment = Alignment.End) {
                Text("$${token.balanceUSD}", fontWeight = FontWeight.Bold)
                Text(token.balance, color = Color.Gray, fontSize = 12.sp)
            }
        }
    }
}

@Composable
fun TradeScreen(modifier: Modifier = Modifier) {
    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text("Trading Terminal", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("ETH/USDT", fontSize = 18.sp, fontWeight = FontWeight.Bold)
                Text("$3,500.00", fontSize = 32.sp, fontWeight = FontWeight.Bold, color = Color(0xFF4CAF50))
                Text("+2.5% (24h)", color = Color(0xFF4CAF50))
            }
        }
        
        Spacer()
    }
}

@Composable
fun DAppsScreen(modifier: Modifier = Modifier) {
    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text("DApps", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        
        val dapps = listOf(
            DApp("Uniswap", "🔄", "DeFi"),
            DApp("Aave", "👻", "Lending"),
            DApp("OpenSea", "🌊", "NFT")
        )
        
        dapps.forEach { dapp ->
            Card(modifier = Modifier.fillMaxWidth()) {
                Row(
                    modifier = Modifier.padding(16.dp).fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(dapp.icon, fontSize = 24.sp)
                    Spacer(modifier = Modifier.width(12.dp))
                    Column {
                        Text(dapp.name, fontWeight = FontWeight.Bold)
                        Text(dapp.category, color = Color.Gray, fontSize = 12.sp)
                    }
                }
            }
        }
    }
}

@Composable
fun SettingsScreen(viewModel: WalletViewModel, modifier: Modifier = Modifier) {
    val isDarkMode by viewModel.isDarkMode.collectAsState()
    
    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text("Settings", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
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
                Text("Biometric Auth")
                Text("Change PIN")
                Text("Recovery Phrase")
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

@Composable
fun ActionButton(icon: String, label: String, color: Color, modifier: Modifier = Modifier, onClick: () -> Unit) {
    Card(modifier = modifier, colors = CardDefaults.cardColors(containerColor = Color(0xFFF5F5F5)), onClick = onClick) {
        Column(modifier = Modifier.padding(16.dp).fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
            Text(icon, fontSize = 24.sp)
            Text(label, fontSize = 12.sp)
        }
    }
}

@Composable
fun ChainChip(name: String, active: Boolean, onClick: () -> Unit) {
    Surface(
        color = if (active) Color(0xFFFF6B35) else Color(0xFFF5F5F5),
        shape = androidx.compose.foundation.shape.RoundedCornerShape(20.dp),
        onClick = onClick
    ) {
        Text(
            name,
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            color = if (active) Color.White else Color.Black
        )
    }
}

// Data classes
data class Chain(val name: String, val symbol: String, val icon: String, val chainId: Int) {
    companion object {
        val chains = listOf(
            Chain("Ethereum", "ETH", "⬡", 1),
            Chain("BSC", "BNB", "🟡", 56),
            Chain("Polygon", "MATIC", "🟣", 137)
        )
    }
}

data class Token(val symbol: String, val name: String, val balance: String, val balanceUSD: Double, val logoUrl: String = "")
data class DApp(val name: String, val icon: String, val category: String)

// Theme
@Composable
fun TigerWalletTheme(darkTheme: Boolean, content: @Composable () -> Unit) {
    val colors = if (darkTheme) {
        darkColorScheme()
    } else {
        lightColorScheme()
    }
    
    MaterialTheme(colorScheme = colors, content = content)
}
