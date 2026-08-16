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
    private let baseURL = "http://localhost:8082/api/v1/admin"
    
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
        let url = URL(string: "\(baseURL)/stats")!
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
    
    // MARK: - Generic domain governance API (12 domains on :8082)
    // A generic JSON payload/value used for all domain records. The Go backend
    // returns different per-domain array keys, so the list parser pulls the
    // first array it finds in the response object.
    typealias JSONDict = [String: Any]

    /// Issue a request to `baseURL/{resource}` (or `.../{resource}/{id}/...`)
    /// and deliver the parsed JSON or an error. Never fabricates data.
    func domainRequest(resource: String,
                       op: DomainOp,
                       id: String? = nil,
                       body: JSONDict? = nil,
                       completion: @escaping (Result<Any, Error>) -> Void) {
        var path = "\(baseURL)/\(resource)"
        if let id = id {
            path += "/\(id)"
            switch op {
            case .status: path += "/status"
            case .approve: path += "/approve"
            case .reject: path += "/reject"
            default: break
            }
        }
        guard let url = URL(string: path) else {
            completion(.failure(NSError(domain: "TigerAdmin", code: 0,
                                        userInfo: [NSLocalizedDescriptionKey: "Invalid URL"])))
            return
        }
        var request = URLRequest(url: url)
        switch op {
        case .list, .get: request.httpMethod = "GET"
        case .create, .approve, .reject: request.httpMethod = "POST"
        case .update, .status: request.httpMethod = "PUT"
        case .delete: request.httpMethod = "DELETE"
        }
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let token = UserDefaults.standard.string(forKey: "authToken") ?? ""
        if !token.isEmpty {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        if request.httpMethod != "GET" && request.httpMethod != "DELETE" {
            request.httpBody = try? JSONSerialization.data(withJSONObject: body ?? [:])
        }

        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                completion(.failure(error))
                return
            }
            guard let http = response as? HTTPURLResponse, let data = data else {
                completion(.failure(NSError(domain: "TigerAdmin", code: 0,
                                            userInfo: [NSLocalizedDescriptionKey: "No response from server"])))
                return
            }
            let parsed: Any? = (data.isEmpty ? nil
                                : try? JSONSerialization.jsonObject(with: data, options: []))
            if !(200..<300).contains(http.statusCode) {
                let msg = (parsed as? JSONDict)?["error"] as? String ?? "HTTP \(http.statusCode)"
                completion(.failure(NSError(domain: "TigerAdmin", code: http.statusCode,
                                            userInfo: [NSLocalizedDescriptionKey: msg])))
                return
            }
            completion(.success(parsed ?? NSNull()))
        }.resume()
    }

    /// Fetch the list of records for a domain as an array of dictionaries.
    func domainList(resource: String, completion: @escaping (Result<[JSONDict], Error>) -> Void) {
        domainRequest(resource: resource, op: .list) { result in
            switch result {
            case .success(let value):
                completion(.success(Self.extractArray(value)))
            case .failure(let error):
                completion(.failure(error))
            }
        }
    }

    static func extractArray(_ value: Any) -> [JSONDict] {
        if let arr = value as? [JSONDict] { return arr }
        if let dict = value as? JSONDict {
            for v in dict.values {
                if let arr = v as? [JSONDict] { return arr }
            }
        }
        return []
    }
}

/// Operations supported by every governance domain.
enum DomainOp {
    case list, get, create, update, delete, status, approve, reject
}

/// Metadata for the 12 governance domains and the actions each supports.
struct AdminDomain: Identifiable, Hashable {
    let id: String
    let label: String
    let resource: String
    let actions: [String]
}

