package com.tigerwallet.admin.ui.screens.liquidity

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.launch

data class LiquidityPool(val id: Long, val pair: String, val tokenA: String, val tokenB: String, val reserveA: Double, val reserveB: Double, val totalSupply: Double, val apr: Double, val volume24h: Double, val fees24h: Double, val status: String)

class LiquidityViewModel : ViewModel() {
    private val _pools = mutableStateOf<List<LiquidityPool>>(emptyList())
    val pools: List<LiquidityPool> = _pools.value
    private val _stats = mutableStateOf<Map<String, Any>>(emptyMap())
    val stats: Map<String, Any> = _stats.value
    private val _loading = mutableStateOf(false)
    val loading: Boolean = _loading.value
    private val _isDark = mutableStateOf(false)
    val isDark: Boolean = _isDark.value
    private val _showAddModal = mutableStateOf(false)
    val showAddModal: Boolean = _showAddModal.value

    init { loadData() }
    fun loadData() { viewModelScope.launch { _loading.value = true; try { _pools.value = getMockPools(); _stats.value = mapOf("totalPools" to 2, "totalValueLocked" to 15000000.0, "volume24h" to 7500000.0, "fees24h" to 22500.0) } catch (e: Exception) { } finally { _loading.value = false } } }
    fun toggleTheme() { _isDark.value = !_isDark.value }
    fun showAddModal() { _showAddModal.value = true }
    fun hideAddModal() { _showAddModal.value = false }

    private fun getMockPools(): List<LiquidityPool> = listOf(
        LiquidityPool(1, "USDT/ETH", "USDT", "ETH", 5000000.0, 2500.0, 100000.0, 15.5, 2500000.0, 7500.0, "active"),
        LiquidityPool(2, "USDT/BTC", "USDT", "BTC", 10000000.0, 200.0, 2500000.0, 12.3, 5000000.0, 15000.0, "active")
    )
}

@Composable
fun LiquidityScreen(viewModel: LiquidityViewModel = androidx.lifecycle.viewModel.compose.rememberViewModel { LiquidityViewModel() }) {
    val isDark = viewModel.isDark
    val pools = viewModel.pools
    val stats = viewModel.stats
    val loading = viewModel.loading
    val colors = if (isDark) darkColors() else lightColors()

    MaterialTheme(colors = colors) {
        Scaffold(topBar = { TopAppBar(title = { Text("Liquidity Pools") }, actions = { IconButton(onClick = { viewModel.toggleTheme() }) { Icon(if (isDark) "light_mode" else "dark_mode", "Theme") }; IconButton(onClick = { viewModel.showAddModal() }) { Icon("add", "Add") } }, backgroundColor = colors.surface) }, backgroundColor = colors.background) { p ->
            Column(Modifier.padding(p)) {
                Row(Modifier.padding(16.dp).fillMaxWidth()) {
                    StatCard("Pools", "${stats["totalPools"]}", Modifier.weight(1f))
                    StatCard("TVL", "$${((stats["totalValueLocked"] as Double? ?: 0.0) / 1000000).toInt()}M", Modifier.weight(1f))
                    StatCard("24h Vol", "$${((stats["volume24h"] as Double? ?: 0.0) / 1000000).toInt()}M", Modifier.weight(1f))
                    StatCard("24h Fees", "$${(stats["fees24h"] as Double? ?: 0.0).toInt()}", Modifier.weight(1f))
                }
                if (loading) Box(Modifier.fillMaxSize(), Alignment.Center) { CircularProgressIndicator() }
                else LazyColumn(Modifier.padding(16.dp)) {
                    items(pools) { pool ->
                        Card(Modifier.fillMaxWidth().padding(bottom = 12.dp)) {
                            Column(Modifier.padding(16.dp)) {
                                Row(Modifier.fillMaxWidth(), Arrangement.SpaceBetween) {
                                    Text(pool.pair, style = MaterialTheme.typography.titleLarge)
                                    Surface(color = if (pool.status == "active") android.graphics.Color.GREEN else android.graphics.Color.GRAY, shape = MaterialTheme.shapes.small) { Text(pool.status.uppercase(), Modifier.padding(horizontal = 8.dp, vertical = 4.dp), color = android.graphics.Color.WHITE) }
                                }
                                Spacer(Modifier.height(12.dp))
                                Row(Modifier.fillMaxWidth()) {
                                    Col("Reserve A", "${pool.reserveA.toInt()} ${pool.tokenA}", Modifier.weight(1f))
                                    Col("Reserve B", "${pool.reserveB} ${pool.tokenB}", Modifier.weight(1f))
                                    Col("APR", "${pool.apr}%", Modifier.weight(1f))
                                }
                                Spacer(Modifier.height(8.dp))
                                Row(Modifier.fillMaxWidth()) {
                                    Col("24h Vol", "$${(pool.volume24h / 1000).toInt()}K", Modifier.weight(1f))
                                    Col("24h Fees", "$${pool.fees24h.toInt()}", Modifier.weight(1f))
                                    Col("Supply", "${pool.totalSupply.toInt()}", Modifier.weight(1f))
                                }
                                Spacer(Modifier.height(12.dp))
                                Row { Button(onClick = { }) { Text("Add Liquidity") }; Spacer(Modifier.width(8.dp)); OutlinedButton(onClick = { }) { Text("Remove") } }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable private fun StatCard(label: String, value: String, m: Modifier) = Card(m = m.padding(4.dp)) { Column(Modifier.padding(12.dp), Alignment.CenterHorizontally) { Text(value, style = MaterialTheme.typography.headlineSmall); Text(label, style = MaterialTheme.typography.caption) } }
@Composable private fun Col(label: String, value: String, m: Modifier) = Column(m, Alignment.CenterHorizontally) { Text(label, style = MaterialTheme.typography.caption); Text(value, style = MaterialTheme.typography.bodyMedium) }
