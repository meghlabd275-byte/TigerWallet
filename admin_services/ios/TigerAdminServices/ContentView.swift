//
//  ContentView.swift
//  TigerAdminServices
//

import SwiftUI

struct ContentView: View {
    @State private var selectedTab = 0
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView().tabItem { Label("Dashboard", systemImage: "house") }.tag(0)
            ServicesView().tabItem { Label("Services", systemImage: "gear") }.tag(1)
            HealthView().tabItem { Label("Health", systemImage: "heart") }.tag(2)
            LogsView().tabItem { Label("Logs", systemImage: "doc.text") }.tag(3)
            SettingsView().tabItem { Label("Settings", systemImage: "gear") }.tag(4)
        }
    }
}

struct DashboardView: View {
    var body: some View {
        NavigationView {
            VStack(spacing: 16) {
                StatCard(title: "Services", value: "0")
                StatCard(title: "Health", value: "OK")
                StatCard(title: "Uptime", value: "0h")
                StatCard(title: "Errors", value: "0")
                Spacer()
            }
            .padding()
            .navigationTitle("Admin Services")
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

struct ServicesView: View {
    var body: some View { NavigationView { List { Text("Loading...").foregroundColor(.secondary) }.navigationTitle("Services") } }
}

struct HealthView: View {
    var body: some View { NavigationView { List { Text("Loading...").foregroundColor(.secondary) }.navigationTitle("Health") } }
}

struct LogsView: View {
    var body: some View { NavigationView { List { Text("Loading...").foregroundColor(.secondary) }.navigationTitle("Logs") } }
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
