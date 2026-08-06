//
//  ContentView.swift
//  TigerMasterAdmin
//

import SwiftUI

struct ContentView: View {
    @State private var selectedTab = 0
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem {
                    Label("Dashboard", systemImage: "house")
                }
                .tag(0)
            
            WhiteLabelsView()
                .tabItem {
                    Label("White Labels", systemImage: "building.2")
                }
                .tag(1)
            
            AdminsView()
                .tabItem {
                    Label("Admins", systemImage: "person.2")
                }
                .tag(2)
            
            UsersView()
                .tabItem {
                    Label("Users", systemImage: "person.circle")
                }
                .tag(3)
            
            SettingsView()
                .tabItem {
                    Label("Settings", systemImage: "gear")
                }
                .tag(4)
        }
    }
}

struct DashboardView: View {
    var body: some View {
        NavigationView {
            VStack(spacing: 20) {
                StatCard(title: "Total White Labels", value: "0")
                StatCard(title: "Total Users", value: "0")
                StatCard(title: "Transactions", value: "0")
                StatCard(title: "Pending Approvals", value: "0")
                Spacer()
            }
            .padding()
            .navigationTitle("Dashboard")
        }
    }
}

struct StatCard: View {
    let title: String
    let value: String
    
    var body: some View {
        VStack(alignment: .leading) {
            Text(title)
                .font(.subheadline)
                .foregroundColor(.secondary)
            Text(value)
                .font(.largeTitle)
                .fontWeight(.bold)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(Color(.systemGray6))
        .cornerRadius(12)
    }
}

struct WhiteLabelsView: View {
    var body: some View {
        NavigationView {
            List {
                Text("Loading...").foregroundColor(.secondary)
            }
            .navigationTitle("White Labels")
        }
    }
}

struct AdminsView: View {
    var body: some View {
        NavigationView {
            List {
                Text("Loading...").foregroundColor(.secondary)
            }
            .navigationTitle("Master Admins")
        }
    }
}

struct UsersView: View {
    var body: some View {
        NavigationView {
            List {
                Text("Loading...").foregroundColor(.secondary)
            }
            .navigationTitle("Users")
        }
    }
}

struct SettingsView: View {
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        NavigationView {
            List {
                Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
                
                Section("About") {
                    HStack {
                        Text("Version")
                        Spacer()
                        Text("2.0.0").foregroundColor(.secondary)
                    }
                }
            }
            .navigationTitle("Settings")
        }
    }
}

struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView().environmentObject(ThemeManager())
    }
}
