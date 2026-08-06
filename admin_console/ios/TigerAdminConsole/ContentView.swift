import SwiftUI

struct ContentView: View {
    var body: some View {
        NavigationView {
            List {
                NavigationLink(destination: DashboardView()) {
                    Label("Dashboard", systemImage: "chart.bar")
                }
                NavigationLink(destination: UsersView()) {
                    Label("Users", systemImage: "person.2")
                }
                NavigationLink(destination: SettingsView()) {
                    Label("Settings", systemImage: "gear")
                }
            }
            .navigationTitle("Admin Console")
            .navigationDestination(for: String.self) { destination in
                if destination == "dashboard" { DashboardView() }
                else if destination == "users" { UsersView() }
                else { SettingsView() }
            }
        }
    }
}

struct DashboardView: View {
    var body: some View {
        VStack(spacing: 20) {
            Text("Dashboard").font(.largeTitle)
            HStack(spacing: 20) {
                StatCard(title: "Users", value: "0")
                StatCard(title: "Tokens", value: "0")
            }
        }.padding()
    }
}

struct UsersView: View {
    var body: some View {
        Text("Users").font(.largeTitle)
    }
}

struct SettingsView: View {
    var body: some View {
        Text("Settings").font(.largeTitle)
    }
}

struct StatCard: View {
    let title: String
    let value: String
    
    var body: some View {
        VStack {
            Text(value).font(.title)
            Text(title).foregroundColor(.secondary)
        }
        .frame(width: 120, height: 80)
        .background(Color.gray.opacity(0.1))
        .cornerRadius(10)
    }
}
