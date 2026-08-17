import AuthenticationServices
import Foundation
import UIKit

// GoogleDriveBackupHelper — REAL Google Drive REST API v3 (appDataFolder)
// encrypted-seed backup for the TigerWallet iOS UserWallet client.
//
// Auth model: pure OAuth2 Authorization Code flow via ASWebAuthenticationSession
// (no GoogleSignIn SDK dependency). The user signs in through Google's web
// consent in a system browser tab; the authorization code is exchanged for an
// access token scoped to `https://www.googleapis.com/auth/drive.appdata`. The
// token is held in-memory only (never persisted to disk).
//
// The encrypted seed blob is opaque to this helper: the wallet core has already
// AES-256-GCM encrypted it before calling in. This helper NEVER decrypts,
// fabricates, or synthesizes seed data. It only uploads/downloads an opaque
// String to/from the user's hidden Drive `appDataFolder` (invisible to the user
// and isolated from their normal Drive files).
//
// Fail-closed: if [googleDriveClientID] (or its redirect URI) is not configured,
// every operation throws [GoogleDriveBackupError.notConfigured] rather than
// silently no-op'ing or faking a backup.

/// OAuth2 / Drive configuration. Replace these with the values from your
/// Google Cloud Console (APIs & Services → Credentials → OAuth 2.0 Client ID
/// of type "iOS" or "Web application"). Empty by default → fail closed.
enum GoogleDriveConfig {
    /// OAuth2 client_id from the Google Cloud Console. Empty = fail closed.
    static let googleDriveClientID: String = ""

    /// OAuth2 client_secret. Required for the installed-app / web exchange.
    /// For native apps, use a "Web application" client created for this app
    /// (Google's loopback/installed-app flow); leave empty to fail closed.
    static let googleDriveClientSecret: String = ""

    /// Redirect URI scheme the app registers. The ASWebAuthenticationSession
    /// callback URL must match an entry in Info.plist's
    /// `CFBundleURLTypes`. Empty = fail closed.
    static let redirectURIScheme: String = "com.tigerwallet.userwallet"
}

/// Descriptive errors thrown by the helper. Callers should never receive a bare
/// `Error`; each case carries enough context to surface a real message.
enum GoogleDriveBackupError: Error, LocalizedError {
    /// Client_id/redirect_uri not configured — fail closed.
    case notConfigured(String)
    /// The OAuth2 consent / token exchange failed.
    case oauthFailed(String)
    /// The user cancelled the ASWebAuthenticationSession prompt.
    case cancelled
    /// A Drive REST API call returned a non-2xx status.
    case driveAPIFailed(status: Int, message: String)
    /// Restore was requested but no backup file exists in appDataFolder.
    case noBackupFound
    /// The response could not be decoded into the expected shape.
    case decodingFailed(String)

    var errorDescription: String? {
        switch self {
        case .notConfigured(let m): return "Google Drive backup not configured: \(m)"
        case .oauthFailed(let m): return "Google OAuth2 failed: \(m)"
        case .cancelled: return "Google Drive sign-in was cancelled."
        case .driveAPIFailed(let s, let m): return "Google Drive API error (\(s)): \(m)"
        case .noBackupFound:
            return "No TigerWallet backup file was found in Google Drive appDataFolder."
        case .decodingFailed(let m): return "Google Drive response decoding failed: \(m)"
        }
    }
}

/// Holds an OAuth2 access token (in-memory only) plus its expiry.
private struct AccessToken {
    let value: String
    let expiresAt: Date
}

enum GoogleDriveBackupHelper {

    /// Name of the backup file inside the Drive `appDataFolder`.
    static let backupFileName = "tigerwallet-wallet-backup.enc"
    /// MIME type for the opaque encrypted blob.
    static let backupMimeType = "application/octet-stream"
    /// Drive `appDataFolder` special folder id.
    static let appDataFolder = "appDataFolder"

    /// The Drive scope that grants access only to the hidden appDataFolder.
    static let driveAppDataScope = "https://www.googleapis.com/auth/drive.appdata"

    /// In-memory cached token; never persisted. Cleared on app termination.
    private static var cachedToken: AccessToken?
    private static let tokenLock = NSLock()

    // MARK: - Public API

    /// Whether this device can present a Google OAuth2 consent (always true on
    /// iOS 12+, which ships ASWebAuthenticationSession). Cheap, non-blocking.
    static var isAvailable: Bool { true }

