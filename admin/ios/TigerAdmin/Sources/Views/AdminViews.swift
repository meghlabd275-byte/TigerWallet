import SwiftUI

// MARK: - Main Tab View

struct AdminMainView: View {
    @StateObject private var dashboardVM = DashboardViewModel()
    @StateObject private var usersVM = UsersViewModel()
    @StateObject private var transactionsVM = TransactionsViewModel()
    @StateObject private var kycVM = KYCViewModel()
    @StateObject private var tokensVM = TokensViewModel()
    @StateObject private var withdrawalsVM = WithdrawalsViewModel()
    @StateObject private var systemVM = SystemViewModel()
    
    @State private var selectedTab = 0
    
    var body: some View {
        TabView(selection: $selectedTab) {
            DashboardView(viewModel: dashboardVM)
                .tabItem {
                    Label("Dashboard", systemImage: "square.grid.2x2")
                }
                .tag(0)
            
            UsersView(viewModel: usersVM)
                .tabItem {
                    Label("Users", systemImage: "person.2")
                }
                .tag(1)
            
            TransactionsView(viewModel: transactionsVM)
                .tabItem {
                    Label("Transactions", systemImage: "arrow.left.arrow.right")
                }
                .tag(2)
            
            KYCView(viewModel: kycVM)
                .tabItem {
                    Label("KYC", systemImage: "checkmark.shield")
                }
                .tag(3)
            
            TokensView(viewModel: tokensVM)
                .tabItem {
                    Label("Tokens", systemImage: "bitcoinsign.circle")
                }
                .tag(4)
            
            WithdrawalsView(viewModel: withdrawalsVM)
                .tabItem {
                    Label("Withdrawals", systemImage: "arrow.down.circle")
                }
                .tag(5)
            
            SystemView(viewModel: systemVM)
                .tabItem {
                    Label("System", systemImage: "server.rack")
                }
                .tag(6)
        }
    }
}

// MARK: - Dashboard View

struct DashboardView: View {
    @ObservedObject var viewModel: DashboardViewModel
    
    var body: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 20) {
                    if let data = viewModel.analyticsData {
                        // Stats Grid
                        LazyVGrid(columns: [
                            GridItem(.flexible()),
                            GridItem(.flexible())
                        ], spacing: 16) {
                            StatCard(
                                title: "Total Users",
                                value: "\(data.totalUsers)",
                                icon: "person.2.fill",
                                color: .blue
                            )
                            StatCard(
                                title: "Active Users",
                                value: "\(data.activeUsers)",
                                icon: "person.fill.checkmark",
                                color: .green
                            )
                            StatCard(
                                title: "Total Volume",
                                value: data.totalVolume,
                                icon: "dollarsign.circle.fill",
                                color: .orange
                            )
                            StatCard(
                                title: "Daily Transactions",
                                value: "\(data.dailyTransactions)",
                                icon: "arrow.left.arrow.right.circle.fill",
                                color: .purple
                            )
                            StatCard(
                                title: "Total Fees",
                                value: data.totalFees,
                                icon: "creditcard.fill",
                                color: .red
                            )
                            StatCard(
                                title: "Pending KYC",
                                value: "\(data.pendingKyc)",
                                icon: "clock.fill",
                                color: .yellow
                            )
                        }
                    }
                    
                    if viewModel.isLoading {
                        ProgressView()
                            .padding()
                    }
                }
                .padding()
            }
            .navigationTitle("Dashboard")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.refresh() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .alert("Error", isPresented: $viewModel.isError) {
                Button("OK") { viewModel.clearError() }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
    }
}

struct StatCard: View {
    let title: String
    let value: String
    let icon: String
    let color: Color
    
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Image(systemName: icon)
                    .font(.title2)
                    .foregroundColor(color)
                Spacer()
            }
            Text(title)
                .font(.caption)
                .foregroundColor(.secondary)
            Text(value)
                .font(.title2)
                .fontWeight(.bold)
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(color: Color.black.opacity(0.1), radius: 5, x: 0, y: 2)
    }
}

// MARK: - Users View

