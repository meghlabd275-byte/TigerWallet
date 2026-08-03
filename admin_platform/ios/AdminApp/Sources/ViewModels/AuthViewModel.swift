import Foundation
import SwiftUI

class AuthViewModel: ObservableObject {
    @Published var isLoggedIn = false
    @Published var isLoading = false
    @Published var error: String?
    @Published var currentAdmin: Admin?
    
    private let api = ApiService.shared
    
    init() {
        isLoggedIn = UserDefaults.standard.string(forKey: "auth_token") != nil
    }
    
    func login(email: String, password: String) {
        isLoading = true
        error = nil
        
        Task {
            do {
                let response = try await api.login(email: email, password: password)
                await MainActor.run {
                    self.currentAdmin = response.admin
                    self.isLoggedIn = true
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }
    
    func logout() {
        isLoading = true
        
        Task {
            do {
                try await api.logout()
                await MainActor.run {
                    self.currentAdmin = nil
                    self.isLoggedIn = false
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }
}

class DashboardViewModel: ObservableObject {
    @Published var stats: DashboardStats?
    @Published var isLoading = false
    @Published var error: String?
    
    private let api = ApiService.shared
    
    init() {
        loadDashboard()
    }
    
    func loadDashboard() {
        isLoading = true
        error = nil
        
        Task {
            do {
                let dashboardStats = try await api.getDashboard()
                await MainActor.run {
                    self.stats = dashboardStats
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }
}

class UsersViewModel: ObservableObject {
    @Published var users: [User] = []
    @Published var isLoading = false
    @Published var error: String?
    @Published var currentPage = 1
    @Published var totalPages = 1
    
    private let api = ApiService.shared
    private var statusFilter: String?
    private var searchQuery: String?
    
    init() {
        loadUsers()
    }
    
    func loadUsers(status: String? = nil, search: String? = nil) {
        self.statusFilter = status
        self.searchQuery = search
        self.currentPage = 1
        fetchUsers()
    }
    
    func loadNextPage() {
        guard currentPage < totalPages else { return }
        currentPage += 1
        fetchUsers()
    }
    
    private func fetchUsers() {
        isLoading = true
        error = nil
        
        Task {
            do {
                let response = try await api.getUsers(
                    page: currentPage,
                    limit: 20,
                    status: statusFilter,
                    search: searchQuery
                )
                await MainActor.run {
                    if self.currentPage == 1 {
                        self.users = response.data
                    } else {
                        self.users.append(contentsOf: response.data)
                    }
                    self.totalPages = response.meta.totalPages
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }
    
    func suspendUser(id: String, reason: String) {
        Task {
            do {
                try await api.suspendUser(id: id, reason: reason)
                await MainActor.run {
                    self.loadUsers(status: self.statusFilter, search: self.searchQuery)
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                }
            }
        }
    }
    
    func banUser(id: String, reason: String) {
        Task {
            do {
                try await api.banUser(id: id, reason: reason)
                await MainActor.run {
                    self.loadUsers(status: self.statusFilter, search: self.searchQuery)
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                }
            }
        }
    }
}

class KYCHandler: ObservableObject {
    @Published var submissions: [KYCSubmission] = []
    @Published var isLoading = false
    @Published var error: String?
    @Published var currentPage = 1
    @Published var totalPages = 1
    
    private let api = ApiService.shared
    
    init() {
        loadSubmissions()
    }
    
    func loadSubmissions(status: String? = nil, level: Int? = nil) {
        currentPage = 1
        fetchSubmissions(status: status, level: level)
    }
    
    private func fetchSubmissions(status: String?, level: Int?) {
        isLoading = true
        
        Task {
            do {
                let response = try await api.getKYCSubmissions(
                    page: currentPage,
                    limit: 20,
                    status: status,
                    level: level
                )
                await MainActor.run {
                    if self.currentPage == 1 {
                        self.submissions = response.data
                    } else {
                        self.submissions.append(contentsOf: response.data)
                    }
                    self.totalPages = response.meta.totalPages
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }
    
    func approve(id: String, notes: String?) {
        Task {
            do {
                try await api.approveKYC(id: id, notes: notes)
                await MainActor.run {
                    self.loadSubmissions()
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                }
            }
        }
    }
    
    func reject(id: String, reason: String) {
        Task {
            do {
                try await api.rejectKYC(id: id, reason: reason)
                await MainActor.run {
                    self.loadSubmissions()
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                }
            }
        }
    }
}

class TokensViewModel: ObservableObject {
    @Published var tokens: [Token] = []
    @Published var isLoading = false
    @Published var error: String?
    
    private let api = ApiService.shared
    
    init() {
        loadTokens()
    }
    
    func loadTokens() {
        isLoading = true
        
        Task {
            do {
                let response = try await api.getTokens()
                await MainActor.run {
                    self.tokens = response.data
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }
    
    func verifyToken(id: String) {
        Task {
            do {
                try await api.verifyToken(id: id)
                await MainActor.run {
                    self.loadTokens()
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                }
            }
        }
    }
    
    func deleteToken(id: String) {
        Task {
            do {
                try await api.deleteToken(id: id)
                await MainActor.run {
                    self.loadTokens()
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                }
            }
        }
    }
}
