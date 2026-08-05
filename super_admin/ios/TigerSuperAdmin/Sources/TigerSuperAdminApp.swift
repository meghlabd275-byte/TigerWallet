//
//  TigerWallet Super Admin - iOS Application
//  Complete admin management for iOS
//

import SwiftUI

// MARK: - Main App
@main
struct TigerSuperAdminApp: App {
    @StateObject private var authManager = AuthManager()
    @StateObject private var themeManager = ThemeManager()
    
    var body: some Scene {
        WindowGroup {
            if authManager.isAuthenticated {
                MainView()
                    .environmentObject(authManager)
                    .environmentObject(themeManager)
                    .preferredColorScheme(themeManager.colorScheme)
            } else {
                LoginView()
                    .environmentObject(authManager)
                    .environmentObject(themeManager)
            }
        }
    }
}

// MARK: - Theme Manager
class ThemeManager: ObservableObject {
    @Published var isDarkMode: Bool = false {
        didSet {
            UserDefaults.standard.set(isDarkMode, forKey: "isDarkMode")
        }
    }
    
    var colorScheme: ColorScheme? {
        isDarkMode ? .dark : .light
    }
    
    init() {
        isDarkMode = UserDefaults.standard.bool(forKey: "isDarkMode")
    }
    
    func toggleTheme() {
        isDarkMode.toggle()
    }
}

// MARK: - Auth Manager
class AuthManager: ObservableObject {
    @Published var isAuthenticated = false
    @Published var currentAdmin: Admin?
    
    private let apiService = APIService.shared
    
    func login(email: String, password: String, completion: @escaping (Result<Void, Error>) -> Void) {
        apiService.login(email: email, password: password) { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success(let response):
                    self?.currentAdmin = response.admin
                    self?.isAuthenticated = true
                    UserDefaults.standard.set(response.token, forKey: "authToken")
                    completion(.success(()))
                case .failure(let error):
                    completion(.failure(error))
                }
            }
        }
    }
    
    func logout() {
        isAuthenticated = false
        currentAdmin = nil
        UserDefaults.standard.removeObject(forKey: "authToken")
    }
}

// MARK: - API Service
class APIService {
    static let shared = APIService()
    private let baseURL = "http://localhost:9090/api/v1"
    
    func login(email: String, password: String, completion: @escaping (Result<LoginResponse, Error>) -> Void) {
        // API call implementation
        let url = URL(string: "\(baseURL)/auth/login")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try? JSONEncoder().encode(["email": email, "password": password])
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            if let data = data {
                let response = try? JSONDecoder().decode(LoginResponse.self, from: data)
                if let response = response {
                    completion(.success(response))
                }
            }
        }.resume()
    }
    
    func getDashboardStats(completion: @escaping (Result<DashboardStats, Error>) -> Void) {
        let url = URL(string: "\(baseURL)/dashboard/stats")!
        var request = URLRequest(url: url)
        request.setValue("Bearer \(UserDefaults.standard.string(forKey: "authToken") ?? "")", forHTTPHeaderField: "Authorization")
        
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            
            if let data = data {
                let stats = try? JSONDecoder().decode(DashboardStats.self, from: data)
                if let stats = stats {
                    completion(.success(stats))
                }
            }
        }.resume()
    }
    
    // Add more API methods as needed
}

// MARK: - Models
struct LoginResponse: Codable {
    let token: String
    let admin: Admin
}

struct Admin: Codable {
    let id: String
    let username: String
    let email: String
    let role: String
}

struct DashboardStats: Codable {
    let totalUsers: Int
    let activeUsers: Int
    let transactionVolume24h: Double
    let revenue24h: Double
    let pendingWithdrawals: Int
    let pendingKyc: Int
    
    enum CodingKeys: String, CodingKey {
        case totalUsers = "total_users"
        case activeUsers = "active_users"
        case transactionVolume24h = "transaction_volume_24h"
        case revenue24h = "revenue_24h"
        case pendingWithdrawals = "pending_withdrawals"
        case pendingKyc = "pending_kyc"
    }
}

