import SwiftUI

enum WLTheme {
    static func background(isDark: Bool) -> Color { isDark ? Color(red: 0.10, green: 0.10, blue: 0.16) : Color(red: 0.96, green: 0.96, blue: 0.98) }
    static func cardBackground(isDark: Bool) -> Color { isDark ? Color(red: 0.16, green: 0.18, blue: 0.26) : Color.white }
    static func primaryText(isDark: Bool) -> Color { isDark ? Color.white : Color(red: 0.10, green: 0.10, blue: 0.14) }
    static func secondaryText(isDark: Bool) -> Color { isDark ? Color(white: 0.70) : Color(white: 0.45) }
    static func accent(isDark: Bool) -> Color { Color(red: 0.10, green: 0.46, blue: 0.98) }
}

struct DomainLink: Identifiable {
    let id: String
    let key: String
    let label: String
    let icon: String
}

struct ContentView: View {
    @AppStorage("wl_dark_mode") private var isDark: Bool = false

    private let domains: [DomainLink] = [
        DomainLink(id: "dashboard", key: "dashboard", label: "Dashboard", icon: "chart.bar"),
        DomainLink(id: "users", key: "users", label: "Users", icon: "person.2"),
        DomainLink(id: "futures", key: "futures", label: "Futures", icon: "chart.line.uptrend.xyaxis"),
        DomainLink(id: "options", key: "options", label: "Options", icon: "chart.bar.doc.horizontal"),
        DomainLink(id: "copy-trading", key: "copy-trading", label: "Copy Trading", icon: "person.crop.circle.badge.plus"),
        DomainLink(id: "convert", key: "convert", label: "Convert", icon: "arrow.left.arrow.right.square"),
        DomainLink(id: "onramp", key: "onramp", label: "On-Ramp", icon: "arrow.down.right.circle"),
        DomainLink(id: "offramp", key: "offramp", label: "Off-Ramp", icon: "arrow.up.right.circle"),
        DomainLink(id: "p2p-clients", key: "p2p-clients", label: "P2P Clients", icon: "person.3.sequence"),
        DomainLink(id: "partners", key: "partners", label: "Partners", icon: "hand.raised.fingers.spread"),
        DomainLink(id: "rewards", key: "rewards", label: "Rewards", icon: "gift"),
        DomainLink(id: "marketing", key: "marketing", label: "Marketing", icon: "megaphone"),
        DomainLink(id: "rbac", key: "rbac", label: "Admin Roles", icon: "person.badge.shield.checkmark"),
        DomainLink(id: "settings", key: "settings", label: "Settings", icon: "gear"),
    ]

    var body: some View {
        NavigationView {
            List {
                Section(header: Text("White Label Admin")) {
                    ForEach(domains) { domain in
                        NavigationLink(destination: DomainView(domainKey: domain.key, isDark: isDark).navigationTitle(domain.label)) {
                            Label(domain.label, systemImage: domain.icon)
                                .foregroundColor(WLTheme.primaryText(isDark: isDark))
                        }
                    }
                }
            }
            .navigationTitle("WL Admin")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { isDark.toggle() }) {
                        Image(systemName: isDark ? "sun.max.fill" : "moon.fill")
                    }
                }
            }
        }
        .preferredColorScheme(isDark ? .dark : .light)
    }
}

struct DomainSpec {
    let title: String
    let subtitle: String
    let endpoints: [String]
    let statusActions: [String]
}