struct UsersView: View {
    @ObservedObject var viewModel: UsersViewModel
    @State private var searchText = ""
    
    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                // Filter Bar
                HStack(spacing: 12) {
                    Picker("Status", selection: Binding(
                        get: { viewModel.statusFilter },
                        set: { viewModel.filterByStatus($0) }
                    )) {
                        Text("All Status").tag(UserStatus?.none)
                        Text("Active").tag(UserStatus?.active)
                        Text("Pending").tag(UserStatus?.pending)
                        Text("Suspended").tag(UserStatus?.suspended)
                        Text("Banned").tag(UserStatus?.banned)
                    }
                    .pickerStyle(MenuPickerStyle())
                    
                    Picker("KYC", selection: Binding(
                        get: { viewModel.kycFilter },
                        set: { viewModel.filterByKYC($0) }
                    )) {
                        Text("All KYC").tag(KYCStatus?.none)
                        Text("None").tag(KYCStatus?.none)
                        Text("Pending").tag(KYCStatus?.pending)
                        Text("Level 1").tag(KYCStatus?.level1)
                        Text("Level 2").tag(KYCStatus?.level2)
                        Text("Level 3").tag(KYCStatus?.level3)
                    }
                    .pickerStyle(MenuPickerStyle())
                }
                .padding()
                .background(Color(.secondarySystemBackground))
                
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
                        ForEach(viewModel.users) { user in
                            UserRow(user: user) {
                                // Action menu
                            }
                        }
                    }
                    .listStyle(PlainListStyle())
                }
            }
            .navigationTitle("Users")
            .searchable(text: $searchText, prompt: "Search users")
            .onChange(of: searchText) { _, newValue in
                viewModel.search(newValue)
            }
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.refresh() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .alert("Error", isPresented: $viewModel.isError) {
                Button("OK") { viewModel.clearError() }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
    }
}

struct UserRow: View {
    let user: PlatformUser
    let action: () -> Void
    
    var body: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text(user.email)
                    .font(.headline)
                Text("ID: \(user.id)")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            VStack(alignment: .trailing, spacing: 4) {
                StatusBadge(status: user.status)
                KYCBadge(status: user.kycStatus, level: user.kycLevel)
            }
        }
        .padding(.vertical, 8)
    }
}

struct StatusBadge: View {
    let status: UserStatus
    
    var body: some View {
        Text(status.displayName)
            .font(.caption)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(status.color.opacity(0.2))
            .foregroundColor(status.color)
            .cornerRadius(4)
    }
    
    var color: Color {
        switch self {
        case .active: return .green
        case .pending: return .orange
        case .suspended: return .yellow
        case .banned: return .red
        }
    }
}

struct KYCBadge: View {
    let status: KYCStatus
    let level: Int
    
    var body: some View {
        Text("KYC L\(level)")
            .font(.caption)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(status.color.opacity(0.2))
            .foregroundColor(status.color)
            .cornerRadius(4)
    }
}

extension KYCStatus {
    var color: Color {
        switch self {
        case .level3: return .green
        case .level2: return .blue
        case .level1: return .orange
        case .pending: return .yellow
        case .none, .rejected: return .gray
        }
    }
}

// MARK: - Transactions View

struct TransactionsView: View {
    @ObservedObject var viewModel: TransactionsViewModel
    
    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                // Filter Bar
                HStack(spacing: 12) {
                    Picker("Status", selection: Binding(
                        get: { viewModel.statusFilter },
                        set: { viewModel.filterByStatus($0) }
                    )) {
                        Text("All").tag(TransactionStatus?.none)
                        Text("Pending").tag(TransactionStatus?.pending)
                        Text("Confirmed").tag(TransactionStatus?.confirmed)
                        Text("Failed").tag(TransactionStatus?.failed)
                    }
                    .pickerStyle(MenuPickerStyle())
                    
                    Toggle("Flagged Only", isOn: Binding(
                        get: { viewModel.flaggedOnly },
                        set: { _ in viewModel.toggleFlaggedOnly() }
                    ))
                }
                .padding()
                .background(Color(.secondarySystemBackground))
                
