import Foundation
import UIKit

#if canImport(GTMAppAuth)
import GTMAppAuth
import AppAuth
#endif
#if canImport(GTLRDrive)
import GTLRDrive
#endif

// GoogleDriveBackup — REAL Google Drive REST API v3 backup of the recovery
// phrase via the Google Drive iOS SDK (GTMAppAuth for OAuth + GTLRDrive for the
// upload). Mirrors the web BackupMnemonic Google Drive path.
//
// Requires (added by the integrator in Xcode):
//   1. GTMAppAuth + GTLRDrive SPM packages added to the app target.
//   2. A Google OAuth client id in Info.plist under "GOOGLE_CLIENT_ID".
//   3. The OAuth redirect URI scheme registered in Info.plist URL types +
//      GOOGLE_REDIRECT_URI.
//
// If the client id is empty OR the SDK is not linked, the feature is HONESTLY
// disabled: `isConfigured` is false and `upload` throws a configuration error.
// It NEVER reports fake success.

enum GoogleDriveBackup {
    /// Reads GOOGLE_CLIENT_ID from the main bundle's Info.plist.
    static var clientID: String {
        Bundle.main.object(forInfoDictionaryKey: "GOOGLE_CLIENT_ID") as? String ?? ""
    }

    /// Reads GOOGLE_REDIRECT_URI from Info.plist (the OAuth redirect URI).
    static var redirectURI: String {
        Bundle.main.object(forInfoDictionaryKey: "GOOGLE_REDIRECT_URI") as? String ?? ""
    }

    /// True only when a client id + redirect are present AND the real SDK is
    /// linked into the target.
    static var isConfigured: Bool {
        guard !clientID.isEmpty, !redirectURI.isEmpty else { return false }
        #if canImport(GTMAppAuth) && canImport(GTLRDrive)
        return true
        #else
        return false
        #endif
    }

    static var notConfiguredMessage: String {
        "Google Drive backup not configured. Set GOOGLE_CLIENT_ID and GOOGLE_REDIRECT_URI in Info.plist and add GTMAppAuth + GTLRDrive via Swift Package Manager."
    }

    // drive.file — only files the app creates are visible (matches the web
    // BackupMnemonic scope).
    private static let scope = "https://www.googleapis.com/auth/drive.file"

    enum BackupError: LocalizedError {
        case notConfigured
        case authFailed(String)
        case uploadFailed(String)

        var errorDescription: String? {
            switch self {
            case .notConfigured: return GoogleDriveBackup.notConfiguredMessage
            case .authFailed(let m): return "Google auth failed: \(m)"
            case .uploadFailed(let m): return "Drive upload failed: \(m)"
            }
        }
    }

    /// Upload the mnemonic as a Drive text file via the real
    /// GTLRDriveQuery_FilesCreate multipart upload. Returns the created file id.
    /// Throws BackupError.notConfigured if the SDK/client id is missing (never
    /// fakes success).
    static func upload(mnemonic: String, walletId: String) async throws -> String {
        guard isConfigured else { throw BackupError.notConfigured }
        let authorizer = try await authorize()
        return try await uploadFile(mnemonic: mnemonic, walletId: walletId, authorizer: authorizer)
    }

    // MARK: - SDK-backed paths (only compiled when the packages are linked)

    /// Presents the real GTMAppAuth OID flow (system SFViewController / browser)
    /// for the configured client id + drive.file scope, returning an
    /// authorization object the Drive service can use.
    private static func authorize() async throws -> Any {
        return try await withCheckedThrowingContinuation { (cont: CheckedContinuation<Any, Error>) in
            DispatchQueue.main.async {
                #if canImport(GTMAppAuth)
                Self.runOIDFlow(clientID: GoogleDriveBackup.clientID,
                               redirectURI: GoogleDriveBackup.redirectURI,
                               scope: GoogleDriveBackup.scope) { result in
                    switch result {
                    case .success(let auth):
                        cont.resume(returning: auth)
                    case .failure(let err):
                        cont.resume(throwing: err)
                    }
                }
                #else
                cont.resume(throwing: BackupError.notConfigured)
                #endif
            }
        }
    }

