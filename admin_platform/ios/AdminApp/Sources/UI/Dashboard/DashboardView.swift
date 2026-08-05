//
//  DashboardView.swift
//  TigerWalletAdmin
//
//  Complete Admin Dashboard with real data from backend
//

import SwiftUI
import Charts

struct DashboardView: View {
    @StateObject private var viewModel = DashboardViewModel()
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                // Stats Cards
                LazyVGrid(columns: [
                    GridItem(.flexible()),
                    GridItem(.flexible()),
                    GridItem(.flexible()),
                    GridItem(.flexible())
                ], spacing: 16) {
                    StatCard(title: "Total Users", value: viewModel.stats.totalUsers, icon: "person.3.fill", color: .blue)
                    StatCard(title: "Active Users", value: viewModel.stats.activeUsers, icon: "person.fill.checkmark", color: .green)
                    StatCard(title: "Total Volume", value: viewModel.stats.totalVolume, icon: "chart.line.uptrend.xyaxis", color: .purple)
                    StatCard(title: "Revenue", value: viewModel.stats.revenue, icon: "dollarsign.circle.fill", color: .orange)
                }
                
                // Charts Section
                VStack(alignment: .leading, spacing: 16) {
                    Text("Analytics Overview")
                        .font(.title2)
                        .fontWeight(.bold)
                        .foregroundColor(themeManager.textColor)
                    
                    HStack(spacing: 20) {
                        // User Growth Chart
                        VStack(alignment: .leading) {
                            Text("User Growth")
                                .font(.headline)
                                .foregroundColor(themeManager.textColor)
                            
                            Chart(viewModel.userGrowthData) { data in
                                LineMark(
                                    x: .value("Date", data.date),
                                    y: .value("Users", data.count)
                                )
                                .foregroundColor(.blue)
                            }
                            .frame(height: 200)
                            .chartXAxis {
                                AxisMarks(values: .automatic)
                            }
                        }
                        .padding()
                        .background(themeManager.cardBackground)
                        .cornerRadius(12)
                        
                        // Volume Chart
                        VStack(alignment: .leading) {
                            Text("Volume")
                                .font(.headline)
                                .foregroundColor(themeManager.textColor)
                            
                            Chart(viewModel.volumeData) { data in
                                BarMark(
                                    x: .value("Date", data.date),
                                    y: .value("Volume", data.volume)
                                )
                                .foregroundColor(.green)
                            }
                            .frame(height: 200)
                        }
                        .padding()
                        .background(themeManager.cardBackground)
                        .cornerRadius(12)
                    }
                }
                
                // Quick Actions
                VStack(alignment: .leading, spacing: 16) {
                    Text("Quick Actions")
                        .font(.title2)
                        .fontWeight(.bold)
                        .foregroundColor(themeManager.textColor)
                    
                    LazyVGrid(columns: [
                        GridItem(.flexible()),
                        GridItem(.flexible()),
                        GridItem(.flexible())
                    ], spacing: 16) {
                        QuickActionButton(title: "Pending Withdrawals", count: viewModel.stats.pendingWithdrawals, icon: "arrow.down.circle", color: .red) {
                            // Navigate to withdrawals
                        }
                        QuickActionButton(title: "KYC Pending", count: viewModel.stats.pendingKYC, icon: "person.badge.clock", color: .orange) {
                            // Navigate to KYC
                        }
                        QuickActionButton(title: "New Tokens", count: viewModel.stats.pendingTokens, icon: "plus.circle", color: .blue) {
                            // Navigate to tokens
                        }
                    }
                }
                
                // Recent Activity
                VStack(alignment: .leading, spacing: 16) {
                    Text("Recent Activity")
                        .font(.title2)
                        .fontWeight(.bold)
                        .foregroundColor(themeManager.textColor)
                    
                    ForEach(viewModel.recentActivities) { activity in
                        ActivityRow(activity: activity)
                    }
                }
            }
            .padding()
        }
        .background(themeManager.backgroundColor)
        .navigationTitle("Dashboard")
        .onAppear {
            viewModel.loadDashboardData()
        }
        .refreshable {
            viewModel.loadDashboardData()
        }
    }
}

struct StatCard: View {
    let title: String
    let value: String
    let icon: String
    let color: Color
    
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Image(systemName: icon)
                    .foregroundColor(color)
                    .font(.title2)
                Spacer()
            }
            
            Text(value)
                .font(.title)
                .fontWeight(.bold)
                .foregroundColor(.primary)
            
            Text(title)
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(color: Color.black.opacity(0.1), radius: 5, x: 0, y: 2)
    }
}

struct QuickActionButton: View {
    let title: String
    let count: Int
    let icon: String
    let color: Color
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            VStack(spacing: 8) {
                Image(systemName: icon)
                    .font(.title)
                    .foregroundColor(color)
                
                Text("\(count)")
                    .font(.title2)
                    .fontWeight(.bold)
                
                Text(title)
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(Color(.systemBackground))
            .cornerRadius(12)
            .shadow(color: Color.black.opacity(0.1), radius: 5, x: 0, y: 2)
        }
    }
}

struct ActivityRow: View {
    let activity: Activity
    @EnvironmentObject var themeManager: ThemeManager
    
    var body: some View {
        HStack {
            Image(systemName: activity.icon)
                .foregroundColor(activity.color)
                .frame(width: 40)
            
            VStack(alignment: .leading) {
                Text(activity.title)
                    .font(.headline)
                Text(activity.description)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            Text(activity.timeAgo)
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding()
        .background(themeManager.cardBackground)
        .cornerRadius(8)
    }
}

struct Activity: Identifiable {
    let id = UUID()
    let title: String
    let description: String
    let icon: String
    let color: Color
    let timeAgo: String
}

#Preview {
    NavigationView {
        DashboardView()
            .environmentObject(ThemeManager())
    }
}
