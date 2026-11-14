import SwiftUI

// MARK: - Shared UI building blocks

/// Renders a backend error with a retry affordance. Every screen uses this
/// instead of showing an empty list, so a failure is never mistaken for
/// "no data".
struct ErrorStateView: View {
    let message: String
    var retry: (() async -> Void)?

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle")
                .font(.largeTitle)
                .foregroundStyle(.orange)
            Text(message)
                .multilineTextAlignment(.center)
                .font(.callout)
                .foregroundStyle(.secondary)
            if let retry {
                Button("Retry") { Task { await retry() } }
                    .buttonStyle(.borderedProminent)
            }
        }
        .padding()
        .frame(maxWidth: .infinity)
    }
}

struct EmptyStateView: View {
    let title: String
    let systemImage: String
    var detail: String?

    var body: some View {
        VStack(spacing: 10) {
            Image(systemName: systemImage)
                .font(.largeTitle)
                .foregroundStyle(.secondary)
            Text(title).font(.headline)
            if let detail {
                Text(detail)
                    .font(.callout)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }
        }
        .padding()
        .frame(maxWidth: .infinity)
    }
}

/// A labelled value row used across the detail screens.
struct DetailRow: View {
    let label: String
    let value: String

    var body: some View {
        HStack(alignment: .top) {
            Text(label)
                .foregroundStyle(.secondary)
            Spacer()
            Text(value.isEmpty ? "—" : value)
                .multilineTextAlignment(.trailing)
        }
        .font(.callout)
    }
}

enum BotFormat {
    /// Formats an arbitrary JSON value the backend may return as a string,
    /// number or bool without inventing a placeholder.
    static func value(_ raw: Any?) -> String {
        switch raw {
        case nil, is NSNull:
            return "—"
        case let s as String:
            return s
        case let b as Bool:
            return b ? "true" : "false"
        case let n as NSNumber:
            return NumberFormatter.localizedString(from: n, number: .decimal)
        case let d as [String: Any]:
            return d.isEmpty ? "—" : d.map { "\($0.key): \(value($0.value))" }.joined(separator: ", ")
        case let a as [Any]:
            return a.isEmpty ? "—" : a.map { value($0) }.joined(separator: ", ")
        default:
            return String(describing: raw!)
        }
    }

    /// Pulls the first array found under any of `keys`, or the raw array.
    static func array(_ dict: [String: Any], keys: [String]) -> [[String: Any]] {
        for key in keys {
            if let arr = dict[key] as? [[String: Any]] { return arr }
        }
        if let arr = dict["data"] as? [[String: Any]] { return arr }
        if let arr = dict["items"] as? [[String: Any]] { return arr }
        return []
    }

    static func shortDate(_ raw: Any?) -> String {
        guard let s = raw as? String, !s.isEmpty else { return "—" }
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let fallback = ISO8601DateFormatter()
        if let d = iso.date(from: s) ?? fallback.date(from: s) {
            let out = DateFormatter()
            out.dateStyle = .medium
            out.timeStyle = .short
            return out.string(from: d)
        }
        return s
    }
}

// MARK: - Auth

struct AuthView: View {
    @EnvironmentObject var session: BotSession
    @State private var mode: Mode = .login
    @State private var username = ""
    @State private var password = ""
    @State private var email = ""
    @State private var error: String?
    @State private var busy = false

    enum Mode { case login, register }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Picker("Mode", selection: $mode) {
                        Text("Sign in").tag(Mode.login)
                        Text("Register").tag(Mode.register)
                    }
                    .pickerStyle(.segmented)
                }

                Section("Credentials") {
                    TextField("Username", text: $username)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    SecureField("Password", text: $password)
                    if mode == .register {
                        TextField("Email (optional)", text: $email)
                            .textInputAutocapitalization(.never)
                            .keyboardType(.emailAddress)
                            .autocorrectionDisabled()
                    }
                }

                if let error {
                    Section {
                        Text(error).foregroundStyle(.red).font(.callout)
                    }
                }

                Section {
                    Button {
                        Task { await submit() }
                    } label: {
                        HStack {
                            Spacer()
                            if busy { ProgressView() } else {
                                Text(mode == .login ? "Sign in" : "Create account")
                            }
                            Spacer()
                        }
                    }
                    .disabled(busy || username.isEmpty || password.isEmpty)
                }
            }
            .navigationTitle("TigerBots")
        }
    }

    private func submit() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let auth: BotAuthResponse
            if mode == .login {
                auth = try await BotApiService.shared.login(
                    username: username, password: password
                )
            } else {
                auth = try await BotApiService.shared.register(
                    username: username,
                    password: password,
                    email: email.isEmpty ? nil : email
                )
            }
            session.signedIn(auth)
        } catch {
            self.error = error.localizedDescription
        }
    }
}