let adminDomains: [AdminDomain] = [
    AdminDomain(id: "futures", label: "Futures", resource: "futures", actions: ["status"]),
    AdminDomain(id: "options", label: "Options", resource: "options", actions: ["status"]),
    AdminDomain(id: "copy-trading", label: "Copy Trading", resource: "copy-trading", actions: ["status"]),
    AdminDomain(id: "convert", label: "Convert", resource: "convert", actions: ["status"]),
    AdminDomain(id: "onramp", label: "Onramp", resource: "onramp", actions: ["approve", "reject"]),
    AdminDomain(id: "offramp", label: "Offramp", resource: "offramp", actions: ["approve", "reject"]),
    AdminDomain(id: "p2p-clients", label: "P2P Clients", resource: "p2p-clients", actions: ["status"]),
    AdminDomain(id: "partners", label: "Partners", resource: "partners", actions: ["status", "approve", "reject"]),
    AdminDomain(id: "rewards", label: "Rewards", resource: "rewards", actions: ["status"]),
    AdminDomain(id: "marketing", label: "Marketing", resource: "marketing", actions: ["status"]),
    AdminDomain(id: "admin-roles", label: "Admin Roles", resource: "admin-roles", actions: []),
    AdminDomain(id: "wl-control", label: "WL Control", resource: "wl-clients", actions: ["status"])
]

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

            DomainsView()
                .tabItem {
                    Label("Domains", systemImage: "slider.horizontal.3")
                }
                .tag(5)
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
    @State private var errorMessage = ""
    
    var body: some View {
        NavigationView {
            ScrollView {
                if isLoading {
                    ProgressView()
                } else if !errorMessage.isEmpty {
                    Text(errorMessage)
                        .foregroundColor(.red)
                        .padding()
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
                switch result {
                case .success(let stats):
                    self.stats = stats
                case .failure(let error):
                    self.errorMessage = error.localizedDescription
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

// MARK: - Governance Domains Navigation
struct DomainsView: View {
    var body: some View {
        NavigationView {
            List(adminDomains) { domain in
                NavigationLink(destination: DomainScreen(domain: domain)) {
                    Label(domain.label, systemImage: "circle.grid.cross.fill")
                }
            }
            .navigationTitle("Domains")
        }
    }
}

// MARK: - Generic Domain Screen (loading / error / empty + CRUD + actions)
struct DomainScreen: View {
    let domain: AdminDomain

    @State private var records: [APIService.JSONDict] = []
    @State private var isLoading = true
    @State private var errorMessage = ""
    @State private var showingCreate = false
    @State private var editing: APIService.JSONDict?
    @State private var statusTarget: String?
    @State private var rejectTarget: String?

    var body: some View {
        List {
            if isLoading {
                HStack { Spacer(); ProgressView(); Spacer() }
            } else if !errorMessage.isEmpty {
                Text(errorMessage).foregroundColor(.red)
            } else if records.isEmpty {
                Text("No \(domain.label.lowercased()) records found.")
                    .foregroundColor(.secondary)
            } else {
                ForEach(Array(records.enumerated()), id: \.offset) { _, record in
                    DomainRow(domain: domain,
                              record: record,
                              onEdit: { editing = record },
                              onDelete: { delete(record) },
                              onStatus: { statusTarget = idOf(record) },
                              onApprove: { approve(record) },
                              onReject: { rejectTarget = idOf(record) })
                }
            }
        }
        .navigationTitle(domain.label)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button(action: { showingCreate = true }) { Image(systemName: "plus") }
            }
            ToolbarItem(placement: .navigationBarLeading) {
                Button(action: load) { Image(systemName: "arrow.clockwise") }
            }
        }
        .sheet(isPresented: $showingCreate) {
            DomainEditorView(domain: domain, record: nil) { body in create(body) }
        }
        .sheet(item: Binding(get: { editing.map { DomainRecord(domain: domain, dict: $0) } },
                             set: { editing = $0?.dict })) { wrapped in
            DomainEditorView(domain: domain, record: wrapped.dict) { body in update(wrapped.dict, body) }
        }
        .sheet(item: Binding(get: { statusTarget.map { DomainId(domain: domain, id: $0) } },
                             set: { statusTarget = $0?.id })) { wrapped in
            StatusPickerView { status in setStatus(wrapped.id, status) }
        }
        .sheet(item: Binding(get: { rejectTarget.map { DomainId(domain: domain, id: $0) } },
                             set: { rejectTarget = $0?.id })) { wrapped in
            RejectReasonView { reason in reject(wrapped.id, reason) }
        }
        .onAppear { if records.isEmpty && isLoading { load() } }
    }

    private func load() {
        isLoading = true
        errorMessage = ""
        APIService.shared.domainList(resource: domain.resource) { result in
            DispatchQueue.main.async {
                isLoading = false
                switch result {
                case .success(let rows): self.records = rows
                case .failure(let error): self.errorMessage = error.localizedDescription
                }
            }
        }
    }

    private func idOf(_ record: APIService.JSONDict) -> String {
        (record["id"] as? String)
            ?? (record["id"] as? NSNumber).map { $0.stringValue }
            ?? (record["uuid"] as? String)
            ?? ""
    }

    private func create(_ body: APIService.JSONDict) {
        APIService.shared.domainRequest(resource: domain.resource, op: .create, body: body) { result in
            DispatchQueue.main.async { if case .failure(let e) = result { errorMessage = e.localizedDescription } else { load() } }
        }
    }

    private func update(_ record: APIService.JSONDict, _ body: APIService.JSONDict) {
        APIService.shared.domainRequest(resource: domain.resource, op: .update, id: idOf(record), body: body) { result in
            DispatchQueue.main.async { if case .failure(let e) = result { errorMessage = e.localizedDescription } else { load() } }
        }
    }

    private func delete(_ record: APIService.JSONDict) {
        APIService.shared.domainRequest(resource: domain.resource, op: .delete, id: idOf(record)) { result in
            DispatchQueue.main.async { if case .failure(let e) = result { errorMessage = e.localizedDescription } else { load() } }
        }
    }

    private func setStatus(_ id: String, _ status: String) {
        APIService.shared.domainRequest(resource: domain.resource, op: .status, id: id, body: ["status": status]) { result in
            DispatchQueue.main.async { if case .failure(let e) = result { errorMessage = e.localizedDescription } else { load() } }
        }
    }

    private func approve(_ record: APIService.JSONDict) {
        APIService.shared.domainRequest(resource: domain.resource, op: .approve, id: idOf(record), body: [:]) { result in
            DispatchQueue.main.async { if case .failure(let e) = result { errorMessage = e.localizedDescription } else { load() } }
        }
    }

    private func reject(_ id: String, _ reason: String) {
        APIService.shared.domainRequest(resource: domain.resource, op: .reject, id: id, body: ["reason": reason]) { result in
            DispatchQueue.main.async { if case .failure(let e) = result { errorMessage = e.localizedDescription } else { load() } }
        }
    }
}

private struct DomainRow: View {
    let domain: AdminDomain
    let record: APIService.JSONDict
    let onEdit: () -> Void
    let onDelete: () -> Void
    let onStatus: () -> Void
    let onApprove: () -> Void
    let onReject: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            ForEach(topFields, id: \.0) { field, value in
                HStack {
                    Text(field).font(.caption).foregroundColor(.secondary)
                    Spacer()
                    Text(value).font(.caption.monospaced())
                }
            }
            HStack {
                Button("Edit", action: onEdit).font(.caption)
                Button("Delete", role: .destructive, action: onDelete).font(.caption)
                if domain.actions.contains("status") {
                    Button("Status", action: onStatus).font(.caption)
                }
                if domain.actions.contains("approve") {
                    Button("Approve", action: onApprove).font(.caption)
                }
                if domain.actions.contains("reject") {
                    Button("Reject", role: .destructive, action: onReject).font(.caption)
                }
            }
        }
    }

    private var topFields: [(String, String)] {
        let keys = Array(record.keys).sorted()
        return keys.prefix(6).map { k in (k, formatValue(record[k])) }
    }
}

// MARK: - Editor / Status / Reject sheets
private struct DomainEditorView: View {
    let domain: AdminDomain
    let record: APIService.JSONDict?
    let onSave: (APIService.JSONDict) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var fields: [String: String] = [:]

    private let presets: [String: [String]] = [
        "futures": ["pair", "side", "size", "leverage", "entry_price", "liquidation_price", "margin", "chain_id"],
        "options": ["underlying", "option_type", "strike", "expiry", "premium", "size", "chain_id"],
        "copy-trading": ["follower_id", "leader_id", "allocation", "max_leverage"],
        "convert": ["user_id", "from_token", "to_token", "from_amount", "to_amount", "rate", "chain_id"],
        "onramp": ["user_id", "provider", "fiat_currency", "crypto_token", "fiat_amount", "crypto_amount"],
        "offramp": ["user_id", "provider", "crypto_token", "fiat_currency", "crypto_amount", "fiat_amount"],
        "p2p-clients": ["user_id", "username"],
        "partners": ["name", "contact_email", "revenue_share"],
        "rewards": ["name", "reward_type", "amount", "token", "start_at", "end_at"],
        "marketing": ["name", "channel", "budget", "start_at", "end_at"],
        "admin-roles": ["name", "description"],
        "wl-control": ["name", "domain"]
    ]

    private var fieldNames: [String] {
        if let r = record { return Array(r.keys).sorted() }
        return presets[domain.id] ?? ["name"]
    }

    var body: some View {
        NavigationView {
            Form {
                ForEach(fieldNames, id: \.self) { name in
                    TextField(name, text: Binding(
                        get: { fields[name] ?? (record?[name].map { formatValue($0) } ?? "") },
                        set: { fields[name] = $0 }
                    ))
                }
            }
            .navigationTitle(record == nil ? "New \(domain.label)" : "Edit \(domain.label)")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        var body: APIService.JSONDict = record ?? [:]
                        for (k, v) in fields { body[k] = v }
                        onSave(body)
                        dismiss()
                    }
                }
            }
        }
    }
}

