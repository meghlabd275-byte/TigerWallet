import Foundation
import SwiftUI

// MARK: - White-label branding

/// White-label branding config for the iOS app.
///
/// Loading order:
///  1. `WL_BRANDING_SLUG` Info.plist key (configurable per WL-client build).
///     Absent/empty => stock TigerWallet build; no remote fetch happens.
///  2. If a slug is set, fetch `GET {CONTROL_PLANE_URL}/api/v1/branding/{slug}`
///     on app startup (URLSession). The endpoint is PUBLIC (no auth) so a
///     WL-branded app needs no secrets.
///  3. Cache the fetched JSON in UserDefaults so a transient network failure
///     or cold start still shows the WL brand instead of TigerWallet.
///  4. Fall back to TigerWallet defaults if there is no slug, the fetch fails,
///     or the endpoint returns 404 (no WL client matches the slug).
///
/// `Info.plist` `CFBundleDisplayName` remains "TigerWallet" as the launcher
/// label default; [BrandingConfig.shared.appName] overrides the in-app
/// displayed name at runtime (the springboard label can't be changed at
/// runtime, so the in-app title + theme colors are what we override).
struct Branding: Codable, Equatable {
    var slug: String = ""
    var appName: String = "TigerWallet"
    var logoUrl: String = ""
    var primaryColor: String = "#FF6B35"
    var secondaryColor: String = "#1E3A5F"
    var domain: String = "tigerwallet.io"
    var supportEmail: String = "support@tigerwallet.io"
    var termsUrl: String = "https://tigerwallet.io/terms"
    var privacyUrl: String = "https://tigerwallet.io/privacy"

    /// TigerWallet stock branding — the backward-compatible default.
    static let defaults = Branding()

    /// Coding keys map snake_case JSON from the control plane to Swift fields.
    enum CodingKeys: String, CodingKey {
        case slug, appName = "app_name", logoUrl = "logo_url",
             primaryColor = "primary_color", secondaryColor = "secondary_color",
             domain, supportEmail = "support_email",
             termsUrl = "terms_url", privacyUrl = "privacy_url"
    }

    init(slug: String = "", appName: String = "TigerWallet", logoUrl: String = "",
         primaryColor: String = "#FF6B35", secondaryColor: String = "#1E3A5F",
         domain: String = "tigerwallet.io", supportEmail: String = "support@tigerwallet.io",
         termsUrl: String = "https://tigerwallet.io/terms",
         privacyUrl: String = "https://tigerwallet.io/privacy") {
        self.slug = slug
        self.appName = appName
        self.logoUrl = logoUrl
        self.primaryColor = primaryColor
        self.secondaryColor = secondaryColor
        self.domain = domain
        self.supportEmail = supportEmail
        self.termsUrl = termsUrl
        self.privacyUrl = privacyUrl
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        // Merge over defaults: missing/empty fields fall back to TigerWallet.
        func val(_ key: CodingKeys, _ dflt: String) -> String {
            let s = (try? c.decode(String.self, forKey: key)) ?? ""
            return s.isEmpty ? dflt : s
        }
        self.init(
            slug: (try? c.decode(String.self, forKey: .slug)) ?? "",
            appName: val(.appName, Branding.defaults.appName),
            logoUrl: val(.logoUrl, Branding.defaults.logoUrl),
            primaryColor: val(.primaryColor, Branding.defaults.primaryColor),
            secondaryColor: val(.secondaryColor, Branding.defaults.secondaryColor),
            domain: val(.domain, Branding.defaults.domain),
            supportEmail: val(.supportEmail, Branding.defaults.supportEmail),
            termsUrl: val(.termsUrl, Branding.defaults.termsUrl),
            privacyUrl: val(.privacyUrl, Branding.defaults.privacyUrl)
        )
    }
}

final class BrandingConfig: ObservableObject {
    static let shared = BrandingConfig()

    /// Published so SwiftUI views can react to async branding refreshes.
    @Published private(set) var branding: Branding

    private let defaults = UserDefaults.standard
    private let cacheKey = "wl_branding_json"
    private let slugKey = "wl_branding_slug"

    /// WL_BRANDING_SLUG from Info.plist (per WL-client build). Empty => stock.
    let slug: String

    /// CONTROL_PLANE_URL from Info.plist; falls back to local dev control plane.
    private let controlPlaneUrl: String