                if viewModel.isLoading {
                    Spacer()
                    ProgressView()
                    Spacer()
                } else if viewModel.transactions.isEmpty {
                    Spacer()
                    Text("No transactions found")
                        .foregroundColor(.secondary)
                    Spacer()
                } else {
                    List {
                        ForEach(viewModel.transactions) { tx in
                            TransactionRow(transaction: tx) {
                                if tx.flagged {
                                    viewModel.unflagTransaction(tx)
                                } else {
                                    viewModel.flagTransaction(tx, reason: "Flagged by admin")
                                }
                            }
                        }
                    }
                    .listStyle(PlainListStyle())
                }
            }
            .navigationTitle("Transactions")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.refresh() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .alert("Error", isPresented: $viewModel.isError) {
                Button("OK") { viewModel.clearError() }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
    }
}

struct TransactionRow: View {
    let transaction: Transaction
    let onFlag: () -> Void
    
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(transaction.type.displayName)
                    .font(.headline)
                Spacer()
                TransactionStatusBadge(status: transaction.status)
            }
            
            HStack {
                Text("From: \(transaction.fromAddress.prefix(8))...")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Spacer()
                Text("To: \(transaction.toAddress.prefix(8))...")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            HStack {
                Text("\(transaction.amount) \(transaction.token)")
                    .font(.subheadline)
                    .fontWeight(.semibold)
                Spacer()
                Text(transaction.chain)
                    .font(.caption)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.blue.opacity(0.2))
                    .foregroundColor(.blue)
                    .cornerRadius(4)
            }
            
            if transaction.flagged {
                HStack {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.red)
                    Text("Flagged")
                        .foregroundColor(.red)
                        .font(.caption)
                }
            }
        }
        .padding(.vertical, 8)
        .swipeActions(edge: .trailing) {
            Button(action: onFlag) {
                Label(transaction.flagged ? "Unflag" : "Flag", systemImage: transaction.flagged ? "checkmark.circle" : "exclamationmark.triangle")
            }
            .tint(transaction.flagged ? .green : .red)
        }
    }
}

struct TransactionStatusBadge: View {
    let status: TransactionStatus
    
    var body: some View {
        Text(status.displayName)
            .font(.caption)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(status.color.opacity(0.2))
            .foregroundColor(status.color)
            .cornerRadius(4)
    }
}

extension TransactionStatus {
    var color: Color {
        switch self {
        case .confirmed: return .green
        case .pending: return .orange
        case .failed: return .red
        }
    }
}

// MARK: - KYC View

struct KYCView: View {
    @ObservedObject var viewModel: KYCViewModel
    
    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                // Filter Bar
                HStack(spacing: 12) {
                    Picker("Status", selection: Binding(
                        get: { viewModel.statusFilter },
                        set: { viewModel.filterByStatus($0) }
                    )) {
                        Text("All").tag(KYCApplicationStatus?.none)
                        Text("Pending").tag(KYCApplicationStatus?.pending)
                        Text("Approved").tag(KYCApplicationStatus?.approved)
                        Text("Rejected").tag(KYCApplicationStatus?.rejected)
                    }
                    .pickerStyle(MenuPickerStyle())
                }
                .padding()
                .background(Color(.secondarySystemBackground))
                
                if viewModel.isLoading {
                    Spacer()
                    ProgressView()
                    Spacer()
                } else if viewModel.applications.isEmpty {
                    Spacer()
                    Text("No KYC applications")
                        .foregroundColor(.secondary)
                    Spacer()
                } else {
                    List {
                        ForEach(viewModel.applications) { app in
                            KYCApplicationRow(application: app) {
                                viewModel.approveKYC(app)
                            } onReject: { reason in
                                viewModel.rejectKYC(app, reason: reason)
                            }
                        }
                    }
                    .listStyle(PlainListStyle())
                }
            }
            .navigationTitle("KYC Verification")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.refresh() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .alert("Error", isPresented: $viewModel.isError) {
                Button("OK") { viewModel.clearError() }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
    }
}

