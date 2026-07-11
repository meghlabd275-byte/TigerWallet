//
//  TigerWalletApp.swift
//  TigerWallet
//

import SwiftUI

@main
struct TigerWalletApp: App {
    var body: some Scene {
        WindowGroup { ContentView() }
    }
}

struct ContentView: View {
    @State private var balance: Double = 12450.00
    
    var body: some View {
        NavigationView {
            ZStack {
                Color(.systemBackground).ignoresSafeArea()
                VStack(spacing: 20) {
                    HStack { Text("🐯").font(.largeTitle); Text("TigerWallet").font(.title).fontWeight(.bold); Spacer(); Image(systemName: "gearshape.fill").foregroundColor(.gray) }
                    .padding()
                    VStack(spacing: 8) {
                        Text("Total Balance").font(.subheadline).foregroundColor(.gray)
                        Text("$\(balance, specifier: "%.2f")").font(.system(size: 40, weight: .bold))
                        Text("4.2 ETH").foregroundColor(.gray)
                    }.frame(maxWidth: .infinity).padding(.vertical, 30).background(Color.orange.opacity(0.1)).cornerRadius(20).padding(.horizontal)
                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 15) {
                        ActionButton(icon: "arrow.up.circle.fill", title: "Send", color: .orange)
                        ActionButton(icon: "arrow.down.circle.fill", title: "Receive", color: .green)
                        ActionButton(icon: "arrow.triangle.2.circlepath", title: "Swap", color: .blue)
                        ActionButton(icon: "chart.pie.fill", title: "Portfolio", color: .purple)
                    }.padding(.horizontal)
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Chains").font(.headline).padding(.horizontal)
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 10) {
                                ChainChip(name: "ETH", icon: "⬡", active: true)
                                ChainChip(name: "BNB", icon: "🟡", active: false)
                                ChainChip(name: "SOL", icon: "☀️", active: false)
                            }.padding(.horizontal)
                        }
                    }
                    Spacer()
                    HStack {
                        TabBarItem(icon: "wallet.pass.fill", title: "Wallet", active: true)
                        TabBarItem(icon: "chart.line.uptrend.xyaxis", title: "Trade", active: false)
                        TabBarItem(icon: "square.grid.2x2", title: "DApps", active: false)
                        TabBarItem(icon: "person.fill", title: "Profile", active: false)
                    }.padding()
                }
            }
        }
    }
}

struct ActionButton: View {
    let icon: String; let title: String; let color: Color
    var body: some View {
        VStack(spacing: 8) {
            Image(systemName: icon).font(.system(size: 30)).foregroundColor(color)
            Text(title).font(.caption)
        }.frame(maxWidth: .infinity).padding(.vertical, 20).background(Color(.secondarySystemBackground)).cornerRadius(15)
    }
}

struct ChainChip: View {
    let name: String; let icon: String; let active: Bool
    var body: some View {
        HStack(spacing: 6) { Text(icon); Text(name).font(.caption).fontWeight(.medium) }.padding(.horizontal, 12).padding(.vertical, 8).background(active ? Color.orange : Color(.secondarySystemBackground)).foregroundColor(active ? .white : .primary).cornerRadius(20)
    }
}

struct TabBarItem: View {
    let icon: String; let title: String; let active: Bool
    var body: some View {
        VStack(spacing: 4) { Image(systemName: icon).font(.system(size: 20)); Text(title).font(.caption2) }.foregroundColor(active ? .orange : .gray).frame(maxWidth: .infinity)
    }
}