private struct StatusPickerView: View {
    let onSelect: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var status = "pending"
    private let options = ["pending", "active", "paused", "inactive", "suspended", "completed", "failed"]

    var body: some View {
        NavigationView {
            Form {
                Picker("Status", selection: $status) {
                    ForEach(options, id: \.self) { Text($0).tag($0) }
                }
            }
            .navigationTitle("Set Status")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Apply") { onSelect(status); dismiss() }
                }
            }
        }
    }
}

private struct RejectReasonView: View {
    let onSubmit: (String) -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var reason = ""

    var body: some View {
        NavigationView {
            Form {
                TextField("Reason", text: $reason, axis: .vertical)
                    .lineLimit(3...6)
            }
            .navigationTitle("Reject Record")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit") { if !reason.trimmingCharacters(in: .whitespaces).isEmpty { onSubmit(reason); dismiss() } }
                        .disabled(reason.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
        }
    }
}

// MARK: - Helpers for wrapping dicts/ids as sheet items
private struct DomainRecord: Identifiable {
    let domain: AdminDomain
    let dict: APIService.JSONDict
    var id: String { String(ObjectIdentifier(domain).hashValue) + "|" + (dict["id"].map { formatValue($0) } ?? UUID().uuidString) }
}

private struct DomainId: Identifiable {
    let domain: AdminDomain
    let id: String
    var hash: String { domain.id + "|" + id }
}

private func formatValue(_ v: Any?) -> String {
    guard let v = v else { return "—" }
    if let s = v as? String { return s }
    if let n = v as? NSNumber { return n.stringValue }
    if let b = v as? Bool { return b ? "yes" : "no" }
    if let arr = v as? [Any] { return "[\(arr.count) items]" }
    if let dict = v as? [String: Any] { return "{\(dict.count) fields}" }
    return String(describing: v)
}