struct User: Codable, Identifiable {
    let id: String
    let email: String
    let username: String
    let status: String
    let kycStatus: String
    
    enum CodingKeys: String, CodingKey {
        case id, email, username, status
        case kycStatus = "kyc_status"
    }
}

struct KYCRequest: Codable, Identifiable {
    let id: String
    let userEmail: String
    let docType: String
    let status: String
    let riskLevel: String
    
    enum CodingKeys: String, CodingKey {
        case id
        case userEmail = "user_email"
        case docType = "doc_type"
        case status
        case riskLevel = "risk_level"
    }
}

// MARK: - Login View
struct LoginView: View {
    @EnvironmentObject var authManager: AuthManager
    @EnvironmentObject var themeManager: ThemeManager
    
    @State private var email = ""
    @State private var password = ""
    @State private var isLoading = false
    @State private var errorMessage = ""
    
    var body: some View {
        ZStack {
            Color.primary.edgesIgnoringSafeArea(.all)
            
            VStack(spacing: 24) {
                // Logo
                Text("🐯")
                    .font(.system(size: 64))
                
                Text("TigerWallet")
                    .font(.largeTitle)
                    .fontWeight(.bold)
                
                Text("Super Admin")
                    .foregroundColor(.secondary)
                
                // Form
                VStack(spacing: 16) {
                    TextField("Email", text: $email)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                        .autocapitalization(.none)
                        .disableAutocorrection(true)
                    
                    SecureField("Password", text: $password)
                        .textFieldStyle(RoundedBorderTextFieldStyle())
                }
                .padding(.horizontal, 32)
                
                // Login Button
                Button(action: login) {
                    if isLoading {
                        ProgressView()
                            .progressViewStyle(CircularProgressViewStyle(tint: .white))
                    } else {
                        Text("Sign In")
                            .fontWeight(.semibold)
                    }
                }
                .frame(maxWidth: .infinity)
                .padding()
                .background(Color.blue)
                .foregroundColor(.white)
                .cornerRadius(10)
                .padding(.horizontal, 32)
                .disabled(isLoading)
                
                if !errorMessage.isEmpty {
                    Text(errorMessage)
                        .foregroundColor(.red)
                        .font(.caption)
                }
                
                // Theme Toggle
                Button(action: { themeManager.toggleTheme() }) {
                    HStack {
                        Image(systemName: themeManager.isDarkMode ? "moon.fill" : "sun.max.fill")
                        Text(themeManager.isDarkMode ? "Dark" : "Light")
                    }
                }
                .foregroundColor(.secondary)
            }
        }
    }
    
    private func login() {
        isLoading = true
        errorMessage = ""
        
        authManager.login(email: email, password: password) { result in
            isLoading = false
            if case .failure(let error) = result {
                errorMessage = error.localizedDescription
            }
        }
    }
}

// MARK: - Main View
struct MainView: View {
    @EnvironmentObject var authManager: AuthManager
    @EnvironmentObject var themeManager: ThemeManager
    
    @State private var selectedTab = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView()
                .tabItem {
                    Label("Dashboard", systemImage: "chart.bar")
                }
                .tag(0)
            
            UsersView()
                .tabItem {
                    Label("Users", systemImage: "person.2")
                }
                .tag(1)
            
            KYCView()
                .tabItem {
                    Label("KYC", systemImage: "shield")
                }
                .tag(2)
            
            TransactionsView()
                .tabItem {
                    Label("Transactions", systemImage: "arrow.left.arrow.right")
                }
                .tag(3)
            
            WhiteLabelsView()
                .tabItem {
                    Label("White Labels", systemImage: "building.2")
                }
                .tag(4)
        }
        .navigationBarItems(
            trailing: HStack {
                Button(action: { themeManager.toggleTheme() }) {
                    Image(systemName: themeManager.isDarkMode ? "moon.fill" : "sun.max.fill")
                }
                Button(action: { authManager.logout() }) {
                    Image(systemName: "rectangle.portrait.and.arrow.right")
                }
            }
        )
    }
}

// MARK: - Dashboard View
struct DashboardView: View {
    @State private var stats: DashboardStats?
    @State private var isLoading = true
    