struct KYCApplicationRow: View {
    let application: KYCApplication
    let onApprove: () -> Void
    let onReject: (String) -> Void
    
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(application.userEmail)
                    .font(.headline)
                Spacer()
                Text("Level \(application.level)")
                    .font(.caption)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.blue.opacity(0.2))
                    .foregroundColor(.blue)
                    .cornerRadius(4)
            }
            
            HStack {
                Text("Submitted: \(formatDate(application.submittedAt))")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Spacer()
                KYCStatusBadge(status: application.status)
            }
        }
        .padding(.vertical, 8)
        .swipeActions(edge: .trailing) {
            if application.status == .pending {
                Button(action: onApprove) {
                    Label("Approve", systemImage: "checkmark.circle")
                }
                .tint(.green)
                
                Button(action: { onReject("Rejected by admin") }) {
                    Label("Reject", systemImage: "xmark.circle")
                }
                .tint(.red)
            }
        }
    }
    
    func formatDate(_ dateString: String) -> String {
        // Simple date formatting
        return dateString.prefix(10).description
    }
}

struct KYCStatusBadge: View {
    let status: KYCApplicationStatus
    
    var body: some View {
        Text(status.displayName)
            .font(.caption)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(status.color.opacity(0.2))
            .foregroundColor(status.color)
            .cornerRadius(4)
    }
}

extension KYCApplicationStatus {
    var color: Color {
        switch self {
        case .approved: return .green
        case .pending: return .orange
        case .rejected: return .red
        }
    }
}

// MARK: - Tokens View

struct TokensView: View {
    @ObservedObject var viewModel: TokensViewModel
    
    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                if viewModel.isLoading {
                    Spacer()
                    ProgressView()
                    Spacer()
                } else if viewModel.tokens.isEmpty {
                    Spacer()
                    Text("No tokens found")
                        .foregroundColor(.secondary)
                    Spacer()
                } else {
                    List {
                        ForEach(viewModel.tokens) { token in
                            TokenRow(token: token) {
                                if token.isActive {
                                    viewModel.deactivateToken(token)
                                } else {
                                    viewModel.activateToken(token)
                                }
                            }
                        }
                    }
                    .listStyle(PlainListStyle())
                }
            }
            .navigationTitle("Tokens")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.refresh() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .alert("Error", isPresented: $viewModel.isError) {
                Button("OK") { viewModel.clearError() }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
    }
}

struct TokenRow: View {
    let token: Token
    let onToggle: () -> Void
    
    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(token.name)
                        .font(.headline)
                    if token.isVerified {
                        Image(systemName: "checkmark.seal.fill")
                            .foregroundColor(.blue)
                            .font(.caption)
                    }
                }
                Text(token.symbol)
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            VStack(alignment: .trailing, spacing: 4) {
                if let price = token.price {
                    Text("$\(price)")
                        .font(.subheadline)
                        .fontWeight(.semibold)
                }
                
                HStack(spacing: 8) {
                    if token.isActive {
                        Text("Active")
                            .font(.caption)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(Color.green.opacity(0.2))
                            .foregroundColor(.green)
                            .cornerRadius(4)
                    }
                    
                    Button(action: onToggle) {
                        Image(systemName: token.isActive ? "pause.circle" : "play.circle")
                    }
                }
            }
        }
        .padding(.vertical, 8)
    }
}

// MARK: - Withdrawals View

struct WithdrawalsView: View {
    @ObservedObject var viewModel: WithdrawalsViewModel
    
    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                if viewModel.isLoading {
                    Spacer()
                    ProgressView()
                    Spacer()
                } else if viewModel.withdrawals.isEmpty {
                    Spacer()
                    Text("No withdrawals found")
                        .foregroundColor(.secondary)
                    Spacer()
                } else {
                    List {
                        ForEach(viewModel.withdrawals) { withdrawal in
                            WithdrawalRow(withdrawal: withdrawal) {
                                viewModel.approveWithdrawal(withdrawal)
                            } onReject: { reason in
                                viewModel.rejectWithdrawal(withdrawal, reason: reason)
                            }
                        }
                    }
                    .listStyle(PlainListStyle())
                }
            }
            .navigationTitle("Withdrawals")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.refresh() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .alert("Error", isPresented: $viewModel.isError) {
                Button("OK") { viewModel.clearError() }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
    }
}

