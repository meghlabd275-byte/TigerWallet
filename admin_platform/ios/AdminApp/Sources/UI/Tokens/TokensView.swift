//
//  TokensView.swift
//  TigerWalletAdmin
//
//  Complete Token Management Screen
//

import SwiftUI

struct TokensView: View {
    @StateObject private var viewModel = TokensViewModel()
    @EnvironmentObject var themeManager: ThemeManager
    @State private var searchText = ""
    @State private var showAddToken = false
    
    var body: some View {
        VStack(spacing: 0) {
            // Search Bar
            HStack {
                SearchBar(text: $searchText, placeholder: "Search tokens...")
                
                Button(action: { showAddToken = true }) {
                    Image(systemName: "plus.circle.fill")
                        .font(.title2)
                        .foregroundColor(.blue)
                }
            }
            .padding()
            
            if viewModel.isLoading {
                Spacer()
                ProgressView()
                Spacer()
            } else if viewModel.tokens.isEmpty {
                Spacer()
                Text("No tokens found")
                    .foregroundColor(.secondary)
                Spacer()
            } else {
                List {
                    ForEach(viewModel.filteredTokens(searchText: searchText)) { token in
                        TokenRow(token: token) {
                            viewModel.toggleTokenStatus(token)
                        }
                    }
                }
                .listStyle(.plain)
            }
        }
        .background(themeManager.backgroundColor)
        .navigationTitle("Tokens")
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button(action: { viewModel.loadTokens() }) {
                    Image(systemName: "arrow.clockwise")
                }
            }
        }
        .sheet(isPresented: $showAddToken) {
            AddTokenView()
        }
        .onAppear {
            viewModel.loadTokens()
        }
    }
}

struct TokenRow: View {
    let token: Token
    let onToggleStatus: () -> Void
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        HStack(spacing: 12) {
            // Token Icon
            Circle()
                .fill(token.statusColor.opacity(0.2))
                .frame(width: 50, height: 50)
                .overlay(
                    Text(token.symbol.prefix(2)).uppercased()
                        .font(.headline)
                        .fontWeight(.bold)
                        .foregroundColor(token.statusColor)
                )
            
            // Token Info
            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(token.name)
                        .font(.headline)
                    
                    if token.isVerified {
                        Image(systemName: "checkmark.seal.fill")
                            .foregroundColor(.blue)
                            .font(.caption)
                    }
                }
                
                Text(token.symbol)
                    .font(.caption)
                    .foregroundColor(.secondary)
                
                HStack(spacing: 8) {
                    Text(token.chain)
                        .font(.caption2)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color.gray.opacity(0.2))
                        .cornerRadius(4)
                    
                    StatusBadge(status: token.status)
                }
            }
            
            Spacer()
            
            // Price Info
            VStack(alignment: .trailing, spacing: 4) {
                Text(token.formattedPrice)
                    .font(.headline)
                
                Text(token.formattedChange)
                    .font(.caption)
                    .foregroundColor(token.changeColor)
            }
            
            // Actions Menu
            Menu {
                Button("Edit Token") { }
                Button(token.status == "active" ? "Pause Token" : "Activate Token") {
                    onToggleStatus()
                }
                Button("View Holders") { }
                Button("View Transactions") { }
                Divider()
                Button("Delist Token", role: .destructive) { }
            } label: {
                Image(systemName: "ellipsis.circle")
                    .foregroundColor(.secondary)
            }
        }
        .padding(.vertical, 8)
    }
}

struct Token: Identifiable {
    let id: String
    let name: String
    let symbol: String
    let chain: String
    let price: Double
    let change24h: Double
    let marketCap: Double
    let status: String
    let isVerified: Bool
    let decimals: Int
    let contractAddress: String?
    
    var formattedPrice: String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        return formatter.string(from: NSNumber(value: price)) ?? "$0"
    }
    
    var formattedChange: String {
        let sign = change24h >= 0 ? "+" : ""
        return "\(sign)\(String(format: "%.2f", change24h))%"
    }
    
    var statusColor: Color {
        switch status.lowercased() {
        case "active": return .green
        case "paused": return .orange
        case "delisted": return .red
        default: return .gray
        }
    }
    
    var changeColor: Color {
        return change24h >= 0 ? .green : .red
    }
}

struct SearchBar: View {
    @Binding var text: String
    let placeholder: String
    
    var body: some View {
        HStack {
            Image(systemName: "magnifyingglass")
                .foregroundColor(.secondary)
            
            TextField(placeholder, text: $text)
                .textFieldStyle(.plain)
            
            if !text.isEmpty {
                Button(action: { text = "" }) {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundColor(.secondary)
                }
            }
        }
        .padding(10)
        .background(Color(.systemGray6))
        .cornerRadius(10)
    }
}

#Preview {
    NavigationView {
        TokensView()
            .environmentObject(ThemeManager())
    }
}
