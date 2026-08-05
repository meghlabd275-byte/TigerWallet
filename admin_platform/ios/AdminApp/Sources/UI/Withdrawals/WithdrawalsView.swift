//
//  WithdrawalsView.swift
//  TigerWalletAdmin
//
//  Complete Withdrawal Management Screen
//

import SwiftUI

struct WithdrawalsView: View {
    @StateObject private var viewModel = WithdrawalsViewModel()
    @EnvironmentObject var themeManager: ThemeManager
    @State private var selectedTab = 0
    
    var body: some View {
        VStack(spacing: 0) {
            // Tab Selector
            Picker("Status", selection: $selectedTab) {
                Text("Pending").tag(0)
                Text("Approved").tag(1)
                Text("Rejected").tag(2)
            }
            .pickerStyle(.segmented)
            .padding()
            
            if viewModel.isLoading {
                Spacer()
                ProgressView()
                Spacer()
            } else {
                let filtered = viewModel.filteredWithdrawals(tab: selectedTab)
                
                if filtered.isEmpty {
                    Spacer()
                    Text("No withdrawals found")
                        .foregroundColor(.secondary)
                    Spacer()
                } else {
                    List {
                        ForEach(filtered) { withdrawal in
                            WithdrawalRow(
                                withdrawal: withdrawal,
                                onApprove: { viewModel.approveWithdrawal(withdrawal) },
                                onReject: { viewModel.rejectWithdrawal(withdrawal) }
                            )
                        }
                    }
                    .listStyle(.plain)
                }
            }
        }
        .background(themeManager.backgroundColor)
        .navigationTitle("Withdrawals")
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button(action: { viewModel.loadWithdrawals() }) {
                    Image(systemName: "arrow.clockwise")
                }
            }
        }
        .onAppear {
            viewModel.loadWithdrawals()
        }
    }
}

struct WithdrawalRow: View {
    let withdrawal: Withdrawal
    let onApprove: () -> Void
    let onReject: () -> Void
    @EnvironmentObject var themeManager: ThemeManager
    @State private var showRejectSheet = false
    
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header
            HStack {
                VStack(alignment: .leading) {
                    Text(withdrawal.userEmail)
                        .font(.headline)
                    
                    Text(withdrawal.userID)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
                
                Spacer()
                
                StatusBadge(status: withdrawal.status)
            }
            
            // Amount Details
            HStack {
                VStack(alignment: .leading) {
                    Text("Amount")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(withdrawal.formattedAmount)
                        .font(.title3)
                        .fontWeight(.bold)
                }
                
                Spacer()
                
                VStack(alignment: .trailing) {
                    Text("Fee")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(withdrawal.formattedFee)
                        .font(.subheadline)
                }
            }
            
            // Destination
            VStack(alignment: .leading) {
                Text("Destination")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Text(withdrawal.toAddress)
                    .font(.caption)
                    .lineLimit(1)
                    .truncationMode(.middle)
            }
            
            // Actions for pending
            if withdrawal.status == "pending" {
                HStack(spacing: 12) {
                    Button(action: onReject) {
                        Text("Reject")
                            .font(.headline)
                            .foregroundColor(.red)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(Color.red.opacity(0.1))
                            .cornerRadius(8)
                    }
                    
                    Button(action: onApprove) {
                        Text("Approve")
                            .font(.headline)
                            .foregroundColor(.green)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(Color.green.opacity(0.1))
                            .cornerRadius(8)
                    }
                }
            }
            
            // Timestamp
            Text(withdrawal.formattedDate)
                .font(.caption2)
                .foregroundColor(.secondary)
        }
        .padding(.vertical, 8)
    }
}

struct Withdrawal: Identifiable {
    let id: String
    let userID: String
    let userEmail: String
    let token: String
    let amount: Double
    let fee: Double
    let chain: String
    let toAddress: String
    let status: String
    let createdAt: Date
    
    var formattedAmount: String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.minimumFractionDigits = 2
        formatter.maximumFractionDigits = 8
        return "\(formatter.string(from: NSNumber(value: amount)) ?? "") \(token)"
    }
    
    var formattedFee: String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        formatter.minimumFractionDigits = 2
        formatter.maximumFractionDigits = 8
        return "\(formatter.string(from: NSNumber(value: fee)) ?? "") \(token)"
    }
    
    var formattedDate: String {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        return formatter.string(from: createdAt)
    }
}

#Preview {
    NavigationView {
        WithdrawalsView()
            .environmentObject(ThemeManager())
    }
}