    /// Backs up [encryptedSeedBlob] to Google Drive `appDataFolder`.
    ///
    /// If a backup file named [backupFileName] already exists in the
    /// appDataFolder, it is **updated** in place (PATCH) using the same file id;
    /// otherwise a new file is created (POST) with `parents=[appDataFolder]`.
    /// The blob is uploaded as raw UTF-8 bytes.
    ///
    /// - Parameter encryptedSeedBlob: The already-encrypted seed blob.
    /// - Returns: The Drive file id of the (created or updated) backup file.
    /// - Throws: `GoogleDriveBackupError` on any failure; never returns a
    ///   fabricated id.
    static func backupToDrive(encryptedSeedBlob: String) async throws -> String {
        try requireConfigured()
        let token = try await ensureAccessToken()

        let existingId = try await findBackupFileId(token: token)
        if let fileId = existingId {
            try await updateFile(id: fileId, content: encryptedSeedBlob, token: token)
            return fileId
        } else {
            let fileId = try await createFile(content: encryptedSeedBlob, token: token)
            return fileId
        }
    }

    /// Restores the encrypted seed blob from Google Drive `appDataFolder`.
    ///
    /// Searches for the backup file; if found, downloads and returns its raw
    /// content as a UTF-8 String. If no backup file exists, returns `nil`
    /// (distinct from a failure — a missing backup is a valid restore outcome).
    ///
    /// - Returns: The encrypted blob, or `nil` if no backup exists.
    /// - Throws: `GoogleDriveBackupError` on any failure; never fabricates content.
    static func restoreFromDrive() async throws -> String? {
        try requireConfigured()
        let token = try await ensureAccessToken()

        guard let fileId = try await findBackupFileId(token: token) else {
            return nil
        }
        return try await downloadFile(id: fileId, token: token)
    }

    // MARK: - Configuration

    /// Fails closed if the OAuth2 client_id/secret/redirect are not configured.
    private static func requireConfigured() throws {
        var missing: [String] = []
        if GoogleDriveConfig.googleDriveClientID.isEmpty { missing.append("googleDriveClientID") }
        if GoogleDriveConfig.googleDriveClientSecret.isEmpty { missing.append("googleDriveClientSecret") }
        if GoogleDriveConfig.redirectURIScheme.isEmpty { missing.append("redirectURIScheme") }
        if !missing.isEmpty {
            throw GoogleDriveBackupError.notConfigured(
                "missing \(missing.joined(separator: ", ")). Set these in GoogleDriveConfig."
            )
        }
    }

    private static var redirectURI: String {
        "\(GoogleDriveConfig.redirectURIScheme):/oauth2redirect"
    }

    // MARK: - OAuth2 via ASWebAuthenticationSession

    /// Returns a non-expired access token, obtaining one via the OAuth2
    /// Authorization Code flow (ASWebAuthenticationSession) if needed.
    private static func ensureAccessToken() async throws -> String {
        tokenLock.lock()
        let cached = cachedToken
        tokenLock.unlock()
        if let t = cached, t.expiresAt > Date().addingTimeInterval(60) {
            return t.value
        }

        let code = try await requestAuthorizationCode()
        let token = try await exchangeCodeForToken(code: code)
        tokenLock.lock()
        cachedToken = token
        tokenLock.unlock()
        return token.value
    }

