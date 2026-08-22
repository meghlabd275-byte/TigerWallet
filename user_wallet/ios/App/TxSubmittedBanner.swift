import SwiftUI
import UIKit

// TxSubmittedBanner — the post-send confirmation shown after a successful
// sendTransaction (mirrors the web component of the same name).
//
// Displays "Transaction submitted to the blockchain network" alongside the real
// tx hash + a link to the block explorer (per chain). Auto-dismisses after 30s;
// the user can also dismiss manually.

struct TxSubmittedBanner: View {
    let txHash: String
    let chainId: Int
    var onDismiss: (() -> Void)? = nil

    @State private var visible = true
    @State private var dismissTask: Task<Void, Never>?

    // Per-chain block explorer tx URL prefix (mirrors the web EXPLORERS map).
    private static let explorers: [Int: String] = [
        1: "https://etherscan.io/tx/",
        56: "https://bscscan.com/tx/",
        137: "https://polygonscan.com/tx/",
        42161: "https://arbiscan.io/tx/",
        10: "https://optimistic.etherscan.io/tx/",
        8453: "https://basescan.org/tx/",
        43114: "https://snowtrace.io/tx/",
    ]

    private var explorerURL: URL? {
        guard let prefix = Self.explorers[chainId] else { return nil }
        return URL(string: prefix + txHash)
    }

    private var shortHash: String {
        guard txHash.count > 18 else { return txHash }
        let head = txHash.prefix(10)
        let tail = txHash.suffix(8)
        return "\(head)…\(tail)"
    }

    var body: some View {
        if visible {
            HStack(alignment: .top, spacing: 12) {
                Text("⛓️")
                    .font(.title3)

                VStack(alignment: .leading, spacing: 4) {
                    Text("Transaction submitted to the blockchain network")
                        .font(.subheadline)
                        .fontWeight(.semibold)

                    if let url = explorerURL {
                        Button(action: { openURL(url) }) {
                            HStack(spacing: 4) {
                                Text(shortHash)
                                    .font(.system(.footnote, design: .monospaced))
                                Image(systemName: "arrow.up.right.square")
                                    .font(.footnote)
                            }
                            .foregroundColor(.accentColor)
                        }
                    } else {
                        Text("\(txHash.prefix(16))…")
                            .font(.system(.footnote, design: .monospaced))
                            .foregroundColor(.secondary)
                    }

                    Text("Awaiting on-chain confirmation. This may take a few moments depending on network congestion.")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }

                Spacer()

                Button(action: dismiss) {
                    Image(systemName: "xmark")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
            }
            .padding(12)
            .background(Color(.systemGreen).opacity(0.12))
            .overlay(
                RoundedRectangle(cornerRadius: 10)
                    .stroke(Color.green.opacity(0.5), lineWidth: 1)
            )
            .cornerRadius(10)
            .onAppear { scheduleAutoDismiss() }
            .onDisappear { dismissTask?.cancel() }
        }
    }

    private func scheduleAutoDismiss() {
        dismissTask?.cancel()
        dismissTask = Task {
            try? await Task.sleep(nanoseconds: 30_000_000_000)
            if Task.isCancelled { return }
            await MainActor.run { dismiss() }
        }
    }

    private func dismiss() {
        withAnimation { visible = false }
        onDismiss?()
    }

    private func openURL(_ url: URL) {
        UIApplication.shared.open(url)
    }
}