struct WithdrawalRow: View {
    let withdrawal: WithdrawalRequest
    let onApprove: () -> Void
    let onReject: (String) -> Void
    
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(withdrawal.userEmail)
                    .font(.headline)
                Spacer()
                WithdrawalStatusBadge(status: withdrawal.status)
            }
            
            HStack {
                Text("\(withdrawal.amount) \(withdrawal.token)")
                    .font(.subheadline)
                    .fontWeight(.semibold)
                Spacer()
                Text(withdrawal.chain)
                    .font(.caption)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.blue.opacity(0.2))
                    .foregroundColor(.blue)
                    .cornerRadius(4)
            }
            
            Text("To: \(withdrawal.toAddress)")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 8)
        .swipeActions(edge: .trailing) {
            if withdrawal.status == .pending {
                Button(action: onApprove) {
                    Label("Approve", systemImage: "checkmark.circle")
                }
                .tint(.green)
                
                Button(action: { onReject("Rejected by admin") }) {
                    Label("Reject", systemImage: "xmark.circle")
                }
                .tint(.red)
            }
        }
    }
}

struct WithdrawalStatusBadge: View {
    let status: WithdrawalStatus
    
    var body: some View {
        Text(status.displayName)
            .font(.caption)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(status.color.opacity(0.2))
            .foregroundColor(status.color)
            .cornerRadius(4)
    }
}

extension WithdrawalStatus {
    var color: Color {
        switch self {
        case .completed: return .green
        case .approved, .processing: return .blue
        case .pending: return .orange
        case .rejected, .failed: return .red
        }
    }
}

// MARK: - System View

struct SystemView: View {
    @ObservedObject var viewModel: SystemViewModel
    
    var body: some View {
        NavigationView {
            ScrollView {
                VStack(spacing: 20) {
                    if let health = viewModel.health {
                        HealthCard(health: health)
                    }
                    
                    if !viewModel.services.isEmpty {
                        StatusSection(title: "Services", items: viewModel.services)
                    }
                    
                    if !viewModel.databases.isEmpty {
                        StatusSection(title: "Databases", items: viewModel.databases)
                    }
                    
                    if !viewModel.networks.isEmpty {
                        StatusSection(title: "Networks", items: viewModel.networks)
                    }
                }
                .padding()
            }
            .navigationTitle("System Status")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { viewModel.refresh() }) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .alert("Error", isPresented: $viewModel.isError) {
                Button("OK") { viewModel.clearError() }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
    }
}

struct HealthCard: View {
    let health: SystemHealth
    
    var body: some View {
        VStack(spacing: 16) {
            HStack {
                VStack(alignment: .leading) {
                    Text("Status")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(health.status)
                        .font(.headline)
                        .foregroundColor(health.status == "healthy" ? .green : .red)
                }
                Spacer()
                VStack(alignment: .trailing) {
                    Text("Uptime")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(health.uptime)
                        .font(.headline)
                }
            }
            
            HStack {
                VStack(alignment: .leading) {
                    Text("CPU")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(health.cpuUsage)
                        .font(.subheadline)
                }
                Spacer()
                VStack(alignment: .center) {
                    Text("Memory")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(health.memoryUsage)
                        .font(.subheadline)
                }
                Spacer()
                VStack(alignment: .trailing) {
                    Text("Disk")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(health.diskUsage)
                        .font(.subheadline)
                }
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(12)
        .shadow(color: Color.black.opacity(0.1), radius: 5, x: 0, y: 2)
    }
}

struct StatusSection: View {
    let title: String
    let items: [SystemStatus]
    
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title)
                .font(.headline)
                .padding(.horizontal)
            
            VStack(spacing: 8) {
                ForEach(items, id: \.serviceName) { item in
                    StatusItem(status: item)
                }
            }
            .padding()
            .background(Color(.systemBackground))
            .cornerRadius(12)
            .shadow(color: Color.black.opacity(0.1), radius: 5, x: 0, y: 2)
        }
    }
}

struct StatusItem: View {
    let status: SystemStatus
    
    var body: some View {
        HStack {
            Circle()
                .fill(status.isHealthy ? Color.green : Color.red)
                .frame(width: 8, height: 8)
            
            Text(status.serviceName)
                .font(.subheadline)
            
            Spacer()
            
            Text(status.uptime)
                .font(.caption)
                .foregroundColor(.secondary)
            
            Text(status.latency)
                .font(.caption)
                .foregroundColor(.secondary)
                .frame(width: 60, alignment: .trailing)
        }
    }
}

// MARK: - Preview

struct AdminMainView_Previews: PreviewProvider {
    static var previews: some View {
        AdminMainView()
    }
}
