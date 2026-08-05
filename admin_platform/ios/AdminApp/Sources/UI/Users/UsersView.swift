//
//  UsersView.swift
//  TigerWalletAdmin
//
//  Complete Users Management Screen
//

import SwiftUI

struct UsersView: View {
    @StateObject private var viewModel = UsersViewModel()
    @EnvironmentObject var themeManager: ThemeManager
    @State private var searchText = ""
    @State private var selectedFilter: UserFilter = .all
    
    enum UserFilter: String, CaseIterable {
        case all = "All"
        case active = "Active"
        case suspended = "Suspended"
        case pending = "Pending KYC"
    }
    
    var body: some View {
        VStack(spacing: 0) {
            // Search and Filter Bar
            HStack {
                SearchBar(text: $searchText, placeholder: "Search users...")
                
                Picker("Filter", selection: $selectedFilter) {
                    ForEach(UserFilter.allCases, id: \.self) { filter in
                        Text(filter.rawValue).tag(filter)
                    }
                }
                .pickerStyle(.menu)
            }
            .padding()
            
            if viewModel.isLoading {
                Spacer()
                ProgressView()
                Spacer()
            } else if viewModel.users.isEmpty {
                Spacer()
                Text("No users found")
                    .foregroundColor(.secondary)
                Spacer()
            } else {
                List {
                    ForEach(viewModel.filteredUsers(searchText: searchText, filter: selectedFilter)) { user in
                        UserRow(user: user) {
                            viewModel.toggleUserStatus(user)
                        }
                    }
                }
                .listStyle(.plain)
            }
        }
        .background(themeManager.backgroundColor)
        .navigationTitle("Users")
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button(action: { viewModel.loadUsers() }) {
                    Image(systemName: "arrow.clockwise")
                }
            }
        }
        .onAppear {
            viewModel.loadUsers()
        }
    }
}

struct UserRow: View {
    let user: User
    let onToggleStatus: () -> Void
    @EnvironmentObject var themeManager: ThemeManager
    @State private var showActions = false
    
    var body: some View {
        HStack(spacing: 12) {
            // Avatar
            Circle()
                .fill(user.statusColor.opacity(0.2))
                .frame(width: 50, height: 50)
                .overlay(
                    Text(String(user.username.prefix(1)).uppercased())
                        .font(.title3)
                        .fontWeight(.bold)
                        .foregroundColor(user.statusColor)
                )
            
            // User Info
            VStack(alignment: .leading, spacing: 4) {
                Text(user.username)
                    .font(.headline)
                
                Text(user.email)
                    .font(.caption)
                    .foregroundColor(.secondary)
                
                HStack(spacing: 8) {
                    StatusBadge(status: user.kycStatus)
                    
                    if user.isVerified {
                        Image(systemName: "checkmark.seal.fill")
                            .foregroundColor(.blue)
                            .font(.caption)
                    }
                }
            }
            
            Spacer()
            
            // Balance
            VStack(alignment: .trailing, spacing: 4) {
                Text(user.formattedBalance)
                    .font(.headline)
                
                Text("Volume: \(user.formattedVolume)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            // Actions Menu
            Menu {
                Button("View Details") { }
                Button("Edit User") { }
                Button(user.isSuspended ? "Activate" : "Suspend") {
                    onToggleStatus()
                }
                Button("View Transactions") { }
                Button("Reset KYC") { }
                Divider()
                Button("Delete User", role: .destructive) { }
            } label: {
                Image(systemName: "ellipsis.circle")
                    .foregroundColor(.secondary)
            }
        }
        .padding(.vertical, 8)
    }
}

struct StatusBadge: View {
    let status: String
    
    var body: some View {
        Text(status.capitalized)
            .font(.caption2)
            .fontWeight(.medium)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(statusColor.opacity(0.2))
            .foregroundColor(statusColor)
            .cornerRadius(4)
    }
    
    var statusColor: Color {
        switch status.lowercased() {
        case "verified", "approved", "active":
            return .green
        case "pending":
            return .orange
        case "suspended", "rejected":
            return .red
        default:
            return .gray
        }
    }
}

struct User: Identifiable {
    let id: String
    let username: String
    let email: String
    let balance: Double
    let tradingVolume: Double
    let kycStatus: String
    let isVerified: Bool
    let isSuspended: Bool
    let createdAt: Date
    
    var formattedBalance: String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        return formatter.string(from: NSNumber(value: balance)) ?? "$0"
    }
    
    var formattedVolume: String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        return formatter.string(from: NSNumber(value: tradingVolume)) ?? "0"
    }
    
    var statusColor: Color {
        if isSuspended { return .red }
        switch kycStatus.lowercased() {
        case "verified": return .green
        case "pending": return .orange
        default: return .gray
        }
    }
}

#Preview {
    NavigationView {
        UsersView()
            .environmentObject(ThemeManager())
    }
}