    /// Runs ASWebAuthenticationSession to obtain a single-use authorization code.
    private static func requestAuthorizationCode() async throws -> String {
        let authURL = URL(string: (
            "https://accounts.google.com/o/oauth2/v2/auth" +
            "?client_id=\(percentEncoded(GoogleDriveConfig.googleDriveClientID))" +
            "&redirect_uri=\(percentEncoded(redirectURI))" +
            "&response_type=code" +
            "&scope=\(percentEncoded(driveAppDataScope))" +
            "&access_type=offline" +
            "&prompt=consent"
        ))!

        return try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<String, Error>) in
            let session = ASWebAuthenticationSession(
                url: authURL,
                callbackURLScheme: GoogleDriveConfig.redirectURIScheme
            ) { callbackURL, error in
                if let error = error as? ASWebAuthenticationSessionError,
                   error.code == .canceledLogin {
                    continuation.resume(throwing: GoogleDriveBackupError.cancelled)
                    return
                }
                if let error = error {
                    continuation.resume(throwing: GoogleDriveBackupError.oauthFailed(
                        error.localizedDescription
                    ))
                    return
                }
                guard let callbackURL = callbackURL,
                      let comps = URLComponents(url: callbackURL, resolvingAgainstBaseURL: false),
                      let code = comps.queryItems?.first(where: { $0.name == "code" })?.value,
                      !code.isEmpty else {
                    continuation.resume(throwing: GoogleDriveBackupError.oauthFailed(
                        "no authorization code in callback URL"
                    ))
                    return
                }
                continuation.resume(returning: code)
            }
            // Requires a presentation context provider on iOS 13+.
            let presenter = WebAuthPresenter()
            session.presentationContextProvider = presenter
            session.prefersEphemeralWebBrowserSession = false
            presenter.session = session // retain for the prompt duration
            if !session.start() {
                continuation.resume(throwing: GoogleDriveBackupError.oauthFailed(
                    "ASWebAuthenticationSession failed to start"
                ))
            }
        }
    }

    /// Exchanges an authorization code for an access token (and optional refresh
    /// token) via the Google token endpoint.
    private static func exchangeCodeForToken(code: String) async throws -> AccessToken {
        var req = URLRequest(url: URL(string: "https://oauth2.googleapis.com/token")!)
        req.httpMethod = "POST"
        req.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
        let body = (
            "code=\(percentEncoded(code))" +
            "&client_id=\(percentEncoded(GoogleDriveConfig.googleDriveClientID))" +
            "&client_secret=\(percentEncoded(GoogleDriveConfig.googleDriveClientSecret))" +
            "&redirect_uri=\(percentEncoded(redirectURI))" +
            "&grant_type=authorization_code"
        )
        req.httpBody = body.data(using: .utf8)

        let (data, response) = try await dataTask(for: req)
        guard let http = response as? HTTPURLResponse else {
            throw GoogleDriveBackupError.oauthFailed("token endpoint returned no HTTP response")
        }
        guard http.statusCode == 200 else {
            throw GoogleDriveBackupError.oauthFailed(
                "token exchange HTTP \(http.statusCode): \(String(data: data, encoding: .utf8) ?? "")"
            )
        }
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let accessToken = json["access_token"] as? String,
              let expiresIn = json["expires_in"] as? TimeInterval else {
            throw GoogleDriveBackupError.decodingFailed("token response missing access_token/expires_in")
        }
        return AccessToken(
            value: accessToken,
            expiresAt: Date().addingTimeInterval(expiresIn)
        )
    }

    // MARK: - Drive REST

    /// Searches the appDataFolder for [backupFileName]; returns its id or nil.
    private static func findBackupFileId(token: String) async throws -> String? {
        var comps = URLComponents(string: "https://www.googleapis.com/drive/v3/files")!
        let q = "name = '\(backupFileName)' and trashed = false"
        comps.queryItems = [
            URLQueryItem(name: "spaces", value: appDataFolder),
            URLQueryItem(name: "q", value: q),
            URLQueryItem(name: "fields", value: "files(id, name)")
        ]
        var req = URLRequest(url: comps.url!)
        req.httpMethod = "GET"
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await dataTask(for: req)
        try assertOK(response, data, context: "files.list")
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let files = json["files"] as? [[String: Any]] else {
            throw GoogleDriveBackupError.decodingFailed("files.list response missing files array")
        }
        return files.first?["id"] as? String
    }

    /// Creates a new file in appDataFolder with the given content; returns its id.
    private static func createFile(content: String, token: String) async throws -> String {
        // Drive multipart upload: metadata (parents=appDataFolder) + raw media.
        let boundary = "tigerwallet_\(UUID().uuidString)"
        var req = URLRequest(
            url: URL(string: "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&fields=id")!
        )
        req.httpMethod = "POST"
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        req.setValue("multipart/related; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")

        let metadata: [String: Any] = [
            "name": backupFileName,
            "mimeType": backupMimeType,
            "parents": [appDataFolder]
        ]
        let metadataJSON = try JSONSerialization.data(withJSONObject: metadata, options: [])
        let mediaData = content.data(using: .utf8) ?? Data()

        var body = Data()
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Type: application/json; charset=UTF-8\r\n\r\n".data(using: .utf8)!)
        body.append(metadataJSON)
        body.append("\r\n--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Type: \(backupMimeType)\r\n\r\n".data(using: .utf8)!)
        body.append(mediaData)
        body.append("\r\n--\(boundary)--\r\n".data(using: .utf8)!)
        req.httpBody = body

        let (data, response) = try await dataTask(for: req)
        try assertOK(response, data, context: "files.create")
        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let id = json["id"] as? String else {
            throw GoogleDriveBackupError.decodingFailed("files.create response missing id")
        }
        return id
    }

    /// Updates an existing file's content via PATCH (media) — metadata unchanged.
    private static func updateFile(id: String, content: String, token: String) async throws {
        let boundary = "tigerwallet_\(UUID().uuidString)"
        var req = URLRequest(
            url: URL(string: "https://www.googleapis.com/upload/drive/v3/files/\(id)?uploadType=multipart&fields=id")!
        )
        req.httpMethod = "PATCH"
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        req.setValue("multipart/related; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")

        // Empty metadata object on update keeps the existing name/parents.
        let metadataJSON = "{}".data(using: .utf8)!
        let mediaData = content.data(using: .utf8) ?? Data()

        var body = Data()
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Type: application/json; charset=UTF-8\r\n\r\n".data(using: .utf8)!)
        body.append(metadataJSON)
        body.append("\r\n--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Type: \(backupMimeType)\r\n\r\n".data(using: .utf8)!)
        body.append(mediaData)
        body.append("\r\n--\(boundary)--\r\n".data(using: .utf8)!)
        req.httpBody = body

        let (data, response) = try await dataTask(for: req)
        try assertOK(response, data, context: "files.update")
    }

    /// Downloads a file's raw content as a UTF-8 String.
    private static func downloadFile(id: String, token: String) async throws -> String {
        var req = URLRequest(
            url: URL(string: "https://www.googleapis.com/drive/v3/files/\(id)?alt=media")!
        )
        req.httpMethod = "GET"
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let (data, response) = try await dataTask(for: req)
        try assertOK(response, data, context: "files.get")
        guard let text = String(data: data, encoding: .utf8) else {
            throw GoogleDriveBackupError.decodingFailed("downloaded backup is not valid UTF-8")
        }
        return text
    }

    // MARK: - Networking helpers

    /// Runs a URLRequest via URLSession and returns (data, response).
    private static func dataTask(for request: URLRequest) async throws -> (Data, URLResponse) {
        do {
            return try await URLSession.shared.data(for: request)
        } catch {
            throw GoogleDriveBackupError.oauthFailed("network error: \(error.localizedDescription)")
        }
    }

    /// Throws a descriptive [GoogleDriveBackupError.driveAPIFailed] if the HTTP
    /// status is not in the 2xx range.
    private static func assertOK(
        _ response: URLResponse,
        _ data: Data,
        context: String
    ) throws {
        guard let http = response as? HTTPURLResponse else {
            throw GoogleDriveBackupError.driveAPIFailed(
                status: -1,
                message: "\(context): no HTTP response"
            )
        }
        guard (200..<300).contains(http.statusCode) else {
            let body = String(data: data, encoding: .utf8) ?? "<binary>"
            throw GoogleDriveBackupError.driveAPIFailed(
                status: http.statusCode,
                message: "\(context): \(body)"
            )
        }
    }

    /// Percent-encodes a string for use in an `application/x-www-form-urlencoded`
    /// body or a URL query value.
    private static func percentEncoded(_ s: String) -> String {
        var allowed = CharacterSet.urlQueryAllowed
        // urlQueryAllowed is too permissive for form bodies; remove the
        // characters that must be encoded in form data.
        allowed.remove(charactersIn: "?=&/:%")
        return s.addingPercentEncoding(withAllowedCharacters: allowed) ?? s
    }
}

// MARK: - ASWebAuthenticationSession presentation

/// Provides the presentation anchor for ASWebAuthenticationSession. Strongly
/// retains the live session so it is not deallocated mid-prompt.
private final class WebAuthPresenter: NSObject, ASWebAuthenticationPresentationContextProviding {
    var session: ASWebAuthenticationSession?

    func presentationAnchor(for session: ASWebAuthenticationSession) -> ASPresentationAnchor {
        // Resolve the foreground key window from the active scene.
        if let scene = UIApplication.shared.connectedScenes
            .first(where: { $0.activationState == .foregroundActive }) as? UIWindowScene {
            if let w = scene.windows.first(where: { $0.isKeyWindow }) ?? scene.windows.first {
                return w
            }
        }
        return ASPresentationAnchor()
    }
}
