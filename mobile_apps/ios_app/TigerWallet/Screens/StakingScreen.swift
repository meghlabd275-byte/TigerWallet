import SwiftUI

// Staking Screen - Stake and earn
struct StakingScreen: View {
    @State private var selectedTab: String = "Stake"
    @State private var stakeAmount: String = ""
    @State private var selectedPool: String = "ETH 2.0"
    @State private var isStaking: Bool = false
    
    let tabs = ["Stake", "Earn", "Pools"]
    
    let stakingPools = [
        StakingPool(name: "ETH 2.0", apy: "4.2%", staked: "1.5 ETH", reward: "0.063 ETH"),
        StakingPool(name: "BNB", apy: "3.8%", staked: "0 BNB", reward: "0 BNB"),
        StakingPool(name: "SOL", apy: "6.5%", staked: "0 SOL", reward: "0 SOL"),
        StakingPool(name: "MATIC", apy: "5.2%", staked: "0 MATIC", reward: "0 MATIC")
    ]
    
    var body: some View {
        NavigationView {
            VStack(spacing: 0) {
                // Tab Selector
                HStack(spacing: 0) {
                    ForEach(tabs, id: \.self) { tab in
                        Button(action: { selectedTab = tab }) {
                            VStack {
                                Text(tab)
                                    .fontWeight(selectedTab == tab ? .bold : .regular)
                                Rectangle()
                                    .fill(selectedTab == tab ? Color.orange : Color.clear)
                                    .frame(height: 2)
                            }
                        }
                        .frame(maxWidth: .infinity)
                    }
                }
                .padding()
                
                ScrollView {
                    VStack(spacing: 20) {
                        if selectedTab == "Stake" {
                            stakeView
                        } else if selectedTab == "Earn" {
                            earnView
                        } else {
                            poolsView
                        }
                    }
                    .padding()
                }
            }
            .navigationTitle("Staking")
        }
    }
    
    var stakeView: some View {
        VStack(spacing: 20) {
            // Pool Selector
            VStack(alignment: .leading, spacing: 8) {
                Text("Select Pool")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                
                Menu {
                    ForEach(stakingPools, id: \.name) { pool in
                        Button(action: { selectedPool = pool.name }) {
                            Text("\(pool.name) - \(pool.apy)")
                        }
                    }
                } label: {
                    HStack {
                        Text(selectedPool)
                            .fontWeight(.semibold)
                        Spacer()
                        if let pool = stakingPools.first(where: { $0.name == selectedPool }) {
                            Text("APY: \(pool.apy)")
                                .font(.caption)
                                .foregroundColor(.green)
                        }
                        Image(systemName: "chevron.down")
                    }
                    .padding()
                    .background(Color.gray.opacity(0.1))
                    .cornerRadius(8)
                }
            }
            
            // Amount
            VStack(alignment: .leading, spacing: 8) {
                Text("Amount")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                
                HStack {
                    TextField("0.0", text: $stakeAmount)
                        .keyboardType(.decimalPad)
                        .font(.title)
                    
                    Button(action: { stakeAmount = "1.0" }) {
                        Text("MAX")
                            .foregroundColor(.orange)
                    }
                }
                .padding()
                .background(Color.gray.opacity(0.1))
                .cornerRadius(8)
            }
            
            // Stake Button
            Button(action: stake) {
                HStack {
                    if isStaking {
                        ProgressView()
                            .progressViewStyle(CircularProgressViewStyle(tint: .white))
                    } else {
                        Text("Stake")
                            .fontWeight(.bold)
                    }
                }
                .frame(maxWidth: .infinity)
                .padding()
                .background(Color.orange)
                .foregroundColor(.white)
                .cornerRadius(12)
            }
            .disabled(isStaking)
        }
    }
    
    var earnView: some View {
        VStack(spacing: 16) {
            ForEach(stakingPools, id: \.name) { pool in
                earnCard(pool: pool)
            }
        }
    }
    
    func earnCard(pool: StakingPool) -> some View {
        VStack(spacing: 12) {
            HStack {
                VStack(alignment: .leading) {
                    Text(pool.name)
                        .font(.headline)
                    Text("APY: \(pool.apy)")
                        .font(.caption)
                        .foregroundColor(.green)
                }
                Spacer()
                VStack(alignment: .trailing) {
                    Text("Staked")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(pool.staked)
                        .font(.subheadline)
                }
            }
            
            HStack {
                VStack(alignment: .leading) {
                    Text("Pending Reward")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Text(pool.reward)
                        .font(.subheadline)
                        .foregroundColor(.green)
                }
                Spacer()
                Button(action: {}) {
                    Text("Claim")
                        .font(.caption)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 8)
                        .background(Color.orange)
                        .foregroundColor(.white)
                        .cornerRadius(8)
                }
            }
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(12)
    }
    
    var poolsView: some View {
        VStack(spacing: 16) {
            ForEach(stakingPools, id: \.name) { pool in
                poolCard(pool: pool)
            }
        }
    }
    
    func poolCard(pool: StakingPool) -> some View {
        VStack(spacing: 12) {
            HStack {
                Text(pool.name)
                    .font(.headline)
                Spacer()
                Text(pool.apy)
                    .font(.title2)
                    .fontWeight(.bold)
                    .foregroundColor(.green)
                Text("APY")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            HStack {
                Text("Total Staked: \(pool.staked)")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Spacer()
                Button(action: {}) {
                    Text("Stake")
                        .font(.caption)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 8)
                        .background(Color.orange)
                        .foregroundColor(.white)
                        .cornerRadius(8)
                }
            }
        }
        .padding()
        .background(Color.gray.opacity(0.1))
        .cornerRadius(12)
    }
    
    func stake() {
        guard !stakeAmount.isEmpty else { return }
        isStaking = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
            isStaking = false
            stakeAmount = ""
        }
    }
}

struct StakingPool: Identifiable {
    let id = UUID()
    let name: String
    let apy: String
    let staked: String
    let reward: String
}

#Preview {
    StakingScreen()
}
