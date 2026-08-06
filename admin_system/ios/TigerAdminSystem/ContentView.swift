//
//  ContentView.swift
//  TigerAdminSystem
//

import SwiftUI

struct ContentView: View {
    @State private var selectedTab = 0
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView().tabItem { Label("Dashboard", systemImage: "house") }.tag(0)
            UsersView().tabItem { Label("Users", systemImage: "person.circle") }.tag(1)
            ConfigView().tabItem { Label("Config", systemImage: "gear") }.tag(2)
            AuditView().tabItem { Label("Audit", systemImage: "doc.text") }.tag(3)
            SettingsView().tabItem { Label("Settings", systemImage: "gear") }.tag(4)
        }
    }
}

struct DashboardView: View {
    var body: some View {
        NavigationView {
            VStack(spacing: 16) {
                StatCard(title: "Users", value: "0")
                StatCard(title: "Configurations", value: "0")
                StatCard(title: "Audit Logs", value: "0")
                StatCard(title: "System", value: "OK")
                Spacer()
            }
            .padding()
            .navigationTitle("Admin System")
        }
    }
}

struct StatCard: View {
    let title: String
    let value: String
    
    var body: some View {
        VStack(alignment: .leading) {
            Text(title).font(.subheadline).foregroundColor(.secondary)
            Text(value).font(.largeTitle).fontWeight(.bold)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(Color(.systemGray6))
        .cornerRadius(12)
    }
}

struct UsersView: View {
    var body: some View { NavigationView { List { Text("Loading...").foregroundColor(.secondary) }.navigationTitle("Users") } }
}

struct ConfigView: View {
    var body: some View { NavigationView { List { Text("Loading...").foregroundColor(.secondary) }.navigationTitle("Configuration") } }
}

struct AuditView: View {
    var body: some View { NavigationView { List { Text("Loading...").foregroundColor(.secondary) }.navigationTitle("Audit Logs") } }
}

struct SettingsView: View {
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        NavigationView {
            List {
                Toggle("Dark Mode", isOn: $themeManager.isDarkMode)
                Section("About") { HStack { Text("Version"); Spacer(); Text("2.0.0").foregroundColor(.secondary) } }
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
