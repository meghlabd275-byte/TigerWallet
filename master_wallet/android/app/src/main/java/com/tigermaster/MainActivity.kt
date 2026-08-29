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
import com.tigermaster.services.ThemeService

class MainActivity : ComponentActivity() {
    private lateinit var themeService: ThemeService

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        themeService = ThemeService(applicationContext)
        themeService.initialize()
        setContent {
            val viewModel: MasterWalletViewModel = androidx.lifecycle.viewmodel.compose.viewModel()
            val isDarkMode by viewModel.isDarkMode.collectAsState()

            // Keep Compose theme in sync with the AppCompatDelegate-driven ThemeService.
            LaunchedEffect(isDarkMode) {
                themeService.setTheme(if (isDarkMode) "dark" else "light")
            }

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
    val selectedFeature by viewModel.selectedFeature.collectAsState()
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
                    icon = { Text("🧩") },
                    label = { Text("More") },
                    selected = selectedTab == 4,
                    onClick = { selectedTab = 4; viewModel.closeFeature() }
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
        if (selectedFeature != null) {
            FeatureHostScreen(
                viewModel = viewModel,
                feature = selectedFeature!!,
                modifier = Modifier.padding(padding)
            )
        } else when (selectedTab) {
            0 -> DashboardScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
            1 -> WalletsScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
            2 -> TransactionsScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
            3 -> SettingsScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
            4 -> MoreScreen(viewModel = viewModel, modifier = Modifier.padding(padding))
        }
    }
}

