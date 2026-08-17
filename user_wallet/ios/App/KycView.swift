import SwiftUI

// KycView — Identity verification (KYC) screen.
//
// Fetches the caller's real KYC status via
// `UserWalletApiService.shared.getKycStatus(userId)` (the backend resolves the
// user from the Bearer JWT when userId is nil) and, when not yet verified,
// walks the user through the real registration + submission flow
// (`registerKyc` -> `submitKyc`). No mock data: every state displayed comes
// from the backend response. Mirrors the Android `KycFragment` 1:1.
//
// KYC is only required for P2P trading, surfaced as an inline note. The view is
// theme-aware via semantic system colors (Color(.label) / Color(.systemBackground)
// / Color(.systemGray6)) which adapt to `.preferredColorScheme` set on the root.
struct KycView: View {
    @State private var isLoading = true
    @State private var isSubmitting = false
    @State private var status: String = "not_submitted"
    @State private var rawStatus: [String: Any] = [:]
    @State private var error: String?

    @State private var fullName = ""
    @State private var documentType = "passport"
    @State private var documentNumber = ""

    private let documentTypes = ["passport", "national_id", "drivers_license"]

    private var isVerified: Bool {
        status == "verified" || status == "approved"
    }

    private var showForm: Bool {
        // Show the Start-KYC form whenever the user is not verified/pending.
        // i.e. rejected/declined or not_submitted (or any unknown status).
        switch status {
        case "verified", "approved", "pending", "under_review":
            return false
        default:
            return true
        }
    }

    var body: some View {
        NavigationView {
            Form {
                Section {
                    if isLoading {
                        ProgressView("Loading KYC status...")
                    } else {
                        statusRow
                    }
                }

                if isVerified {
                    Section {
                        verifiedBanner
                    }
                }

                if showForm {
                    Section(header: Text("Start KYC")) {
                        TextField("Full name", text: $fullName)
                            .autocapitalization(.words)
                        Picker("Document type", selection: $documentType) {
                            ForEach(documentTypes, id: \.self) { Text($0).tag($0) }
                        }
                        TextField("Document number", text: $documentNumber)
                            .autocapitalization(.none)
                            .disableAutocorrection(true)
                    }

                    Section {
                        Button(action: submitKyc) {
                            HStack {
                                if isSubmitting { ProgressView().tint(.orange) }
                                Text(isSubmitting ? "Submitting..." : "Start KYC")
                                    .fontWeight(.semibold)
                            }
                            .frame(maxWidth: .infinity)
                        }
                        .disabled(isSubmitting || fullName.trimmingCharacters(in: .whitespaces).isEmpty
                                  || documentNumber.trimmingCharacters(in: .whitespaces).isEmpty)
                    }
                }

                Section {
                    Button(action: loadStatus) {
                        Label("Refresh status", systemImage: "arrow.clockwise")
                    }
                }

                Section {
                    Text("KYC is required only for P2P trading.")
                        .font(.footnote)
                        .foregroundColor(.secondary)
                }
            }
            .navigationTitle("KYC")
            .onAppear { loadStatus() }
        }
    }

    private var statusRow: some View {
        HStack(alignment: .top) {
            Image(systemName: statusIcon)
                .foregroundColor(statusColor)
                .font(.title3)
            VStack(alignment: .leading, spacing: 4) {
                Text(statusLabel)
                    .font(.headline)
                if let rejection = rawStatus["rejection_reason"] as? String, !rejection.isEmpty {
                    Text("Reason: \(rejection)")
                        .font(.caption)
                        .foregroundColor(.red)
                }
            }
        }
    }

    private var verifiedBanner: some View {
        VStack(spacing: 6) {
            Label("KYC Verified — P2P trading enabled", systemImage: "checkmark.seal.fill")
                .font(.headline)
                .foregroundColor(.green)
            Text("Your identity is verified. You can place P2P trades.")
                .font(.caption)
                .foregroundColor(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 6)
    }

    private var statusLabel: String {
        switch status {
        case "verified", "approved": return "Status: verified"
        case "pending", "under_review": return "Status: pending review"
        case "rejected", "declined": return "Status: rejected"
        default: return "Status: not submitted"
        }
    }

    private var statusIcon: String {
        switch status {
        case "verified", "approved": return "checkmark.circle.fill"
        case "pending", "under_review": return "clock.fill"
        case "rejected", "declined": return "xmark.circle.fill"
        default: return "person.crop.circle.badge.questionmark"
        }
    }

    private var statusColor: Color {
        switch status {
        case "verified", "approved": return .green
        case "pending", "under_review": return .orange
        case "rejected", "declined": return .red
        default: return Color(.secondaryLabel)
        }
    }

    private func loadStatus() {
        isLoading = true
        error = nil
        Task {
            do {
                let json = try await UserWalletApiService.shared.getKycStatus(userId: nil)
                await MainActor.run {
                    self.rawStatus = json
                    self.status = (json["status"] as? String) ?? "not_submitted"
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

    private func submitKyc() {
        isSubmitting = true
        error = nil
        let name = fullName.trimmingCharacters(in: .whitespacesAndNewlines)
        let docType = documentType
        let docNumber = documentNumber.trimmingCharacters(in: .whitespacesAndNewlines)
        Task {
            do {
                // Step 1: begin the KYC registration session with the provider.
                _ = try await UserWalletApiService.shared.registerKyc([:])
                // Step 2: submit the actual identity payload.
                _ = try await UserWalletApiService.shared.submitKyc([
                    "full_name": name,
                    "document_type": docType,
                    "document_number": docNumber,
                ])
                await MainActor.run {
                    self.isSubmitting = false
                    self.loadStatus()
                }
            } catch {
                await MainActor.run {
                    self.error = error.localizedDescription
                    self.isSubmitting = false
                }
            }
        }
    }
}