func specFor(_ key: String) -> DomainSpec {
    switch key {
    case "dashboard": return DomainSpec(title: "Dashboard", subtitle: "Platform overview stats.", endpoints: ["GET /api/v1/admin/stats"], statusActions: [])
    case "users": return DomainSpec(title: "Users", subtitle: "Manage platform users.", endpoints: ["GET /api/v1/admin/users"], statusActions: [])
    case "futures": return DomainSpec(title: "Futures Positions", subtitle: "Futures positions governance records.", endpoints: ["GET /futures", "POST /futures", "PUT /futures/:id", "DELETE /futures/:id"], statusActions: ["PUT /futures/:id/status"])
    case "options": return DomainSpec(title: "Options Contracts", subtitle: "Options contracts governance records.", endpoints: ["GET /options", "POST /options", "PUT /options/:id", "DELETE /options/:id"], statusActions: ["PUT /options/:id/status"])
    case "copy-trading": return DomainSpec(title: "Copy Trading", subtitle: "Copy-trading configs governance records.", endpoints: ["GET /copy-trading", "POST /copy-trading", "PUT /copy-trading/:id", "DELETE /copy-trading/:id"], statusActions: ["PUT /copy-trading/:id/status"])
    case "convert": return DomainSpec(title: "Convert Orders", subtitle: "Convert orders governance records.", endpoints: ["GET /convert", "POST /convert", "PUT /convert/:id", "DELETE /convert/:id"], statusActions: ["PUT /convert/:id/status"])
    case "onramp": return DomainSpec(title: "On-Ramp Orders", subtitle: "On-ramp order governance (approve/reject).", endpoints: ["GET /onramp", "POST /onramp", "PUT /onramp/:id", "DELETE /onramp/:id"], statusActions: ["POST /onramp/:id/approve", "POST /onramp/:id/reject {reason}"])
    case "offramp": return DomainSpec(title: "Off-Ramp Orders", subtitle: "Off-ramp order governance (approve/reject).", endpoints: ["GET /offramp", "POST /offramp", "PUT /offramp/:id", "DELETE /offramp/:id"], statusActions: ["POST /offramp/:id/approve", "POST /offramp/:id/reject {reason}"])
    case "p2p-clients": return DomainSpec(title: "P2P Clients", subtitle: "P2P merchant/client governance.", endpoints: ["GET /p2p-clients", "POST /p2p-clients", "PUT /p2p-clients/:id", "DELETE /p2p-clients/:id"], statusActions: ["PUT /p2p-clients/:id/status"])
    case "partners": return DomainSpec(title: "Partners", subtitle: "Partner governance (status + approve/reject).", endpoints: ["GET /partners", "POST /partners", "PUT /partners/:id", "DELETE /partners/:id"], statusActions: ["PUT /partners/:id/status", "POST /partners/:id/approve", "POST /partners/:id/reject {reason}"])
    case "rewards": return DomainSpec(title: "Reward Campaigns", subtitle: "Reward campaigns governance records.", endpoints: ["GET /rewards", "POST /rewards", "PUT /rewards/:id", "DELETE /rewards/:id"], statusActions: ["PUT /rewards/:id/status"])
    case "marketing": return DomainSpec(title: "Marketing Campaigns", subtitle: "Marketing campaigns governance records.", endpoints: ["GET /marketing", "POST /marketing", "PUT /marketing/:id", "DELETE /marketing/:id"], statusActions: ["PUT /marketing/:id/status"])
    case "rbac": return DomainSpec(title: "Admin Roles & Permissions", subtitle: "Structured RBAC over the scope system.", endpoints: ["GET /admin-roles", "POST /admin-roles", "GET /admin-permissions", "POST /admins/:id/role", "GET /admins/:id/permissions"], statusActions: ["DELETE /admins/:id/role/:roleId"])
    default: return DomainSpec(title: "Settings", subtitle: "Application settings.", endpoints: [], statusActions: [])
    }
}

struct DomainView: View {
    let domainKey: String
    let isDark: Bool
    private var spec: DomainSpec { specFor(domainKey) }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text(spec.subtitle)
                    .font(.subheadline)
                    .foregroundColor(WLTheme.secondaryText(isDark: isDark))
                WLCard(isDark: isDark) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("CRUD Endpoints").font(.headline).foregroundColor(WLTheme.primaryText(isDark: isDark))
                        ForEach(spec.endpoints, id: \.self) { ep in
                            Text(ep).font(.system(.body, design: .monospaced))
                                .foregroundColor(WLTheme.primaryText(isDark: isDark))
                        }
                        if !spec.statusActions.isEmpty {
                            Divider().background(WLTheme.secondaryText(isDark: isDark))
                            Text("Governance Actions").font(.headline).foregroundColor(WLTheme.primaryText(isDark: isDark))
                            ForEach(spec.statusActions, id: \.self) { act in
                                Text(act).font(.system(.body, design: .monospaced))
                                    .foregroundColor(WLTheme.accent(isDark: isDark))
                            }
                        }
                        Text("No fund movement - governance records only.")
                            .font(.caption)
                            .foregroundColor(WLTheme.secondaryText(isDark: isDark))
                    }
                }
            }.padding()
        }
        .background(WLTheme.background(isDark: isDark).ignoresSafeArea())
    }
}

struct WLCard<Content: View>: View {
    let isDark: Bool
    @ViewBuilder let content: Content
    var body: some View {
        content.padding()
            .background(WLTheme.cardBackground(isDark: isDark))
            .cornerRadius(12)
            .shadow(color: Color.black.opacity(0.08), radius: 4, y: 2)
    }
}

struct DashboardView: View {
    @AppStorage("wl_dark_mode") private var isDark: Bool = false
    var body: some View {
        VStack(spacing: 20) {
            Text("Dashboard").font(.largeTitle).foregroundColor(WLTheme.primaryText(isDark: isDark))
            HStack(spacing: 20) {
                StatCard(title: "Users", value: "0", isDark: isDark)
                StatCard(title: "Transactions", value: "0", isDark: isDark)
            }
        }.padding()
    }
}

struct StatCard: View {
    let title: String
    let value: String
    var isDark: Bool = false
    var body: some View {
        VStack {
            Text(value).font(.title).foregroundColor(WLTheme.primaryText(isDark: isDark))
            Text(title).foregroundColor(WLTheme.secondaryText(isDark: isDark))
        }
        .frame(width: 120, height: 80)
        .background(WLTheme.cardBackground(isDark: isDark))
        .cornerRadius(10)
    }
}
