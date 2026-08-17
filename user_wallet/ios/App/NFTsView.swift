import SwiftUI

// NFTs tab. Pick a wallet, fetch its real ERC-721 inventory via getNFTs, and
// transfer a selected NFT via transferNFT. Mirrors the web /nfts page.
struct NFTsView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var nfts: [UserWalletApiService.NFT] = []
    @State private var isLoading = false
    @State private var errorMessage: String?

    @State private var transferTarget: UserWalletApiService.NFT?
    @State private var recipient = ""
    @State private var password = ""
    @State private var isTransferring = false

    @State private var showSuccess = false
    @State private var successDetail = ""

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    private var columns: [GridItem] {
        [GridItem(.flexible()), GridItem(.flexible())]
    }

    var body: some View {
        NavigationView {
            Group {
                if wallets.isEmpty {
                    VStack(spacing: 12) {
                        if isLoading {
                            ProgressView("Loading wallets...")
                        } else {
                            Text("No wallets yet")
                                .foregroundColor(.secondary)
                            Text("Create or import a wallet to view its NFTs.")
                                .font(.caption)
                                .foregroundColor(.secondary)
                                .multilineTextAlignment(.center)
                        }
                    }
                    .padding()
                } else {
                    content
                }
            }
            .navigationTitle("NFTs")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: loadNFTs) { Image(systemName: "arrow.clockwise") }
                        .disabled(selectedWallet == nil)
                }
            }
            .onAppear { loadWallets() }
            .sheet(item: $transferTarget) { nft in
                transferSheet(for: nft)
            }
            .alert(isPresented: $showSuccess) {
                Alert(
                    title: Text("\u{2713} Transaction submitted to the blockchain network"),
                    message: Text(successDetail),
                    dismissButton: .default(Text("OK"))
                )
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        VStack(spacing: 0) {
            Picker("Wallet", selection: $selectedWalletId) {
                ForEach(wallets) { wallet in
                    Text("\(wallet.label) - \(wallet.address.prefix(8))...")
                        .tag(Optional(wallet.id))
                }
            }
            .pickerStyle(.menu)
            .padding()

            if isLoading {
                ProgressView("Loading NFTs...").padding()
            } else if let errorMessage = errorMessage {
                VStack(spacing: 8) {
                    Text(errorMessage).foregroundColor(.red).font(.subheadline)
                        .multilineTextAlignment(.center)
                    Button("Retry", action: loadNFTs).buttonStyle(.bordered)
                }.padding()
            } else if nfts.isEmpty {
                Text("No NFTs in this wallet.")
                    .foregroundColor(.secondary).padding()
            } else {
                ScrollView {
                    LazyVGrid(columns: columns, spacing: 16) {
                        ForEach(nfts) { nft in
                            NFTCard(nft: nft) { transferTarget = nft }
                        }
                    }
                    .padding()
                }
            }
        }
    }

    @ViewBuilder
    private func transferSheet(for nft: UserWalletApiService.NFT) -> some View {
        NavigationView {
            Form {
                Section("NFT") {
                    Text(nft.name.isEmpty ? "(unnamed)" : nft.name)
                    Text("Symbol: \(nft.symbol)").font(.caption).foregroundColor(.secondary)
                    Text("Token #\(nft.token_id)").font(.caption).foregroundColor(.secondary)
                    Text("Contract: \(nft.contract_address.prefix(14))...")
                        .font(.caption.monospaced()).foregroundColor(.secondary)
                }
                Section("Transfer") {
                    TextField("Recipient address (0x...)", text: $recipient)
                        .autocapitalization(.none).disableAutocorrection(true)
                    SecureField("Wallet password", text: $password)
                }
                if let errorMessage = errorMessage {
                    Section { Text(errorMessage).foregroundColor(.red).font(.subheadline) }
                }
                Section {
                    Button(action: { transfer(nft: nft) }) {
                        HStack {
                            Text("Transfer NFT")
                            Spacer()
                            if isTransferring { ProgressView().tint(.orange) }
                        }
                    }
                    .disabled(isTransferring || recipient.trimmingCharacters(in: .whitespaces).isEmpty
                              || password.count < 8)
                }
            }
            .navigationTitle("Transfer NFT")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        transferTarget = nil
                        recipient = ""
                        password = ""
                        errorMessage = nil
                    }
                }
            }
        }
    }

    private func loadWallets() {
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    if self.selectedWalletId == nil { self.selectedWalletId = result.first?.id }
                    self.loadNFTs()
                }
            } catch {
                await MainActor.run { self.errorMessage = error.localizedDescription }
            }
        }
    }

    private func loadNFTs() {
        guard let wallet = selectedWallet else { return }
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let result = try await UserWalletApiService.shared.getNFTs(
                    address: wallet.address, chainId: wallet.chain_id)
                await MainActor.run {
                    self.nfts = result
                    self.isLoading = false
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isLoading = false
                }
            }
        }
    }

    private func transfer(nft: UserWalletApiService.NFT) {
        guard let wallet = selectedWallet else { return }
        isTransferring = true
        errorMessage = nil
        let to = recipient.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                let res = try await UserWalletApiService.shared.transferNFT(
                    walletId: wallet.id, password: password, to: to,
                    tokenId: nft.token_id, contractAddress: nft.contract_address,
                    chainId: wallet.chain_id)
                let hash = (res["tx_hash"] as? String) ?? (res["hash"] as? String) ?? ""
                await MainActor.run {
                    self.isTransferring = false
                    self.transferTarget = nil
                    self.successDetail = hash.isEmpty ? "" : "Tx hash: \(hash)"
                    self.showSuccess = true
                    self.recipient = ""
                    self.password = ""
                }
            } catch {
                await MainActor.run {
                    self.errorMessage = error.localizedDescription
                    self.isTransferring = false
                }
            }
        }
    }
}

struct NFTCard: View {
    let nft: UserWalletApiService.NFT
    let onTransfer: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            ZStack {
                Rectangle().fill(Color(.systemGray5))
                if let url = URL(string: nft.image_uri), !nft.image_uri.isEmpty {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .empty:
                            ProgressView()
                        case .success(let image):
                            image.resizable().scaledToFill()
                        case .failure:
                            Image(systemName: "photo")
                                .foregroundColor(.secondary)
                        @unknown default:
                            Image(systemName: "photo")
                                .foregroundColor(.secondary)
                        }
                    }
                } else {
                    Image(systemName: "photo")
                        .font(.largeTitle)
                        .foregroundColor(.secondary)
                }
            }
            .frame(height: 140)
            .clipped()
            .cornerRadius(10)

            VStack(alignment: .leading, spacing: 2) {
                Text(nft.name.isEmpty ? nft.symbol : nft.name)
                    .font(.headline).lineLimit(1)
                Text("ID #\(nft.token_id)")
                    .font(.caption).foregroundColor(.secondary).lineLimit(1)
            }

            Button(action: onTransfer) {
                Label("Transfer", systemImage: "arrow.up.right.square")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.bordered).tint(.orange)
        }
        .padding(8)
        .background(Color(.systemGray6))
        .cornerRadius(12)
    }
}
