import SwiftUI

// NFT Screen - NFT Gallery
struct NFTScreen: View {
    @State private var selectedTab: String = "Collectibles"
    @State private var searchText: String = ""
    
    let tabs = ["Collectibles", "Activity", "OpenSea"]
    
    let sampleNFTs = [
        NFTItem(name: "Bored Ape #1234", collection: "Bored Ape Yacht Club", image: "🦍", price: "45.5 ETH"),
        NFTItem(name: "CryptoPunk #5678", collection: "CryptoPunks", image: "👾", price: "32.0 ETH"),
        NFTItem(name: "Azuki #9012", collection: "Azuki", image: "🥷", price: "15.2 ETH"),
        NFTItem(name: "Doodle #3456", collection: "Doodles", image: "🎨", price: "3.5 ETH"),
        NFTItem(name: "Moonbird #7890", collection: "Moonbirds", image: "🐦", price: "8.1 ETH")
    ]
    
    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                // Search Bar
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Search NFTs", text: $searchText)
                }
                .padding()
                .background(Color.gray.opacity(0.1))
                .cornerRadius(10)
                .padding()
                
                // Tab Selector
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 16) {
                        ForEach(tabs, id: \.self) { tab in
                            Button(action: { selectedTab = tab }) {
                                VStack {
                                    Text(tab)
                                        .fontWeight(selectedTab == tab ? .bold : .regular)
                                    Rectangle()
                                        .fill(selectedTab == tab ? Color.orange : Color.clear)
                                        .frame(height: 2)
                                }
                            }
                            .foregroundColor(selectedTab == tab ? .primary : .secondary)
                        }
                    }
                    .padding(.horizontal)
                }
                
                if selectedTab == "Collectibles" {
                    // NFT Grid
                    ScrollView {
                        LazyVGrid(columns: [
                            GridItem(.flexible()),
                            GridItem(.flexible())
                        ], spacing: 16) {
                            ForEach(filteredNFTs) { nft in
                                nftCard(nft: nft)
                            }
                        }
                        .padding()
                    }
                } else if selectedTab == "Activity" {
                    activityView
                } else {
                    openSeaView
                }
            }
            .navigationTitle("NFTs")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: {}) {
                        Image(systemName: "square.and.arrow.down")
                    }
                }
            }
        }
    }
    
    var filteredNFTs: [NFTItem] {
        if searchText.isEmpty {
            return sampleNFTs
        }
        return sampleNFTs.filter { nft in
            nft.name.localizedCaseInsensitiveContains(searchText) ||
            nft.collection.localizedCaseInsensitiveContains(searchText)
        }
    }
    
    func nftCard(nft: NFTItem) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            // NFT Image placeholder
            ZStack {
                RoundedRectangle(cornerRadius: 12)
                    .fill(Color.gray.opacity(0.2))
                    .aspectRatio(1, contentMode: .fit)
                
                Text(nft.image)
                    .font(.system(size: 60))
            }
            
            VStack(alignment: .leading, spacing: 4) {
                Text(nft.name)
                    .font(.caption)
                    .fontWeight(.semibold)
                    .lineLimit(1)
                
                Text(nft.collection)
                    .font(.caption2)
                    .foregroundColor(.secondary)
                    .lineLimit(1)
                
                Text(nft.price)
                    .font(.caption)
                    .foregroundColor(.orange)
            }
        }
        .padding(8)
        .background(Color.gray.opacity(0.1))
        .cornerRadius(12)
    }
    
    var activityView: some View {
        ScrollView {
            VStack(spacing: 12) {
                ForEach(0..<5, id: \.self) { _ in
                    HStack {
                        Image(systemName: "arrow.up.right")
                            .foregroundColor(.red)
                        VStack(alignment: .leading) {
                            Text("Sent")
                                .font(.subheadline)
                            Text("Bored Ape #1234")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        Spacer()
                        Text("- 1 NFT")
                            .font(.caption)
                    }
                    .padding()
                    .background(Color.gray.opacity(0.1))
                    .cornerRadius(8)
                }
            }
            .padding()
        }
    }
    
    var openSeaView: some View {
        VStack(spacing: 20) {
            Image(systemName: "globe")
                .font(.system(size: 60))
                .foregroundColor(.secondary)
            
            Text("OpenSea Integration")
                .font(.headline)
            
            Text("Connect to OpenSea to view and trade NFTs")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
            
            Button(action: {}) {
                Text("Connect OpenSea")
                    .padding()
                    .background(Color.orange)
                    .foregroundColor(.white)
                    .cornerRadius(12)
            }
        }
        .padding()
    }
}

struct NFTItem: Identifiable {
    let id = UUID()
    let name: String
    let collection: String
    let image: String
    let price: String
}

#Preview {
    NFTScreen()
}