    private let session: URLSession

    private init() {
        let info = Bundle.main.infoDictionary ?? [:]
        self.slug = (info["WL_BRANDING_SLUG"] as? String ?? "").trimmingCharacters(in: .whitespaces)
        let cp = (info["WL_CONTROL_PLANE_URL"] as? String ?? "").trimmingCharacters(in: .whitespaces)
        self.controlPlaneUrl = cp.isEmpty ? "http://localhost:9008" : cp

        let cfg = URLSessionConfiguration.default
        cfg.timeoutIntervalForRequest = 15
        cfg.timeoutIntervalForResource = 20
        cfg.waitsForConnectivity = false
        self.session = URLSession(configuration: cfg)

        // Load cached branding synchronously so first frame is WL-branded
        // when a cache is present and matches the current build's slug.
        self.branding = BrandingConfig.loadCached(slug: self.slug, defaults: defaults)
    }

    /// Load from cache, then async-refresh from the control plane. Call from
    /// `AppDelegate.application(_:didFinishLaunchingWithOptions:)`.
    func bootstrap() {
        guard !slug.isEmpty else { return }
        refresh()
    }

    private static func loadCached(slug: String, defaults ud: UserDefaults) -> Branding {
        guard let data = ud.data(forKey: "wl_branding_json"),
              let decoded = try? JSONDecoder().decode(Branding.self, from: data) else {
            return Branding.defaults
        }
        // Only trust a cache that matches the current build's slug.
        guard decoded.slug == slug || (slug.isEmpty && decoded.slug.isEmpty) else {
            return Branding.defaults
        }
        return decoded
    }

    /// Fetch `GET /api/v1/branding/{slug}` and apply on success. Failures are
    /// silent — the cached/default branding remains (backward compatible).
    private func refresh() {
        let trimmed = slug.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? slug
        guard let url = URL(string: "\(controlPlaneUrl)/api/v1/branding/\(trimmed)") else { return }
        var req = URLRequest(url: url)
        req.httpMethod = "GET"

        let task = session.dataTask(with: req) { [weak self] data, response, _ in
            guard let self = self,
                  let http = response as? HTTPURLResponse,
                  (200..<300).contains(http.statusCode),
                  let data = data,
                  let decoded = try? JSONDecoder().decode(Branding.self, from: data) else {
                return
            }
            if let encoded = try? JSONEncoder().encode(decoded) {
                self.defaults.set(encoded, forKey: "wl_branding_json")
                self.defaults.set(self.slug, forKey: self.slugKey)
            }
            DispatchQueue.main.async {
                self.branding = decoded
            }
        }
        task.resume()
    }

    // --- Convenience accessors ---

    var appName: String { branding.appName }
    var logoUrl: String { branding.logoUrl }
    var primaryColor: String { branding.primaryColor }
    var secondaryColor: String { branding.secondaryColor }
    var domain: String { branding.domain }
    var supportEmail: String { branding.supportEmail }

    /// Parse the WL primary color into a SwiftUI Color (fails closed to default).
    var primarySwiftUIColor: Color { Color(hex: branding.primaryColor) ?? Color(hex: Branding.defaults.primaryColor)! }
    var secondarySwiftUIColor: Color { Color(hex: branding.secondaryColor) ?? Color(hex: Branding.defaults.secondaryColor)! }
}

// MARK: - Hex color helper

extension Color {
    /// Init from a `#RRGGBB` / `RRGGBB` / `#RRGGBBAA` hex string. nil on failure.
    init?(hex: String) {
        var s = hex.trimmingCharacters(in: .whitespaces)
        if s.hasPrefix("#") { s.removeFirst() }
        guard let v = UInt32(s, radix: 16) else { return nil }
        let r, g, b, a: Double
        switch s.count {
        case 6:
            r = Double((v >> 16) & 0xFF) / 255.0
            g = Double((v >> 8) & 0xFF) / 255.0
            b = Double(v & 0xFF) / 255.0
            a = 1.0
        case 8:
            r = Double((v >> 24) & 0xFF) / 255.0
            g = Double((v >> 16) & 0xFF) / 255.0
            b = Double((v >> 8) & 0xFF) / 255.0
            a = Double(v & 0xFF) / 255.0
        default:
            return nil
        }
        self.init(.sRGB, red: r, green: g, blue: b, opacity: a)
    }
}
