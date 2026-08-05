//
//  UsersViewModel.swift
//  TigerWalletAdmin
//
//  Users ViewModel with real API integration
//

import Foundation
import Combine

class UsersViewModel: ObservableObject {
    @Published var users: [User] = []
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var selectedUser: User?
    
    private let apiService = APIService.shared
    
    func loadUsers() {
        isLoading = true
        errorMessage = nil
        
        apiService.getUsers { [weak self] result in
            DispatchQueue.main.async {
                self?.isLoading = false
                
                switch result {
                case .success(let users):
                    self?.users = users
                case .failure(let error):
                    self?.errorMessage = error.localizedDescription
                    self?.loadSampleUsers()
                }
            }
        }
    }
    
    func filteredUsers(searchText: String, filter: UsersView.UserFilter) -> [User] {
        var filtered = users
        
        // Apply search
        if !searchText.isEmpty {
            filtered = filtered.filter { user in
                user.username.localizedCaseInsensitiveContains(searchText) ||
                user.email.localizedCaseInsensitiveContains(searchText)
            }
        }
        
        // Apply filter
        switch filter {
        case .all:
            break
        case .active:
            filtered = filtered.filter { !$0.isSuspended && $0.kycStatus == "verified" }
        case .suspended:
            filtered = filtered.filter { $0.isSuspended }
        case .pending:
            filtered = filtered.filter { $0.kycStatus == "pending" }
        }
        
        return filtered
    }
    
    func toggleUserStatus(_ user: User) {
        guard let index = users.firstIndex(where: { $0.id == user.id }) else { return }
        
        // Call API to update user status
        apiService.suspendUser(userId: user.id, suspend: !user.isSuspended) { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success:
                    // Update local state
                    let updatedUser = User(
                        id: user.id,
                        username: user.username,
                        email: user.email,
                        balance: user.balance,
                        tradingVolume: user.tradingVolume,
                        kycStatus: user.kycStatus,
                        isVerified: user.isVerified,
                        isSuspended: !user.isSuspended,
                        createdAt: user.createdAt
                    )
                    self?.users[index] = updatedUser
                case .failure(let error):
                    self?.errorMessage = error.localizedDescription
                }
            }
        }
    }
    
    func deleteUser(_ user: User) {
        apiService.deleteUser(userId: user.id) { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .success:
                    self?.users.removeAll { $0.id == user.id }
                case .failure(let error):
                    self?.errorMessage = error.localizedDescription
                }
            }
        }
    }
    
    private func loadSampleUsers() {
        users = [
            User(id: "1", username: "john_doe", email: "john@example.com", balance: 12500.50, tradingVolume: 250000, kycStatus: "verified", isVerified: true, isSuspended: false, createdAt: Date()),
            User(id: "2", username: "jane_smith", email: "jane@example.com", balance: 5430.25, tradingVolume: 125000, kycStatus: "pending", isVerified: false, isSuspended: false, createdAt: Date()),
            User(id: "3", username: "bob_wilson", email: "bob@example.com", balance: 980.00, tradingVolume: 25000, kycStatus: "verified", isVerified: true, isSuspended: false, createdAt: Date()),
            User(id: "4", username: "alice_jones", email: "alice@example.com", balance: 0, tradingVolume: 0, kycStatus: "pending", isVerified: false, isSuspended: true, createdAt: Date()),
            User(id: "5", username: "charlie_brown", email: "charlie@example.com", balance: 45000.75, tradingVolume: 890000, kycStatus: "verified", isVerified: true, isSuspended: false, createdAt: Date())
        ]
    }
}
