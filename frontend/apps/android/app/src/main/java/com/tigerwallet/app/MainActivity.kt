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

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent { MainScreen() }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen() {
    Scaffold(
        topBar = { TopAppBar(title = { Row { Text("🐯", fontSize = 24.sp); Text(" TigerWallet", fontWeight = FontWeight.Bold) } }) }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
            Card(modifier = Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = Color(0xFFFFE5D9))) {
                Column(modifier = Modifier.padding(24.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Total Balance", color = Color.Gray)
                    Text("$12,450.00", fontSize = 40.sp, fontWeight = FontWeight.Bold)
                    Text("4.2 ETH", color = Color.Gray)
                }
            }
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                ActionButton("📤", "Send", Color(0xFFFF6B35), Modifier.weight(1f))
                ActionButton("📥", "Receive", Color(0xFF4CAF50), Modifier.weight(1f))
                ActionButton("🔄", "Swap", Color(0xFF2196F3), Modifier.weight(1f))
                ActionButton("📊", "Portfolio", Color(0xFF9C27B0), Modifier.weight(1f))
            }
            Text("Chains", fontWeight = FontWeight.Bold)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                ChainChip("⬡ ETH", true)
                ChainChip("🟡 BNB", false)
                ChainChip("☀️ SOL", false)
            }
            Spacer()
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceEvenly) {
                NavItem("💳", "Wallet", true)
                NavItem("📈", "Trade", false)
                NavItem("🌐", "DApps", false)
                NavItem("👤", "Profile", false)
            }
        }
    }
}

@Composable
fun ActionButton(icon: String, label: String, color: Color, modifier: Modifier = Modifier) {
    Card(modifier = modifier, colors = CardDefaults.cardColors(containerColor = Color(0xFFF5F5F5))) {
        Column(modifier = Modifier.padding(16.dp).fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
            Text(icon, fontSize = 24.sp)
            Text(label, fontSize = 12.sp)
        }
    }
}

@Composable
fun ChainChip(label: String, active: Boolean) {
    Surface(color = if (active) Color(0xFFFF6B35) else Color(0xFFF5F5F5), shape = androidx.compose.foundation.shape.RoundedCornerShape(20.dp)) {
        Text(label, modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp), color = if (active) Color.White else Color.Black)
    }
}

@Composable
fun NavItem(icon: String, label: String, active: Boolean) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(icon, fontSize = 20.sp)
        Text(label, fontSize = 10.sp, color = if (active) Color(0xFFFF6B35) else Color.Gray)
    }
}
