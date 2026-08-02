package com.tigerwallet.app.ui.screens

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp

// NFT Screen - NFT Gallery
class NFTScreen : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                NFTScreenContent()
            }
        }
    }
}

@OptIn(ExperimentalMaterialApi::class)
@Composable
fun NFTScreenContent() {
    var selectedTab by remember { mutableStateOf("Collectibles") }
    var searchText by remember { mutableStateOf("") }
    
    val tabs = listOf("Collectibles", "Activity", "OpenSea")
    val nfts = listOf(
        NFTData("Bored Ape #1234", "Bored Ape Yacht Club", "🦍", "45.5 ETH"),
        NFTData("CryptoPunk #5678", "CryptoPunks", "👾", "32.0 ETH"),
        NFTData("Azuki #9012", "Azuki", "🥷", "15.2 ETH"),
        NFTData("Doodle #3456", "Doodles", "🎨", "3.5 ETH"),
        NFTData("Moonbird #7890", "Moonbirds", "🐦", "8.1 ETH"),
        NFTData("Pudgy #2345", "Pudgy Penguins", "🐧", "2.8 ETH")
    )

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("NFTs") },
                actions = {
                    IconButton(onClick = { /* Download */ }) {
                        Icon(androidx.compose.material.icons.Icons.Default.Download, contentDescription = "Download")
                    }
                }
            )
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
        ) {
            // Search Bar
            OutlinedTextField(
                value = searchText,
                onValueChange = { searchText = it },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                placeholder = { Text("Search NFTs") },
                leadingIcon = { Icon(androidx.compose.material.icons.Icons.Default.Search, contentDescription = null) },
                singleLine = true,
                shape = RoundedCornerShape(12.dp)
            )

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
                "Collectibles" -> nftGrid(nfts, searchText)
                "Activity" -> activityView()
                "OpenSea" -> openSeaView()
            }
        }
    }
}

@Composable
fun nftGrid(nfts: List<NFTData>, search: String) {
    val filtered = if (search.isEmpty()) nfts else nfts.filter {
        it.name.contains(search, ignoreCase = true) || it.collection.contains(search, ignoreCase = true)
    }

    LazyVerticalGrid(
        columns = GridCells.Fixed(2),
        contentPadding = PaddingValues(16.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        items(filtered) { nft ->
            NFTCard(nft = nft)
        }
    }
}

@Composable
fun NFTCard(nft: NFTData) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp)
    ) {
        Column {
            // NFT Image placeholder
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .aspectRatio(1f)
                    .background(MaterialTheme.colors.surfaceVariant),
                contentAlignment = Alignment.Center
            ) {
                Text(nft.image, style = MaterialTheme.typography.h2)
            }
            
            Column(modifier = Modifier.padding(12.dp)) {
                Text(
                    nft.name,
                    style = MaterialTheme.typography.subtitle2,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Text(
                    nft.collection,
                    style = MaterialTheme.typography.caption,
                    color = MaterialTheme.colors.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    nft.price,
                    style = MaterialTheme.typography.caption,
                    color = MaterialTheme.colors.primary
                )
            }
        }
    }
}

@Composable
fun activityView() {
    val activities = listOf(
        "Sent Bored Ape #1234" to "- 1 NFT",
        "Received CryptoPunk #5678" to "+ 1 NFT",
        "Listed Azuki #9012" to "2.5 ETH",
        "Sold Doodle #3456" to "3.5 ETH"
    )

    Column(
        modifier = Modifier.fillMaxSize().padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        activities.forEach { (title, amount) ->
            Card(modifier = Modifier.fillMaxWidth()) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column {
                        Text(title, style = MaterialTheme.typography.subtitle2)
                        Text("Just now", style = MaterialTheme.typography.caption, color = MaterialTheme.colors.onSurfaceVariant)
                    }
                    Text(amount, style = MaterialTheme.typography.subtitle2, color = MaterialTheme.colors.primary)
                }
            }
        }
    }
}

@Composable
fun openSeaView() {
    Column(
        modifier = Modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            androidx.compose.material.icons.Icons.Default.Language,
            contentDescription = null,
            modifier = Modifier.size(80.dp),
            tint = MaterialTheme.colors.onSurfaceVariant
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text("OpenSea Integration", style = MaterialTheme.typography.h6)
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            "Connect to OpenSea to view and trade NFTs",
            style = MaterialTheme.typography.body2,
            color = MaterialTheme.colors.onSurfaceVariant
        )
        Spacer(modifier = Modifier.height(24.dp))
        Button(onClick = { /* Connect */ }) {
            Text("Connect OpenSea")
        }
    }
}

data class NFTData(
    val name: String,
    val collection: String,
    val image: String,
    val price: String
)
