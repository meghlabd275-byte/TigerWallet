package com.tigerwallet.app.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController

sealed class Screen(val route: String, val title: String, val icon: ImageVector) {
    object Wallet : Screen("wallet", "Wallet", Icons.Default.AccountBalanceWallet)
    object Swap : Screen("swap", "Swap", Icons.Default.SwapHoriz)
    object Portfolio : Screen("portfolio", "Portfolio", Icons.Default.PieChart)
    object Browser : Screen("browser", "Browser", Icons.Default.Language)
    object Settings : Screen("settings", "Settings", Icons.Default.Settings)
}

val bottomNavItems = listOf(
    Screen.Wallet,
    Screen.Swap,
    Screen.Portfolio,
    Screen.Browser,
    Screen.Settings
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen() {
    val navController = rememberNavController()
    var isDarkMode by remember { mutableStateOf(false) }
    
    Scaffold(
        bottomBar = {
            NavigationBar {
                val navBackStackEntry by navController.currentBackStackEntryAsState()
                val currentDestination = navBackStackEntry?.destination
                
                bottomNavItems.forEach { screen ->
                    NavigationBarItem(
                        icon = { Icon(screen.icon, contentDescription = screen.title) },
                        label = { Text(screen.title) },
                        selected = currentDestination?.hierarchy?.any { it.route == screen.route } == true,
                        onClick = {
                            navController.navigate(screen.route) {
                                popUpTo(navController.graph.findStartDestination().id) {
                                    saveState = true
                                }
                                launchSingleTop = true
                                restoreState = true
                            }
                        }
                    )
                }
            }
        }
    ) { innerPadding ->
        NavHost(
            navController = navController,
            startDestination = Screen.Wallet.route,
            modifier = Modifier.padding(innerPadding)
        ) {
            composable(Screen.Wallet.route) { WalletScreen(isDarkMode) }
            composable(Screen.Swap.route) { SwapScreen(isDarkMode) }
            composable(Screen.Portfolio.route) { PortfolioScreen(isDarkMode) }
            composable(Screen.Browser.route) { BrowserScreen(isDarkMode) }
            composable(Screen.Settings.route) { SettingsScreen(isDarkMode, onThemeChange = { isDarkMode = it }) }
        }
    }
}

@Composable
fun WalletScreen(isDarkMode: Boolean) {
    var selectedNetwork by remember { mutableStateOf("Ethereum") }
    val networks = listOf("Ethereum", "BNB Chain", "Polygon", "Arbitrum", "Optimism", "Avalanche")
    
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .padding(16.dp)
    ) {
        // Network Selector
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text("Wallet", style = MaterialTheme.typography.headlineMedium)
            IconButton(onClick = { /* Theme toggle */ }) {
                Icon(
                    if (isDarkMode) Icons.Default.LightMode else Icons.Default.DarkMode,
                    contentDescription = "Theme"
                )
            }
        }
        
        Spacer(modifier = Modifier.height(16.dp))
        
        // Network Dropdown
        var expanded by remember { mutableStateOf(false) }
        ExposedDropdownMenuBox(
            expanded = expanded,
            onExpandedChange = { expanded = !expanded }
        ) {
            OutlinedTextField(
                value = selectedNetwork,
                onValueChange = {},
                readOnly = true,
                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
                modifier = Modifier.menuAnchor()
            )
            ExposedDropdownMenu(
                expanded = expanded,
                onDismissRequest = { expanded = false }
            ) {
                networks.forEach { network ->
                    DropdownMenuItem(
                        text = { Text(network) },
                        onClick = {
                            selectedNetwork = network
                            expanded = false
                        }
                    )
                }
            }
        }
        
        Spacer(modifier = Modifier.height(24.dp))
        
        // Balance Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.primaryContainer
            )
        ) {
            Column(
                modifier = Modifier.padding(24.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Text(
                    "Total Balance",
                    style = MaterialTheme.typography.bodyMedium
                )
                Text(
                    "$12,450.00",
                    style = MaterialTheme.typography.headlineLarge
                )
                Text(
                    "+$123.45 (2.3%)",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.primary
                )
            }
        }
        
        Spacer(modifier = Modifier.height(24.dp))
        
        // Quick Actions
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceEvenly
        ) {
            ActionButton("Send", Icons.Default.ArrowUpward) { }
            ActionButton("Receive", Icons.Default.ArrowDownward) { }
            ActionButton("Buy", Icons.Default.CreditCard) { }
            ActionButton("Swap", Icons.Default.SwapHoriz) { }
        }
        
        Spacer(modifier = Modifier.height(24.dp))
        
        // Assets
        Text(
            "Assets",
            style = MaterialTheme.typography.titleMedium
        )
        
        Spacer(modifier = Modifier.height(8.dp))
        
        // Token list placeholder
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                TokenItem("Ethereum", "ETH", "1.5", "$5,250.00")
                HorizontalDivider()
                TokenItem("USDT", "USDT", "5,000", "$5,000.00")
                HorizontalDivider()
                TokenItem("USDC", "USDC", "2,200", "$2,200.00")
            }
        }
    }
}