@Composable
fun DashboardScreen(viewModel: MasterWalletViewModel, modifier: Modifier = Modifier) {
    val wallet by viewModel.masterWallet.collectAsState()
    val transactions by viewModel.transactions.collectAsState()
    val subWallets by viewModel.subWallets.collectAsState()
    val volume by viewModel.volumeStats.collectAsState()
    val liveEvent by viewModel.liveEvent.collectAsState()
    val killSwitch by viewModel.killSwitch.collectAsState()

    // Live backend /ws feed: starts once a wallet is selected, stops on dispose.
    LaunchedEffect(wallet?.id) {
        if (wallet != null) viewModel.startLiveFeed()
        viewModel.loadKillSwitchStatus()
    }
    DisposableEffect(Unit) {
        onDispose { viewModel.stopLiveFeed() }
    }

    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text("Dashboard", fontSize = 24.sp, fontWeight = FontWeight.Bold)

        if (killSwitch?.optBoolean("halted") == true) {
            Card(modifier = Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = Color(0x33B00020))) {
                Text(
                    "KILL SWITCH HALTED by SuperAdmin" +
                        (killSwitch?.optString("reason", "")?.takeIf { it.isNotBlank() }?.let { ": $it" } ?: "") +
                        " — all API operations are blocked.",
                    modifier = Modifier.padding(12.dp), fontSize = 12.sp, color = Color(0xFFB00020), fontWeight = FontWeight.Bold
                )
            }
        }

        liveEvent?.let {
            Card(modifier = Modifier.fillMaxWidth()) {
                Text("Live: $it", modifier = Modifier.padding(12.dp), fontSize = 12.sp)
            }
        }

        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Master Wallet", fontSize = 12.sp, color = Color.Gray)
                Text(wallet?.name?.ifBlank { "—" } ?: "—", fontWeight = FontWeight.Bold)
                Text(wallet?.address?.ifBlank { "—" } ?: "—", fontSize = 10.sp, color = Color.Gray)
            }
        }

        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            MasterStatCard(
                title = "Sub-Wallets",
                value = "${subWallets.size}",
                icon = "💼",
                modifier = Modifier.weight(1f)
            )
            MasterStatCard(
                title = "Tx Count",
                value = "${volume?.txCount ?: 0}",
                icon = "📊",
                modifier = Modifier.weight(1f)
            )
        }

        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            MasterStatCard(
                title = "Total Volume",
                value = volume?.totalVolume?.ifBlank { "0" } ?: "0",
                icon = "💰",
                modifier = Modifier.weight(1f)
            )
            MasterStatCard(
                title = "Pending Tx",
                value = "${transactions.count { it.status.equals("pending", ignoreCase = true) }}",
                icon = "⏳",
                modifier = Modifier.weight(1f)
            )
        }

        Text("Recent Activity", fontWeight = FontWeight.Bold)
        if (transactions.isEmpty()) {
            Text("No recent transactions", fontSize = 12.sp, color = Color.Gray)
        } else {
            transactions.take(5).forEach { tx ->
                MasterActivityRow(tx)
            }
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
fun MasterActivityRow(tx: TransactionData) {
    val isPending = tx.status.equals("pending", ignoreCase = true)
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
                    Text(tx.token.ifBlank { tx.chain }.ifBlank { "Transaction" }, fontSize = 14.sp)
                    Text(tx.hash.ifBlank { tx.id }.ifBlank { "—" }, fontSize = 10.sp, color = Color.Gray)
                }
            }
            Text(
                tx.amount.ifBlank { "0" },
                color = if (isPending) Color.Gray else Color(0xFF4CAF50),
                fontWeight = FontWeight.Bold
            )
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
fun TransactionsScreen(viewModel: MasterWalletViewModel, modifier: Modifier = Modifier) {
    val transactions by viewModel.transactions.collectAsState()

    Column(modifier = modifier.fillMaxSize().padding(16.dp)) {
        Text("Transactions", fontSize = 24.sp, fontWeight = FontWeight.Bold)
        if (transactions.isEmpty()) {
            Text("No transactions", fontSize = 12.sp, color = Color.Gray)
        } else {
            transactions.forEach { tx ->
                Card(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
                    Row(
                        modifier = Modifier.padding(12.dp).fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Column {
                            Text(tx.token.ifBlank { tx.chain }.ifBlank { "Transfer" }, fontWeight = FontWeight.Bold)
                            Text("To: ${tx.to.ifBlank { "—" }}", fontSize = 10.sp, color = Color.Gray)
                            Text(tx.hash.ifBlank { tx.id }, fontSize = 10.sp, color = Color.Gray)
                        }
                        Column(horizontalAlignment = Alignment.End) {
                            Text(tx.amount.ifBlank { "0" }, color = Color.Red)
                            Text(tx.status, fontSize = 10.sp, color = if (tx.status.equals("confirmed", ignoreCase = true)) Color(0xFF4CAF50) else Color.Gray)
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun SettingsScreen(viewModel: MasterWalletViewModel, modifier: Modifier = Modifier) {
    val isDarkMode by viewModel.isDarkMode.collectAsState()
    val networks by viewModel.networks.collectAsState()
    val autoSignRules by viewModel.autoSignRules.collectAsState()
    val users by viewModel.users.collectAsState()
    val killSwitch by viewModel.killSwitch.collectAsState()

    LaunchedEffect(Unit) { viewModel.loadKillSwitchStatus() }

    Column(modifier = modifier.fillMaxSize().padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text("Settings", fontSize = 24.sp, fontWeight = FontWeight.Bold)

        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text("Kill Switch (SuperAdmin)", fontWeight = FontWeight.Bold)
                    TextButton(onClick = { viewModel.loadKillSwitchStatus() }) { Text("Refresh") }
                }
                Spacer(modifier = Modifier.height(8.dp))
                val halted = killSwitch?.optBoolean("halted") == true
                if (killSwitch == null) {
                    Text("Status unavailable — sign in to load the kill-switch state.", fontSize = 12.sp, color = Color.Gray)
                } else {
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        Text("Global Halt State")
                        Text(
                            if (halted) "HALTED" else "Operational",
                            color = if (halted) Color.Red else Color(0xFF2E7D32),
                            fontWeight = FontWeight.Bold
                        )
                    }
                    val reason = killSwitch?.optString("reason", "") ?: ""
                    if (reason.isNotBlank()) Text("Reason: $reason", fontSize = 12.sp, color = Color.Gray)
                    val source = killSwitch?.optString("source", "") ?: ""
                    if (source.isNotBlank()) Text("Source: $source", fontSize = 12.sp, color = Color.Gray)
                    val note = killSwitch?.optString("note", "") ?: ""
                    if (note.isNotBlank()) Text(note, fontSize = 11.sp, color = Color.Gray)
                }
            }
        }

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
                Text("Auto-Sign Rules: ${autoSignRules.size}")
                Text("Users: ${users.size}")
            }
        }

        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Networks", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))
                if (networks.isEmpty()) {
                    Text("No networks configured", fontSize = 12.sp, color = Color.Gray)
                } else {
                    networks.forEach { network ->
                        Text("${network.name} (${network.id}) — ${network.symbol.ifBlank { "—" }}")
                    }
                }
            }
        }

        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("About", fontWeight = FontWeight.Bold)
                Spacer(modifier = Modifier.height(8.dp))
                Text("Backend: ${ApiConfig.BASE_URL}")
            }
        }
    }
}

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