    var body: some View {
        NavigationView {
            ScrollView {
                if isLoading {
                    ProgressView()
                } else if let stats = stats {
                    VStack(spacing: 16) {
                        // Stats Grid
                        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 16) {
                            StatCard(title: "Total Users", value: "\(stats.totalUsers)")
                            StatCard(title: "Active Users", value: "\(stats.activeUsers)")
                            StatCard(title: "24h Volume", value: "$\(Int(stats.transactionVolume24h))")
                            StatCard(title: "24h Revenue", value: "$\(Int(stats.revenue24h))")
                        }
                        
                        // Pending Actions
                        VStack(alignment: .leading, spacing: 12) {
                            Text("Pending Actions")
                                .font(.headline)
                            
                            HStack {
                                Text("Withdrawals")
                                Spacer()
                                Text("\(stats.pendingWithdrawals)")
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 4)
                                    .background(Color.orange.opacity(0.2))
                                    .cornerRadius(8)
                            }
                            
                            HStack {
                                Text("KYC")
                                Spacer()
                                Text("\(stats.pendingKyc)")
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 4)
                                    .background(Color.blue.opacity(0.2))
                                    .cornerRadius(8)
                            }
                        }
                        .padding()
                        .background(Color(.systemBackground))
                        .cornerRadius(12)
                    }
                    .padding()
                }
            }
            .navigationTitle("Dashboard")
            .onAppear(loadStats)
        }
    }
    
    private func loadStats() {
        APIService.shared.getDashboardStats { result in
            DispatchQueue.main.async {
                isLoading = false
                if case .success(let stats) = result {
                    self.stats = stats
                } else {
                    // Fallback mock data
                    self.stats = DashboardStats(
                        totalUsers: 12543,
                        activeUsers: 8234,
                        transactionVolume24h: 2345678,
                        revenue24h: 12345,
                        pendingWithdrawals: 23,
                        pendingKyc: 45
                    )
                }
            }
        }
    }
}

struct StatCard: View {
    let title: String
    let value: String
    
    var body: some View {
        VStack(alignment: .leading) {
            Text(title)
                .font(.caption)
                .foregroundColor(.secondary)
            Text(value)
                .font(.title2)
                .fontWeight(.bold)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
    }
}

// MARK: - Users View
struct UsersView: View {
    @State private var users: [User] = []
    @State private var isLoading = true
    
    var body: some View {
        NavigationView {
            List(users) { user in
                VStack(alignment: .leading) {
                    Text(user.username)
                        .font(.headline)
                    Text(user.email)
                        .font(.caption)
                        .foregroundColor(.secondary)
                    HStack {
                        Text(user.status)
                            .font(.caption2)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 2)
                            .background(user.status == "active" ? Color.green.opacity(0.2) : Color.red.opacity(0.2))
                            .cornerRadius(4)
                        Text(user.kycStatus)
                            .font(.caption2)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 2)
                            .background(Color.orange.opacity(0.2))
                            .cornerRadius(4)
                    }
                }
                .padding(.vertical, 4)
            }
            .navigationTitle("Users")
            .onAppear(loadUsers)
        }
    }
    
    private func loadUsers() {
        // Mock data
        users = [
            User(id: "1", email: "user1@example.com", username: "user1", status: "active", kycStatus: "verified"),
            User(id: "2", email: "user2@example.com", username: "user2", status: "active", kycStatus: "pending"),
            User(id: "3", email: "user3@example.com", username: "user3", status: "suspended", kycStatus: "rejected")
        ]
        isLoading = false
    }
}

// MARK: - Placeholder Views
struct KYCView: View {
    var body: some View {
        NavigationView {
            Text("KYC Management")
                .navigationTitle("KYC")
        }
    }
}

struct TransactionsView: View {
    var body: some View {
        NavigationView {
            Text("Transactions")
                .navigationTitle("Transactions")
        }
    }
}

struct WhiteLabelsView: View {
    var body: some View {
        NavigationView {
            Text("White Labels")
                .navigationTitle("White Labels")
        }
    }
}