@Composable
fun ActionButton(label: String, icon: androidx.compose.ui.graphics.vector.ImageVector, onClick: () -> Unit) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        FilledTonalIconButton(onClick = onClick) {
            Icon(icon, contentDescription = label)
        }
        Text(label, style = MaterialTheme.typography.labelSmall)
    }
}

@Composable
fun TokenItem(name: String, symbol: String, balance: String, value: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 8.dp),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Column {
            Text(name, style = MaterialTheme.typography.bodyLarge)
            Text(symbol, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        Column(horizontalAlignment = Alignment.End) {
            Text(balance, style = MaterialTheme.typography.bodyLarge)
            Text(value, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SwapScreen(isDarkMode: Boolean) {
    var fromToken by remember { mutableStateOf("ETH") }
    var toToken by remember { mutableStateOf("USDT") }
    var fromAmount by remember { mutableStateOf("") }
    var toAmount by remember { mutableStateOf("") }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Swap") },
                actions = {
                    IconButton(onClick = { /* Theme toggle */ }) {
                        Icon(
                            if (isDarkMode) Icons.Default.LightMode else Icons.Default.DarkMode,
                            contentDescription = "Theme"
                        )
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp)
        ) {
            // From Token
            OutlinedTextField(
                value = fromAmount,
                onValueChange = { fromAmount = it },
                label = { Text("From") },
                modifier = Modifier.fillMaxWidth()
            )
            
            // Swap Button
            IconButton(
                onClick = {
                    val temp = fromToken
                    fromToken = toToken
                    toToken = temp
                }
            ) {
                Icon(Icons.Default.SwapVert, contentDescription = "Swap")
            }
            
            // To Token
            OutlinedTextField(
                value = toAmount,
                onValueChange = { toAmount = it },
                label = { Text("To") },
                modifier = Modifier.fillMaxWidth()
            )
            
            Spacer(modifier = Modifier.height(16.dp))
            
            // Swap Button
            Button(
                onClick = { /* Execute swap */ },
                modifier = Modifier.fillMaxWidth(),
                enabled = fromAmount.isNotEmpty()
            ) {
                Text("Swap")
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PortfolioScreen(isDarkMode: Boolean) {
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Portfolio") },
                actions = {
                    IconButton(onClick = { /* Theme toggle */ }) {
                        Icon(
                            if (isDarkMode) Icons.Default.LightMode else Icons.Default.DarkMode,
                            contentDescription = "Theme"
                        )
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp)
        ) {
            Text(
                "Total Balance",
                style = MaterialTheme.typography.bodyMedium
            )
            Text(
                "$12,450.00",
                style = MaterialTheme.typography.displaySmall
            )
            
            Spacer(modifier = Modifier.height(24.dp))
            
            // Chart placeholder
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(200.dp)
            ) {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    Text("Chart Placeholder")
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BrowserScreen(isDarkMode: Boolean) {
    var url by remember { mutableStateOf("") }
    
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("DApp Browser") },
                actions = {
                    IconButton(onClick = { /* Theme toggle */ }) {
                        Icon(
                            if (isDarkMode) Icons.Default.LightMode else Icons.Default.DarkMode,
                            contentDescription = "Theme"
                        )
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            // URL Bar
            OutlinedTextField(
                value = url,
                onValueChange = { url = it },
                placeholder = { Text("Enter URL") },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                singleLine = true
            )
            
            // WebView placeholder
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(16.dp),
                contentAlignment = Alignment.Center
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Icon(
                        Icons.Default.Language,
                        contentDescription = null,
                        modifier = Modifier.size(64.dp),
                        tint = MaterialTheme.colorScheme.primary
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                    Text("DApp Browser")
                    Text(
                        "Connect to Web3 dApps",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(isDarkMode: Boolean, onThemeChange: (Boolean) -> Unit) {
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Settings") }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            // Dark Mode Toggle
            ListItem(
                headlineContent = { Text("Dark Mode") },
                leadingContent = {
                    Icon(
                        if (isDarkMode) Icons.Default.DarkMode else Icons.Default.LightMode,
                        contentDescription = null
                    )
                },
                trailingContent = {
                    Switch(
                        checked = isDarkMode,
                        onCheckedChange = onThemeChange
                    )
                }
            )
            
            HorizontalDivider()
            
            // Networks
            ListItem(
                headlineContent = { Text("Networks") },
                leadingContent = {
                    Icon(Icons.Default.NetworkCheck, contentDescription = null)
                },
                trailingContent = {
                    Icon(Icons.Default.ChevronRight, contentDescription = null)
                }
            )
            
            HorizontalDivider()
            
            // Security
            ListItem(
                headlineContent = { Text("Security") },
                leadingContent = {
                    Icon(Icons.Default.Security, contentDescription = null)
                },
                trailingContent = {
                    Icon(Icons.Default.ChevronRight, contentDescription = null)
                }
            )
            
            HorizontalDivider()
            
            // About
            ListItem(
                headlineContent = { Text("About") },
                leadingContent = {
                    Icon(Icons.Default.Info, contentDescription = null)
                },
                trailingContent = {
                    Icon(Icons.Default.ChevronRight, contentDescription = null)
                }
            )
        }
    }
}
