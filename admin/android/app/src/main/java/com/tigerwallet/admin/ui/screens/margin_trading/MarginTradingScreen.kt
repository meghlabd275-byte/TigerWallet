package com.tigerwallet.admin.ui.screens.margin_trading

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

data class MarginPosition(
    val id: Long,
    val userName: String,
    val pair: String,
    val side: String,
    val size: Double,
    val leverage: Int,
    val entryPrice: Double,
    val currentPrice: Double,
    val pnl: Double,
    val liquidationPrice: Double,
    val status: String
)

class MarginTradingViewModel : ViewModel() {
    private val _positions = mutableStateOf<List<MarginPosition>>(emptyList())
    val positions: List<MarginPosition> = _positions.value

    private val _stats = mutableStateOf<Map<String, Any>>(emptyMap())
    val stats: Map<String, Any> = _stats.value

    private val _loading = mutableStateOf(false)
    val loading: Boolean = _loading.value

    private val _filter = mutableStateOf("all")
    val filter: String = _filter.value

    private val _isDark = mutableStateOf(false)
    val isDark: Boolean = _isDark.value

    init { loadData() }

    fun loadData() {
        viewModelScope.launch {
            _loading.value = true
            try {
                // No admin margin endpoint is wired yet. Show honest empty state
                // (no fabricated positions/stats). Wire to a real admin/perpetual
                // backend endpoint before populating.
                _positions.value = emptyList()
                _stats.value = emptyMap()
            } catch (e: Exception) { } finally { _loading.value = false }
        }
    }

    fun setFilter(f: String) { _filter.value = f; loadData() }
    fun toggleTheme() { _isDark.value = !_isDark.value }
    fun liquidate(id: Long) { loadData() }

    // getMockPositions removed: do not fabricate margin positions.
}

@Composable
fun MarginTradingScreen(viewModel: MarginTradingViewModel = androidx.lifecycle.viewModel.compose.rememberViewModel { MarginTradingViewModel() }) {
    val isDark = viewModel.isDark
    val positions = viewModel.positions
    val stats = viewModel.stats
    val loading = viewModel.loading
    val filter = viewModel.filter
    val colors = if (isDark) darkColors() else lightColors()

    MaterialTheme(colors = colors) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Margin Trading") },
                    actions = { IconButton(onClick = { viewModel.toggleTheme() }) { Icon(if (isDark) "light_mode" else "dark_mode", "Theme") } },
                    backgroundColor = colors.surface
                )
            },
            backgroundColor = colors.background
        ) { padding ->
            Column(modifier = Modifier.padding(padding)) {
                Row(modifier = Modifier.padding(16.dp).fillMaxWidth()) {
                    StatCard("Positions", "${stats["totalPositions"]}", Modifier.weight(1f))
                    StatCard("Volume", "$${(stats["totalVolume"] as Double? ?: 0.0) / 1000000}M", Modifier.weight(1f))
                    StatCard("Liq.", "${stats["liquidationsToday"]}", Modifier.weight(1f))
                }

                Row(modifier = Modifier.padding(horizontal = 16.dp)) {
                    FilterChip(selected = filter == "all", onClick = { viewModel.setFilter("all") }, label = { Text("All") })
                    Spacer(Modifier.width(8.dp))
                    FilterChip(selected = filter == "open", onClick = { viewModel.setFilter("open") }, label = { Text("Open") })
                    Spacer(Modifier.width(8.dp))
                    FilterChip(selected = filter == "liquidated", onClick = { viewModel.setFilter("liquidated") }, label = { Text("Liquidated") })
                }

                if (loading) Box(Modifier.fillMaxSize(), Alignment.Center) { CircularProgressIndicator() }
                else {
                    LazyColumn(modifier = Modifier.padding(16.dp)) {
                        items(positions) { pos ->
                            Card(modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp)) {
                                Column(modifier = Modifier.padding(16.dp)) {
                                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                        Text("${pos.pair} (${pos.leverage}x)", style = MaterialTheme.typography.titleMedium)
                                        Surface(color = if (pos.side == "long") android.graphics.Color.GREEN else android.graphics.Color.RED, shape = MaterialTheme.shapes.small) {
                                            Text(pos.side.uppercase(), modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp), color = android.graphics.Color.WHITE)
                                        }
                                    }
                                    Spacer(Modifier.height(8.dp))
                                    Text("Trader: ${pos.userName}")
                                    Row(modifier = Modifier.fillMaxWidth()) {
                                        Col("Size", "${pos.size}", Modifier.weight(1f))
                                        Col("Entry", "$${pos.entryPrice}", Modifier.weight(1f))
                                        Col("Current", "$${pos.currentPrice}", Modifier.weight(1f))
                                        Col("PnL", "$${pos.pnl}", Modifier.weight(1f), if (pos.pnl >= 0) android.graphics.Color.GREEN else android.graphics.Color.RED)
                                    }
                                    if (pos.status == "open") {
                                        Spacer(Modifier.height(8.dp))
                                        Button(onClick = { viewModel.liquidate(pos.id) }, colors = ButtonDefaults.buttonColors(backgroundColor = MaterialTheme.colors.error)) {
                                            Text("Liquidate")
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable private fun StatCard(label: String, value: String, m: Modifier) = Card(m = m.padding(4.dp)) { Column(Modifier.padding(12.dp), Alignment.CenterHorizontally) { Text(value, style = MaterialTheme.typography.headlineSmall); Text(label, style = MaterialTheme.typography.caption) } }
@Composable private fun Col(label: String, value: String, m: Modifier, c: android.graphics.Color = android.graphics.Color.BLACK) = Column(m, Alignment.CenterHorizontally) { Text(label, style = MaterialTheme.typography.caption); Text(value, color = c) }
