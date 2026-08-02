package com.tigerwallet.app.ui.screens

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

// Home Screen - Main Dashboard
class HomeScreen : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                HomeScreenContent()
            }
        }
    }
}

@OptIn(ExperimentalMaterialApi::class)
@Composable
fun HomeScreenContent() {
    var selectedTab by remember { mutableStateOf("Wallet") }
    val totalBalance = 12450.00

    val tokens = listOf(
        TokenData("ETH", "Ethereum", 4.2, 8400.00, "◈"),
        TokenData("USDT", "Tether", 2500.0, 2500.00, "₮"),
        TokenData("BNB", "BNB", 1.5, 450.00, "🟡"),
        TokenData("SOL", "Solana", 22.0, 1100.00, "☀️")
    )

    Scaffold(
        bottomBar = {
            BottomNavigation(
                selectedTab = selectedTab,
                onTabSelected = { selectedTab = it }
            )
        }
    ) { paddingValues ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(20.dp)
        ) {
            // Balance Card
            item {
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    elevation = CardDefaults.elevation(8.dp)
                ) {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .background(
                                Brush.linearGradient(
                                    colors = listOf(
                                        Color(0xFFFF6B35),
                                        Color(0xFFFF8C5A)
                                    )
                                )
                            )
                            .padding(24.dp)
                    ) {
                        Column(
                            horizontalAlignment = Alignment.CenterHorizontally,
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Text(
                                "Total Balance",
                                style = MaterialTheme.typography.subtitle1,
                                color = Color.White.copy(alpha = 0.8f)
                            )
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                "$${String.format("%.2f", totalBalance)}",
                                style = MaterialTheme.typography.h3,
                                fontWeight = FontWeight.Bold,
                                color = Color.White
                            )
                            Spacer(modifier = Modifier.height(20.dp))
                            Row(
                                horizontalArrangement = Arrangement.spacedBy(20.dp)
                            ) {
                                Button(
                                    onClick = { /* Navigate to Send */ },
                                    colors = ButtonDefaults.buttonColors(
                                        backgroundColor = Color.White,
                                        contentColor = Color(0xFFFF6B35)
                                    ),
                                    shape = RoundedCornerShape(20.dp)
                                ) {
                                    Icon(androidx.compose.material.icons.Icons.Default.ArrowUpward, contentDescription = null)
                                    Spacer(modifier = Modifier.width(4.dp))
                                    Text("Send")
                                }
                                OutlinedButton(
                                    onClick = { /* Navigate to Receive */ },
                                    colors = ButtonDefaults.outlinedButtonColors(
                                        contentColor = Color.White
                                    ),
                                    shape = RoundedCornerShape(20.dp)
                                ) {
                                    Icon(androidx.compose.material.icons.Icons.Default.ArrowDownward, contentDescription = null)
                                    Spacer(modifier = Modifier.width(4.dp))
                                    Text("Receive")
                                }
                            }
                        }
                    }
                }
            }

            // Quick Actions
            item {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceEvenly
                ) {
                    QuickActionButton(icon = "🔄", label = "Swap", color = Color.Blue)
                    QuickActionButton(icon = "🌉", label = "Bridge", color = Color.Magenta)
                    QuickActionButton(icon = "📈", label = "Stake", color = Color.Green)
                    QuickActionButton(icon = "🖼️", label = "NFTs", color = Color.Red)
                }
            }

            // Assets Header
            item {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("Assets", style = MaterialTheme.typography.h6)
                    TextButton(onClick = { /* Add token */ }) {
                        Text("Add Token", color = Color(0xFFFF6B35))
                    }
                }
            }

            // Token List
            items(tokens) { token ->
                TokenRow(token = token)
            }
        }
    }
}

@Composable
fun QuickActionButton(icon: String, label: String, color: Color) {
    Column(
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Box(
            modifier = Modifier
                .size(50.dp)
                .clip(CircleShape)
                .background(color.copy(alpha = 0.2f)),
            contentAlignment = Alignment.Center
        ) {
            Text(icon, style = MaterialTheme.typography.h5)
        }
        Spacer(modifier = Modifier.height(4.dp))
        Text(label, style = MaterialTheme.typography.caption)
    }
}

@Composable
fun TokenRow(token: TokenData) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            // Icon
            Box(
                modifier = Modifier
                    .size(44.dp)
                    .clip(CircleShape)
                    .background(Color.Gray.copy(alpha = 0.2f)),
                contentAlignment = Alignment.Center
            ) {
                Text(token.icon, style = MaterialTheme.typography.title4)
            }
            
            Spacer(modifier = Modifier.width(12.dp))
            
            // Name & Symbol
            Column(modifier = Modifier.weight(1f)) {
                Text(token.name, style = MaterialTheme.typography.subtitle1, fontWeight = FontWeight.Medium)
                Text(token.symbol, style = MaterialTheme.typography.caption, color = MaterialTheme.colors.onSurfaceVariant)
            }
            
            // Balance & Value
            Column(horizontalAlignment = Alignment.End) {
                Text("${String.format("%.4f", token.balance)}", style = MaterialTheme.typography.subtitle1, fontWeight = FontWeight.Medium)
                Text("$${String.format("%.2f", token.value)}", style = MaterialTheme.typography.caption, color = MaterialTheme.colors.onSurfaceVariant)
            }
        }
    }
}

data class TokenData(
    val symbol: String,
    val name: String,
    val balance: Double,
    val value: Double,
    val icon: String
)

@Composable
fun BottomNavigation(
    selectedTab: String,
    onTabSelected: (String) -> Unit
) {
    val tabs = listOf("Wallet" to androidx.compose.material.icons.Icons.Default.Wallet, 
                      "DApps" to androidx.compose.material.icons.Icons.Default.Apps,
                      "Activity" to androidx.compose.material.icons.Icons.Default.History)

    BottomNavigation {
        tabs.forEach { (label, icon) ->
            BottomNavigationItem(
                selected = selectedTab == label,
                onClick = { onTabSelected(label) },
                icon = { Icon(icon, contentDescription = label) },
                label = { Text(label) }
            )
        }
    }
}