    private static func uploadFile(mnemonic: String, walletId: String, authorizer: Any) async throws -> String {
        return try await withCheckedThrowingContinuation { (cont: CheckedContinuation<String, Error>) in
            DispatchQueue.main.async {
                #if canImport(GTLRDrive)
                Self.runGTLRUpload(mnemonic: mnemonic, walletId: walletId, authorizer: authorizer) { result in
                    switch result {
                    case .success(let fileId):
                        cont.resume(returning: fileId)
                    case .failure(let err):
                        cont.resume(throwing: err)
                    }
                }
                #else
                cont.resume(throwing: BackupError.notConfigured)
                #endif
            }
        }
    }

    #if canImport(GTMAppAuth)
    // Real AppAuth OIDAuthorizationRequest + system external agent. The
    // resulting OIDAuthState is wrapped in GTMAppAuthFetcherAuthorization which
    // authorizes the GTLRDriveService fetcher.
    private static func runOIDFlow(clientID: String, redirectURI: String, scope: String,
                                   completion: @escaping (Result<Any, Error>) -> Void) {
        guard let redirectURL = URL(string: redirectURI) else {
            completion(.failure(BackupError.authFailed("Invalid redirect URI")))
            return
        }
        let config = OIDServiceConfiguration(
            authorizationEndpoint: URL(string: "https://accounts.google.com/o/oauth2/auth")!,
            tokenEndpoint: URL(string: "https://oauth2.googleapis.com/token")!
        )
        let request = OIDAuthorizationRequest(configuration: config,
                                              clientId: clientID,
                                              scopes: [scope],
                                              redirectURL: redirectURL,
                                              responseType: OIDResponseTypeCode,
                                              additionalParameters: ["prompt": "consent"])
        let presenter = Self.topViewController()
        OIDAuthorizationService.present(request,
                                        externalUserAgent: OIDExternalUserAgentIOS(presenting: presenter)) { response, error in
            if let error = error {
                completion(.failure(BackupError.authFailed(error.localizedDescription)))
                return
            }
            guard let response = response else {
                completion(.failure(BackupError.authFailed("No authorization response")))
                return
            }
            let auth = GTMAppAuthFetcherAuthorization(authResponse: response)
            GTMAppAuthFetcherAuthorization.setCurrent(auth, for: clientID)
            completion(.success(auth as Any))
        }
    }
    #endif

    #if canImport(GTLRDrive)
    private static func runGTLRUpload(mnemonic: String, walletId: String, authorizer: Any,
                                      completion: @escaping (Result<String, Error>) -> Void) {
        let service = GTLRDriveService()
        service.authorizer = authorizer as? GTMFetcherAuthorizationProtocol

        let fileName = "tigerwallet-backup-\(String(walletId.prefix(8)))-\(Int(Date().timeIntervalSince1970)).txt"
        let metadata = GTLRDrive_File()
        metadata.name = fileName
        metadata.mimeType = "text/plain"

        let content = mnemonic.data(using: .utf8) ?? Data()
        let query = GTLRDriveQuery_FilesCreate()
        query.fields = "id"
        query.uploadParameters = GTLRUploadParameters(data: content, mimeType: "text/plain")
        query.file = metadata

        service.executeQuery(query) { ticket, result, error in
            if let error = error {
                completion(.failure(BackupError.uploadFailed(error.localizedDescription)))
                return
            }
            guard let file = result as? GTLRDrive_File, let id = file.identifier else {
                completion(.failure(BackupError.uploadFailed("No file id returned")))
                return
            }
            completion(.success(id))
            _ = ticket
        }
    }
    #endif

    /// Resolves the top-most view controller for presenting the OID flow.
    static func topViewController(_ base: UIViewController? = nil) -> UIViewController {
        var base = base
        if base == nil {
            if let scene = UIApplication.shared.connectedScenes
                .compactMap({ $0 as? UIWindowScene })
                .first(where: { $0.activationState == .foregroundActive }),
               let root = scene.windows.first(where: { $0.isKeyWindow })?.rootViewController {
                base = root
            } else {
                base = UIApplication.shared.delegate?.window??.rootViewController
            }
        }
        var top = base
        while let presented = top?.presentedViewController { top = presented }
        while let nav = top as? UINavigationController, let last = nav.viewControllers.last {
            top = last
        }
        while let tab = top as? UITabBarController, let selected = tab.selectedViewController {
            top = selected
        }
        return top ?? UIViewController()
    }
}
