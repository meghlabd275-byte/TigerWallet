import SwiftUI
import UIKit

// BackupView — the post-create-wallet backup screen (mirrors the web
// BackupMnemonic component).
//
// Shows the 12/24-word recovery phrase ONCE (the backend returns it only on
// create). Provides:
//   1. Copy-to-clipboard (UIPasteboard.general.string).
//   2. Google Drive backup — REAL Google Drive API v3 upload via GTMAppAuth +
//      GTLRDrive. Requires a Google OAuth client id in Info.plist. If none is
//      set (or the SDK is unlinked), the button is disabled with an honest
//      message (NEVER a fake success).
//   3. Download as an encrypted file (CryptoKit AES-256-GCM +
//      CommonCrypto PBKDF2-SHA256 600k), written to the app Documents dir and
//      shared via UIActivityViewController.
//
// The user MUST toggle "I have backed up my recovery phrase" before the
// Continue button enables; on continue the wallet id is remembered and the app
// proceeds to the dashboard.

struct BackupView: View {
    let mnemonic: String
    let walletId: String
    let walletPassword: String   // derives the offline-backup encryption key
    let onConfirmed: () -> Void

    @State private var revealed = false
    @State private var backedUp = false
    @State private var copied = false
    @State private var dlStatus: DownloadStatus = .idle
    @State private var gdriveStatus: GDriveStatus = .idle
    @State private var gdriveMsg = ""
    @State private var showShareSheet = false
    @State private var shareURL: URL?

    enum DownloadStatus { case idle, done, error }
    enum GDriveStatus { case idle, auth, uploading, done, error }

    private var words: [String] {
        mnemonic.trimmingCharacters(in: .whitespacesAndNewlines).split { $0.isWhitespace }.map(String.init)
    }

    // Mirrors the web 3-column mnemonic grid; LazyVGrid adapts to width.
    private let columns = Array(repeating: GridItem(.flexible(), spacing: 8), count: 3)

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                Text("🔒 Back up your recovery phrase")
                    .font(.title2)
                    .fontWeight(.bold)

                Text("These \(words.count) words control your funds. Write them down or back them up now — they are shown only once and cannot be recovered if lost.")
                    .font(.subheadline)
                    .foregroundColor(.secondary)

                Toggle("Reveal recovery phrase (anyone viewing your screen will see it)",
                       isOn: $revealed)
                    .font(.subheadline)

                if revealed {
                    LazyVGrid(columns: columns, spacing: 8) {
                        ForEach(Array(words.enumerated()), id: \.offset) { idx, word in
                            HStack(spacing: 6) {
                                Text("\(idx + 1)").font(.caption).foregroundColor(.secondary)
                                Text(word).font(.system(.body, design: .monospaced))
                                Spacer(minLength: 0)
                            }
                            .padding(8)
                            .background(Color(.systemGray6))
                            .cornerRadius(8)
                        }
                    }
                    .padding(.top, 4)
                }

                // ---- Backup actions (mirror BackupMnemonic) ----
                VStack(spacing: 12) {
                    Button(action: copy) {
                        Label(copied ? "✓ Copied!" : "Copy to clipboard",
                              systemImage: copied ? "checkmark" : "doc.on.doc")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(Color(.systemGray5))
                            .cornerRadius(8)
                    }
                    .disabled(!revealed)

                    Button(action: googleDriveBackup) {
                        label(for: gdriveStatus,
                              idle: "Back up to Google Drive",
                              busy: gdriveStatus == .auth ? "Authorizing…" : "Uploading…",
                              done: "✓ Backed up to Drive")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(googleButtonColor)
                            .foregroundColor(.white)
                            .cornerRadius(8)
                    }
                    .disabled(!revealed || !GoogleDriveBackup.isConfigured
                              || gdriveStatus == .uploading || gdriveStatus == .auth)

                    if !GoogleDriveBackup.isConfigured {
                        Text("Google Drive not configured — use copy or download.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    if !gdriveMsg.isEmpty {
                        Text(gdriveMsg)
                            .font(.caption)
                            .foregroundColor(gdriveStatus == .done ? .green : .red)
                    }

                    Button(action: download) {
                        Label(dlStatus == .done ? "✓ Downloaded (encrypted)" : "Download encrypted backup",
                              systemImage: dlStatus == .done ? "checkmark" : "square.and.arrow.down")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 10)
                            .background(Color(.systemGray5))
                            .cornerRadius(8)
                    }
                    .disabled(!revealed || dlStatus == .done)
                }

                // ---- Confirmation gate ----
                Toggle("I have backed up my recovery phrase and understand it cannot be recovered",
                       isOn: $backedUp)
                    .font(.subheadline)
                    .padding(.top, 8)

                Button(action: onConfirmed) {
                    Text("Continue to wallet")
                        .fontWeight(.semibold)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(backedUp ? Color.orange : Color.gray.opacity(0.4))
                        .foregroundColor(.white)
                        .cornerRadius(12)
                }
                .disabled(!backedUp)
            }
            .padding()
        }
        .background(shareSheet)
    }

    // Drive button color: gray when not configured, accent when ready.
    private var googleButtonColor: Color {
        GoogleDriveBackup.isConfigured ? Color.blue : Color.gray.opacity(0.5)
    }

    private func label(for status: GDriveStatus, idle: String, busy: String, done: String) -> some View {
        let text: String
        switch status {
        case .auth, .uploading: text = busy
        case .done: text = done
        default: text = idle
        }
        return Text(text)
    }

    // MARK: - Actions

    private func copy() {
        UIPasteboard.general.string = mnemonic
        copied = true
        DispatchQueue.main.asyncAfter(deadline: .now() + 2) { copied = false }
    }

    private func googleDriveBackup() {
        gdriveStatus = .auth
        gdriveMsg = ""
        Task {
            do {
                await MainActor.run { gdriveStatus = .uploading }
                let fileId = try await GoogleDriveBackup.upload(mnemonic: mnemonic, walletId: walletId)
                await MainActor.run {
                    gdriveStatus = .done
                    let short = String(fileId.prefix(12))
                    gdriveMsg = "Backed up to Google Drive (file ID: \(short)…)"
                }
            } catch {
                await MainActor.run {
                    gdriveStatus = .error
                    gdriveMsg = error.localizedDescription
                }
            }
        }
    }

    private func download() {
        do {
            let url = try EncryptedBackup.writeEncryptedBackup(mnemonic: mnemonic,
                                                               walletId: walletId,
                                                               walletPassword: walletPassword)
            shareURL = url
            dlStatus = .done
            showShareSheet = true
        } catch {
            dlStatus = .error
        }
    }

    // Presents the real UIActivityViewController for the encrypted file.
    private var shareSheet: some View {
        EmptyView().sheet(isPresented: $showShareSheet) {
            if let url = shareURL {
                ShareSheet(items: [url])
            }
        }
    }
}
