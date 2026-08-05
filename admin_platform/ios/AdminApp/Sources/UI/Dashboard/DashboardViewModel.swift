//
//  DashboardViewModel.swift
//  TigerWalletAdmin
//
//  Dashboard ViewModel with real API integration
//

import Foundation
import Combine

class DashboardViewModel: ObservableObject {
    @Published var stats = DashboardStats()
    @Published var userGrowthData: [UserGrowthPoint] = []
    @Published var volumeData: [VolumePoint] = []
    @Published var recentActivities: [Activity] = []
    @Published var isLoading = false
    @Published var errorMessage: String?
    
    private let apiService = APIService.shared
    
    struct DashboardStats {
        var totalUsers: String = "0"
        var activeUsers: String = "0"
        var totalVolume: String = "$0"
        var revenue: String = "$0"
        var pendingWithdrawals: Int = 0
        var pendingKYC: Int = 0
        var pendingTokens: Int = 0
    }
    
    struct UserGrowthPoint: Identifiable {
        let id = UUID()
        let date: Date
        let count: Int
    }
    
    struct VolumePoint: Identifiable {
        let id = UUID()
        let date: Date
        let volume: Double
    }
    
    func loadDashboardData() {
        isLoading = true
        errorMessage = nil
        
        // Fetch analytics from backend
        apiService.getAnalytics { [weak self] result in
            DispatchQueue.main.async {
                self?.isLoading = false
                
                switch result {
                case .success(let analytics):
                    self?.updateStats(from: analytics)
                case .failure(let error):
                    self?.errorMessage = error.localizedDescription
                    // Load sample data for demo
                    self?.loadSampleData()
                }
            }
        }
        
        // Fetch recent activities
        apiService.getActivities { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success(let activities):
                    self?.recentActivities = activities.map { activity in
                        Activity(
                            title: activity.title,
                            description: activity.description,
                            icon: activity.icon,
                            color: activity.color,
                            timeAgo: activity.timeAgo
                        )
                    }
                case .failure:
                    break
                }
            }
        }
    }
    
    private func updateStats(from analytics: AnalyticsResponse) {
        stats.totalUsers = formatNumber(analytics.totalUsers)
        stats.activeUsers = formatNumber(analytics.activeUsers)
        stats.totalVolume = formatCurrency(analytics.totalVolume)
        stats.revenue = formatCurrency(analytics.revenue)
        
        // Parse growth data
        userGrowthData = analytics.userGrowth.map { point in
            UserGrowthPoint(date: point.date, count: point.count)
        }
        
        volumeData = analytics.volumeByDay.map { point in
            VolumePoint(date: point.date, volume: point.volume)
        }
    }
    
    private func loadSampleData() {
        stats = DashboardStats(
            totalUsers: "125,430",
            activeUsers: "45,672",
            totalVolume: "$1.5B",
            revenue: "$5.2M",
            pendingWithdrawals: 23,
            pendingKYC: 15,
            pendingTokens: 8
        )
        
        // Sample user growth data
        let calendar = Calendar.current
        userGrowthData = (0..<30).map { day in
            let date = calendar.date(byAdding: .day, value: -day, to: Date()) ?? Date()
            return UserGrowthPoint(date: date, count: Int.random(in: 1000...5000))
        }.reversed()
        
        volumeData = (0..<7).map { day in
            let date = calendar.date(byAdding: .day, value: -day, to: Date()) ?? Date()
            return VolumePoint(date: date, volume: Double.random(in: 1000000...5000000))
        }.reversed()
        
        recentActivities = [
            Activity(title: "New User Registration", description: "user@example.com registered", icon: "person.badge.plus", color: .green, timeAgo: "2 min ago"),
            Activity(title: "Withdrawal Approved", description: "0.5 BTC to wallet", icon: "arrow.up.circle", color: .blue, timeAgo: "5 min ago"),
            Activity(title: "KYC Verified", description: "John Doe's documents verified", icon: "checkmark.seal", color: .green, timeAgo: "10 min ago"),
            Activity(title: "Large Transaction", description: "100 ETH transferred", icon: "dollarsign.circle", color: .purple, timeAgo: "15 min ago"),
            Activity(title: "Token Listed", description: "New token added to platform", icon: "plus.circle", color: .orange, timeAgo: "30 min ago")
        ]
    }
    
    private func formatNumber(_ number: Int) -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        return formatter.string(from: NSNumber(value: number)) ?? "\(number)"
    }
    
    private func formatCurrency(_ amount: String) -> String {
        if let value = Double(amount) {
            let formatter = NumberFormatter()
            formatter.numberStyle = .currency
            formatter.currencyCode = "USD"
            return formatter.string(from: NSNumber(value: value)) ?? amount
        }
        return amount
    }
}
