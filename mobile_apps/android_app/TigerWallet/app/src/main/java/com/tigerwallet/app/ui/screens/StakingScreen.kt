package com.tigerwallet.app.ui.screens

import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp

// Staking Screen - Stake and earn
class StakingScreen : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                StakingScreenContent()
            }
        }
    }
}

@OptIn(ExperimentalMaterialApi::class)
@Composable
fun StakingScreenContent() {
    var selectedTab by remember { mutableStateOf("Stake") }
    var stakeAmount by remember { mutableStateOf("") }
    var selectedPool by remember { mutableStateOf("ETH 2.0") }
    var isStaking by remember { mutableStateOf(false) }
    
    val tabs = listOf("Stake", "Earn", "Pools")
    val pools = listOf(
        StakingPool("ETH 2.0", "4.2%", "1.5 ETH", "0.063 ETH"),
        StakingPool("BNB", "3.8%", "0 BNB", "0 BNB"),
        StakingPool("SOL", "6.5%", "0 SOL", "0 SOL"),
        StakingPool("MATIC", "5.2%", "0 MATIC", "0 MATIC")
    )
    
    val context = LocalContext.current

    Scaffold(
        topBar = {
            TopAppBar(title = { Text("Staking") })
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
        ) {
            // Tab Selector
            TabRow(selectedTabIndex = tabs.indexOf(selectedTab)) {
                tabs.forEachIndexed { index, tab ->
                    Tab(
                        selected = selectedTab == tab,
                        onClick = { selectedTab = tab },
                        text = { Text(tab) }
                    )
                }
            }

            when (selectedTab) {
                "Stake" -> stakeView(stakeAmount, { stakeAmount = it }, selectedPool, { selectedPool = it }, isStaking, { isStaking = it }, pools, context)
                "Earn" -> earnView(pools)
                "Pools" -> poolsView(pools)
            }
        }
    }
}

@Composable
fun stakeView(
    amount: String,
    onAmountChange: (String) -> Unit,
    pool: String,
    onPoolChange: (String) -> Unit,
    isLoading: Boolean,
    onLoadingChange: (Boolean) -> Unit,
    pools: List<StakingPool>,
    context: android.content.Context
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        // Pool Selector
        item {
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Select Pool", style = MaterialTheme.typography.subtitle2)
                    Spacer(modifier = Modifier.height(8.dp))
                    pools.forEach { p ->
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Text(p.name)
                            Text("APY: ${p.apy}", color = Color.Green)
                        }
                    }
                }
            }
        }

        // Amount
        item {
            OutlinedCard {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Text("Amount", style = MaterialTheme.typography.subtitle2)
                        TextButton(onClick = { onAmountChange("1.0") }) {
                            Text("MAX")
                        }
                    }
                    OutlinedTextField(
                        value = amount,
                        onValueChange = onAmountChange,
                        modifier = Modifier.fillMaxWidth(),
                        placeholder = { Text("0.0") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        singleLine = true
                    )
                }
            }
        }

        // Stake Button
        item {
            Button(
                onClick = {
                    if (amount.isEmpty()) {
                        Toast.makeText(context, "Enter amount", Toast.LENGTH_SHORT).show()
                        return@Button
                    }
                    onLoadingChange(true)
                    android.os.Handler(android.os.Looper.getMainLooper()).postDelayed({
                        onLoadingChange(false)
                        Toast.makeText(context, "Staked successfully!", Toast.LENGTH_SHORT).show()
                    }, 2000)
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                enabled = !isLoading
            ) {
                if (isLoading) {
                    CircularProgressIndicator(modifier = Modifier.size(24.dp), color = Color.White)
                } else {
                    Text("Stake", style = MaterialTheme.typography.title6)
                }
            }
        }
    }
}

@Composable
fun earnView(pools: List<StakingPool>) {
    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        items(pools) { pool ->
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween
                    ) {
                        Column {
                            Text(pool.name, style = MaterialTheme.typography.subtitle1)
                            Text("APY: ${pool.apy}", color = Color.Green)
                        }
                        Column(horizontalAlignment = Alignment.End) {
                            Text("Staked", style = MaterialTheme.typography.caption)
                            Text(pool.staked)
                        }
                    }
                    Spacer(modifier = Modifier.height(12.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column {
                            Text("Pending Reward", style = MaterialTheme.typography.caption)
                            Text(pool.reward, color = Color.Green)
                        }
                        Button(onClick = { }) {
                            Text("Claim")
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun poolsView(pools: List<StakingPool>) {
    LazyColumn(
        modifier = Modifier.fillMaxSize().padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        items(pools) { pool ->
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(pool.name, style = MaterialTheme.typography.h6)
                        Column(horizontalAlignment = Alignment.End) {
                            Text(pool.apy, style = MaterialTheme.typography.h5, color = Color.Green, fontWeight = androidx.compose.ui.text.font.FontWeight.Bold)
                            Text("APY", style = MaterialTheme.typography.caption)
                        }
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    Text("Total Staked: ${pool.staked}", style = MaterialTheme.typography.caption)
                    Spacer(modifier = Modifier.height(12.dp))
                    Button(onClick = { }, modifier = Modifier.fillMaxWidth()) {
                        Text("Stake")
                    }
                }
            }
        }
    }
}

data class StakingPool(
    val name: String,
    val apy: String,
    val staked: String,
    val reward: String
)
