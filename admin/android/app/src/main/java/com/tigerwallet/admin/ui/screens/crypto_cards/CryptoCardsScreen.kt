package com.tigerwallet.admin.ui.screens.crypto_cards

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.launch
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory

data class CryptoCard(
    val id: Long,
    val userName: String,
    val cardNumber: String,
    val currency: String,
    val balance: Double,
    val limit: Double,
    val status: String,
    val cardType: String
)

class CryptoCardsViewModel : ViewModel() {
    private val _cards = mutableStateOf<List<CryptoCard>>(emptyList())
    val cards: List<CryptoCard> = _cards.value

    private val _loading = mutableStateOf(false)
    val loading: Boolean = _loading.value

    private val _filter = mutableStateOf("all")
    val filter: String = _filter.value

    private val _isDark = mutableStateOf(false)
    val isDark: Boolean = _isDark.value

    init { loadCards() }

    fun loadCards() {
        viewModelScope.launch {
            _loading.value = true
            try {
                // In production, call actual API
                _cards.value = getMockCards()
            } catch (e: Exception) {
                _cards.value = getMockCards()
            } finally {
                _loading.value = false
            }
        }
    }

    fun setFilter(f: String) {
        _filter.value = f
        loadCards()
    }

    fun toggleTheme() {
        _isDark.value = !_isDark.value
    }

    fun blockCard(id: Long) {
        // API call to block card
        loadCards()
    }

    fun activateCard(id: Long) {
        // API call to activate card
        loadCards()
    }

    private fun getMockCards(): List<CryptoCard> = listOf(
        CryptoCard(1, "John Doe", "4532123456789012", "USDT", 5000.0, 10000.0, "active", "virtual"),
        CryptoCard(2, "Jane Smith", "4532987654321098", "USDT", 2500.0, 5000.0, "blocked", "physical")
    )
}

@Composable
fun CryptoCardsScreen(viewModel: CryptoCardsViewModel = androidx.lifecycle.viewModel.compose rememberViewModel { CryptoCardsViewModel() }) {
    val isDark = viewModel.isDark
    val cards = viewModel.cards
    val loading = viewModel.loading
    val filter = viewModel.filter

    val colors = if (isDark) darkColors() else lightColors()
    val backgroundColor = colors.background
    val surfaceColor = colors.surface

    MaterialTheme(colors = colors) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Crypto Cards") },
                    actions = {
                        IconButton(onClick = { viewModel.toggleTheme() }) {
                            Icon(if (isDark) "light_mode" else "dark_mode", contentDescription = "Theme")
                        }
                    },
                    backgroundColor = surfaceColor
                )
            },
            backgroundColor = backgroundColor
        ) { padding ->
            Column(modifier = Modifier.padding(padding).fillMaxSize()) {
                // Stats
                Row(modifier = Modifier.padding(16.dp).fillMaxWidth()) {
                    StatCard("Total", "${cards.size}", Modifier.weight(1f))
                    StatCard("Active", "${cards.count { it.status == "active" }}", Modifier.weight(1f))
                    StatCard("Balance", "$${cards.sumOf { it.balance }}", Modifier.weight(1f))
                }

                // Filters
                Row(modifier = Modifier.padding(horizontal = 16.dp)) {
                    FilterChip(selected = filter == "all", onClick = { viewModel.setFilter("all") }, label = { Text("All") })
                    Spacer(Modifier.width(8.dp))
                    FilterChip(selected = filter == "active", onClick = { viewModel.setFilter("active") }, label = { Text("Active") })
                    Spacer(Modifier.width(8.dp))
                    FilterChip(selected = filter == "blocked", onClick = { viewModel.setFilter("blocked") }, label = { Text("Blocked") })
                }

                if (loading) {
                    Box(Modifier.fillMaxSize(), contentAlignment = androidx.compose.ui.Alignment.Center) {
                        CircularProgressIndicator()
                    }
                } else {
                    LazyColumn(modifier = Modifier.padding(16.dp)) {
                        items(cards) { card ->
                            Card(
                                modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp),
                                elevation = 4.dp
                            ) {
                                Column(modifier = Modifier.padding(16.dp)) {
                                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                        Text("•••• ${card.cardNumber.takeLast(4)}", style = MaterialTheme.typography.titleMedium)
                                        Chip(label = card.status, color = if (card.status == "active") android.graphics.Color.GREEN else android.graphics.Color.RED)
                                    }
                                    Spacer(Modifier.height(8.dp))
                                    Text("${card.userName} - ${card.currency} ${card.balance}")
                                    Spacer(Modifier.height(8.dp))
                                    Row {
                                        if (card.status == "active") {
                                            Button(onClick = { viewModel.blockCard(card.id) }, colors = ButtonDefaults.buttonColors(backgroundColor = MaterialTheme.colors.error)) {
                                                Text("Block")
                                            }
                                        } else {
                                            Button(onClick = { viewModel.activateCard(card.id) }) {
                                                Text("Activate")
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
}

@Composable
private fun StatCard(label: String, value: String, modifier: Modifier = Modifier) {
    Card(modifier = modifier.padding(4.dp)) {
        Column(modifier = Modifier.padding(12.dp), horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally) {
            Text(value, style = MaterialTheme.typography.headlineSmall)
            Text(label, style = MaterialTheme.typography.caption)
        }
    }
}

@Composable
private fun Chip(label: String, color: android.graphics.Color) {
    Surface(color = color.copy(alpha = 0.2f), shape = MaterialTheme.shapes.small) {
        Text(label, modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp), color = color)
    }
}
