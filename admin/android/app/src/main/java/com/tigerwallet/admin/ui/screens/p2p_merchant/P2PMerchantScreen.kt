package com.tigerwallet.admin.ui.screens.p2p_merchant

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

data class P2PMerchant(val id: Long, val businessName: String, val email: String, val country: String, val totalVolume: Double, val transactionCount: Int, val rating: Double, val status: String)

class P2PMerchantViewModel : ViewModel() {
    private val _merchants = mutableStateOf<List<P2PMerchant>>(emptyList())
    val merchants: List<P2PMerchant> = _merchants.value
    private val _loading = mutableStateOf(false)
    val loading: Boolean = _loading.value
    private val _filter = mutableStateOf("all")
    val filter: String = _filter.value
    private val _isDark = mutableStateOf(false)
    val isDark: Boolean = _isDark.value

    init { loadMerchants() }
    fun loadMerchants() { viewModelScope.launch { _loading.value = true; try { _merchants.value = emptyList() } catch (e: Exception) { } finally { _loading.value = false } }
    fun setFilter(f: String) { _filter.value = f; loadMerchants() }
    fun toggleTheme() { _isDark.value = !_isDark.value }
    fun approve(id: Long) { loadMerchants() }
    fun reject(id: Long) { loadMerchants() }

    // getMockMerchants removed: do not fabricate P2P merchants.
    )
}

@Composable
fun P2PMerchantScreen(viewModel: P2PMerchantViewModel = androidx.lifecycle.viewModel.compose.rememberViewModel { P2PMerchantViewModel() }) {
    val isDark = viewModel.isDark
    val merchants = viewModel.merchants
    val loading = viewModel.loading
    val filter = viewModel.filter
    val colors = if (isDark) darkColors() else lightColors()

    MaterialTheme(colors = colors) {
        Scaffold(topBar = { TopAppBar(title = { Text("P2P Merchants") }, actions = { IconButton(onClick = { viewModel.toggleTheme() }) { Icon(if (isDark) "light_mode" else "dark_mode", "Theme") } }, backgroundColor = colors.surface) }, backgroundColor = colors.background) { p ->
            Column(Modifier.padding(p)) {
                Row(Modifier.padding(16.dp).fillMaxWidth()) {
                    StatCard("Total", "${merchants.size}", Modifier.weight(1f))
                    StatCard("Approved", "${merchants.count { it.status == "approved" }}", Modifier.weight(1f))
                    StatCard("Pending", "${merchants.count { it.status == "pending" }}", Modifier.weight(1f))
                    StatCard("Volume", "$${(merchants.sumOf { it.totalVolume } / 1000).toInt()}K", Modifier.weight(1f))
                }
                Row(Modifier.padding(horizontal = 16.dp)) {
                    FilterChip(selected = filter == "all", onClick = { viewModel.setFilter("all") }, label = { Text("All") })
                    Spacer(Modifier.width(8.dp))
                    FilterChip(selected = filter == "pending", onClick = { viewModel.setFilter("pending") }, label = { Text("Pending") })
                    Spacer(Modifier.width(8.dp))
                    FilterChip(selected = filter == "approved", onClick = { viewModel.setFilter("approved") }, label = { Text("Approved") })
                }
                if (loading) Box(Modifier.fillMaxSize(), Alignment.Center) { CircularProgressIndicator() }
                else LazyColumn(Modifier.padding(16.dp)) {
                    items(merchants) { m ->
                        Card(Modifier.fillMaxWidth().padding(bottom = 12.dp)) {
                            Column(Modifier.padding(16.dp)) {
                                Row(Modifier.fillMaxWidth(), Arrangement.SpaceBetween) {
                                    Text(m.businessName, style = MaterialTheme.typography.titleMedium)
                                    Surface(color = when (m.status) { "approved" -> android.graphics.Color.GREEN; "pending" -> android.graphics.Color.parseColor("#FFA500"); else -> android.graphics.Color.GRAY }, shape = MaterialTheme.shapes.small) { Text(m.status.uppercase(), Modifier.padding(horizontal = 8.dp, vertical = 4.dp), color = android.graphics.Color.WHITE) }
                                }
                                Spacer(Modifier.height(8.dp))
                                Text("${m.email} • ${m.country}")
                                Row(Modifier.fillMaxWidth()) {
                                    Col("Volume", "$${(m.totalVolume / 1000).toInt()}K", Modifier.weight(1f))
                                    Col("Txns", "${m.transactionCount}", Modifier.weight(1f))
                                    Col("Rating", "${m.rating} ★", Modifier.weight(1f))
                                }
                                if (m.status == "pending") {
                                    Spacer(Modifier.height(8.dp))
                                    Row { Button(onClick = { viewModel.approve(m.id) }, colors = ButtonDefaults.buttonColors(backgroundColor = android.graphics.Color.GREEN)) { Text("Approve") }; Spacer(Modifier.width(8.dp)); Button(onClick = { viewModel.reject(m.id) }, colors = ButtonDefaults.buttonColors(backgroundColor = android.graphics.Color.RED)) { Text("Reject") } }
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
@Composable private fun Col(label: String, value: String, m: Modifier) = Column(m, Alignment.CenterHorizontally) { Text(label, style = MaterialTheme.typography.caption); Text(value) }
