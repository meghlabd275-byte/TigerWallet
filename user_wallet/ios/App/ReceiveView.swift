import SwiftUI
import CoreImage.CIFilterBuiltins
import UIKit

// Receive tab. Shows the selected wallet's real address with Copy
// (UIPasteboard) + a share note, and a REAL QR code generated via CoreImage's
// CIQRCodeGenerator (no fake QR image, no bundled asset). Mirrors the web
// /receive page 1:1. Fetches wallets via UserWalletApiService.shared.getWallets.
struct ReceiveView: View {
    @State private var wallets: [WalletRecord] = []
    @State private var selectedWalletId: String?
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var copied = false

    private var selectedWallet: WalletRecord? {
        wallets.first { $0.id == selectedWalletId }
    }

    var body: some View {
        NavigationView {
            Group {
                if isLoading {
                    ProgressView("Loading wallets...")
                } else if let errorMessage = errorMessage {
                    VStack(spacing: 12) {
                        Text(errorMessage)
                            .foregroundColor(.red)
                            .font(.subheadline)
                            .multilineTextAlignment(.center)
                        Button("Retry", action: loadWallets)
                            .buttonStyle(.bordered)
                    }
                    .padding()
                } else if wallets.isEmpty {
                    VStack(spacing: 12) {
                        Text("No wallets yet")
                            .foregroundColor(.secondary)
                        Text("Create or import a wallet to see its receive address.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                            .multilineTextAlignment(.center)
                    }
                    .padding()
                } else if let wallet = selectedWallet {
                    receiveContent(for: wallet)
                }
            }
            .navigationTitle("Receive")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: loadWallets) {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .onAppear { loadWallets() }
        }
    }

    @ViewBuilder
    private func receiveContent(for wallet: WalletRecord) -> some View {
        ScrollView {
            VStack(spacing: 20) {
                if wallets.count > 1 {
                    Picker("Wallet", selection: $selectedWalletId) {
                        ForEach(wallets) { w in
                            Text("\(w.label) - \(w.address.prefix(8))...")
                                .tag(Optional(w.id))
                        }
                    }
                    .pickerStyle(.menu)
                    .padding(.horizontal)
                }

                if let qr = generateQRImage(from: wallet.address) {
                    Image(uiImage: qr)
                        .interpolation(.none)
                        .resizable()
                        .scaledToFit()
                        .frame(width: 240, height: 240)
                        .padding(12)
                        .background(Color.white)
                        .cornerRadius(16)
                        .accessibilityLabel("QR code for wallet address")
                }

                VStack(spacing: 6) {
                    Text(wallet.label)
                        .font(.headline)
                    Text("Chain #\(wallet.chain_id)")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                Text(wallet.address)
                    .font(.system(.body, design: .monospaced))
                    .multilineTextAlignment(.center)
                    .padding(12)
                    .background(Color(.systemGray6))
                    .cornerRadius(12)
                    .padding(.horizontal)
                    .textSelection(.enabled)

                HStack(spacing: 16) {
                    Button {
                        UIPasteboard.general.string = wallet.address
                        copied = true
                    } label: {
                        Label("Copy", systemImage: "doc.on.doc")
                            .frame(maxWidth: .infinity, minHeight: 44)
                    }
                    .buttonStyle(.bordered)
                    .tint(.orange)

                    Button {
                        let av = UIActivityViewController(activityItems: [wallet.address], applicationActivities: nil)
                        UIApplication.shared.connectedScenes
                            .compactMap { $0 as? UIWindowScene }
                            .first?.windows
                            .first?.rootViewController?
                            .present(av, animated: true)
                    } label: {
                        Label("Share", systemImage: "square.and.arrow.up")
                            .frame(maxWidth: .infinity, minHeight: 44)
                    }
                    .buttonStyle(.bordered)
                }
                .padding(.horizontal)

                if copied {
                    Text("Address copied!")
                        .font(.caption)
                        .foregroundColor(.green)
                }

                Text("Send only matching-chain assets to this address. Funds sent on other networks may be lost.")
                    .font(.footnote)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal)
            }
            .padding(.vertical)
        }
    }

    // Real QR via CoreImage CIQRCodeGenerator. Returns nil only if CIContext
    // rendering fails; never a placeholder image.
    private func generateQRImage(from string: String) -> UIImage? {
        let context = CIContext()
        let filter = CIFilter.qrCodeGenerator()
        filter.message = Data(string.utf8)
        filter.correctionLevel = "M"
        guard let output = filter.outputImage else { return nil }
        // Scale up — default output is tiny (~21px). 10x keeps it crisp.
        let scaled = output.transformed(by: CGAffineTransform(scaleX: 10, y: 10))
        guard let cg = context.createCGImage(scaled, from: scaled.extent) else { return nil }
        return UIImage(cgImage: cg)
    }

    private func loadWallets() {
        isLoading = true
        errorMessage = nil
        Task {
            do {
                let result = try await UserWalletApiService.shared.getWallets()
                await MainActor.run {
                    self.wallets = result
                    if self.selectedWalletId == nil {
                        self.selectedWalletId = result.first?.id
                    }
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
}
