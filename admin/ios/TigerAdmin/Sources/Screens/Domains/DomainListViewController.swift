//
//  DomainListViewController.swift
//  TigerAdmin
//
//  Generic admin domain list screen with loading / error / empty states.
//  Theme: uses .systemBackground / .label / .secondaryLabel which auto-adapt to
//  light/dark via ThemeManager's window.overrideUserInterfaceStyle.
//

import UIKit
import SnapKit

class DomainListViewController: UIViewController {

    // Closure-based configuration supplied by each concrete domain VC.
    struct Config {
        let title: String
        let load: () async throws -> [DomainRecord]
        let primaryTitle: String?
        let primaryAction: ((DomainRecord, DomainListViewController) async -> Void)?
        let secondaryTitle: String?
        let secondaryAction: ((DomainRecord, DomainListViewController) async -> Void)?
        let rowTitle: (DomainRecord) -> String
        let rowSubtitle: (DomainRecord) -> String
        let rowStatus: (DomainRecord) -> String?
    }

    let api = AdminAPIService()
    let config: Config

    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let activityIndicator = UIActivityIndicatorView(style: .large)
    private let errorLabel = UILabel()
    private let emptyLabel = UILabel()
    private var records: [DomainRecord] = []

    init(config: Config) {
        self.config = config
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = config.title
        view.backgroundColor = .systemBackground
        navigationController?.navigationBar.prefersLargeTitles = true

        setupUI()
        loadData()
    }

    private func setupUI() {
        tableView.delegate = self
        tableView.dataSource = self
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "DomainCell")
        tableView.isHidden = true
        view.addSubview(tableView)

        activityIndicator.hidesWhenStopped = true
        view.addSubview(activityIndicator)

        errorLabel.textColor = ThemeManager.errorColor
        errorLabel.textAlignment = .center
        errorLabel.numberOfLines = 0
        errorLabel.isHidden = true
        view.addSubview(errorLabel)

        emptyLabel.textColor = .secondaryLabel
        emptyLabel.textAlignment = .center
        emptyLabel.text = "No records found"
        emptyLabel.isHidden = true
        view.addSubview(emptyLabel)

        tableView.snp.makeConstraints { make in make.edges.equalToSuperview() }
        activityIndicator.snp.makeConstraints { make in make.center.equalToSuperview() }
        errorLabel.snp.makeConstraints { make in make.center.equalToSuperview(); make.leading.trailing.equalToSuperview().inset(32) }
        emptyLabel.snp.makeConstraints { make in make.center.equalToSuperview() }
    }

    private func loadData() {
        activityIndicator.startAnimating()
        tableView.isHidden = true
        errorLabel.isHidden = true
        emptyLabel.isHidden = true

        Task { [weak self] in
            guard let self = self else { return }
            do {
                let result = try await self.config.load()
                self.records = result
                self.activityIndicator.stopAnimating()
                if result.isEmpty {
                    self.emptyLabel.isHidden = false
                } else {
                    self.tableView.isHidden = false
                    self.tableView.reloadData()
                }
            } catch {
                self.activityIndicator.stopAnimating()
                self.errorLabel.text = "Error: \(error.localizedDescription)"
                self.errorLabel.isHidden = false
            }
        }
    }

    fileprivate func reload() { loadData() }

    fileprivate func showToast(_ message: String) {
        let alert = UIAlertController(title: nil, message: message, preferredStyle: .alert)
        present(alert, animated: true)
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) { alert.dismiss(animated: true) }
    }
}

extension DomainListViewController: UITableViewDelegate, UITableViewDataSource {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        records.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "DomainCell", for: indexPath)
        let item = records[indexPath.row]
        var content = cell.defaultContentConfiguration()
        content.text = config.rowTitle(item)
        content.secondaryText = config.rowSubtitle(item)
        cell.contentConfiguration = content

        var buttons: [UIContextualAction] = []
        if let label = config.secondaryTitle, let action = config.secondaryAction {
            buttons.append(UIContextualAction(style: .normal, title: label) { [weak self] _, _, done in
                guard let self = self else { done(false); return }
                Task {
                    await action(item, self)
                    done(true)
                }
            })
        }
        if let label = config.primaryTitle, let action = config.primaryAction {
            buttons.append(UIContextualAction(style: .destructive, title: label) { [weak self] _, _, done in
                guard let self = self else { done(false); return }
                Task {
                    await action(item, self)
                    done(true)
                }
            })
        }
        if !buttons.isEmpty {
            let cfg = UISwipeActionsConfiguration(actions: buttons)
            cfg.performsFirstActionWithFullSwipe = false
            cell.trailingSwipeActionsConfiguration = cfg
            cell.accessoryType = .disclosureIndicator
        }
        return cell
    }
}

// MARK: - Concrete Domain View Controllers

private func subtitle(_ parts: [String?]) -> String {
    parts.compactMap { $0 }.filter { !$0.isEmpty }.joined(separator: " · ")
}

func toggle(_ status: String?) -> String {
    (status?.lowercased() == "active") ? "paused" : "active"
}

final class FuturesViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Futures",
            load: { try await AdminAPIService().listFutures() },
            primaryTitle: "Toggle",
            primaryAction: { item, vc in
                let next = toggle(item.status)
                _ = try? await vc.api.setFuturesStatus(id: item.id, status: next)
                vc.showToast("Status set to \(next)"); vc.reload()
            },
            secondaryTitle: nil, secondaryAction: nil,
            rowTitle: { $0.name ?? $0.symbol ?? "Futures #\($0.id)" },
            rowSubtitle: { subtitle([$0.symbol, $0.leverage.map { "\($0)x" }, $0.margin]) },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class OptionsViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Options",
            load: { try await AdminAPIService().listOptions() },
            primaryTitle: "Toggle",
            primaryAction: { item, vc in
                let next = toggle(item.status)
                _ = try? await vc.api.setOptionsStatus(id: item.id, status: next)
                vc.showToast("Status set to \(next)"); vc.reload()
            },
            secondaryTitle: nil, secondaryAction: nil,
            rowTitle: { $0.name ?? $0.symbol ?? "Options #\($0.id)" },
            rowSubtitle: { subtitle([$0.symbol, $0.strike, $0.expiry]) },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class CopyTradingViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Copy Trading",
            load: { try await AdminAPIService().listCopyTrading() },
            primaryTitle: "Toggle",
            primaryAction: { item, vc in
                let next = toggle(item.status)
                _ = try? await vc.api.setCopyTradingStatus(id: item.id, status: next)
                vc.showToast("Status set to \(next)"); vc.reload()
            },
            secondaryTitle: nil, secondaryAction: nil,
            rowTitle: { $0.name ?? "Strategy #\($0.id)" },
            rowSubtitle: { subtitle([$0.trader, $0.followers.map { "\($0) followers" }]) },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class ConvertViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Convert",
            load: { try await AdminAPIService().listConvert() },
            primaryTitle: "Toggle",
            primaryAction: { item, vc in
                let next = toggle(item.status)
                _ = try? await vc.api.setConvertStatus(id: item.id, status: next)
                vc.showToast("Status set to \(next)"); vc.reload()
            },
            secondaryTitle: nil, secondaryAction: nil,
            rowTitle: { "\($0.fromAsset ?? "?") → \($0.toAsset ?? "?")" },
            rowSubtitle: { subtitle([$0.amount, $0.rate.map { "rate \($0)" }]) },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class OnRampViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "On-Ramp",
            load: { try await AdminAPIService().listOnRamp() },
            primaryTitle: "Approve",
            primaryAction: { item, vc in
                _ = try? await vc.api.approveOnRamp(id: item.id)
                vc.showToast("Approved"); vc.reload()
            },
            secondaryTitle: "Reject",
            secondaryAction: { item, vc in
                _ = try? await vc.api.rejectOnRamp(id: item.id, reason: "Rejected by admin")
                vc.showToast("Rejected"); vc.reload()
            },
            rowTitle: { "\($0.assetDisplay) \($0.amount ?? "")" },
            rowSubtitle: { subtitle([$0.user, $0.provider]) },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class OffRampViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Off-Ramp",
            load: { try await AdminAPIService().listOffRamp() },
            primaryTitle: "Approve",
            primaryAction: { item, vc in
                _ = try? await vc.api.approveOffRamp(id: item.id)
                vc.showToast("Approved"); vc.reload()
            },
            secondaryTitle: "Reject",
            secondaryAction: { item, vc in
                _ = try? await vc.api.rejectOffRamp(id: item.id, reason: "Rejected by admin")
                vc.showToast("Rejected"); vc.reload()
            },
            rowTitle: { "\($0.assetDisplay) \($0.amount ?? "")" },
            rowSubtitle: { subtitle([$0.user, $0.provider]) },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class P2PClientsViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "P2P Clients",
            load: { try await AdminAPIService().listP2PClients() },
            primaryTitle: "Toggle",
            primaryAction: { item, vc in
                let next = (item.status?.lowercased() == "active") ? "suspended" : "active"
                _ = try? await vc.api.setP2PClientStatus(id: item.id, status: next)
                vc.showToast("Status set to \(next)"); vc.reload()
            },
            secondaryTitle: nil, secondaryAction: nil,
            rowTitle: { $0.name ?? "Client #\($0.id)" },
            rowSubtitle: { $0.email ?? "" },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class P2PMerchantsViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "P2P Merchants",
            load: { try await AdminAPIService().listP2PMerchants() },
            primaryTitle: "Approve",
            primaryAction: { item, vc in
                _ = try? await vc.api.approveP2PMerchant(id: item.id)
                vc.showToast("Approved"); vc.reload()
            },
            secondaryTitle: "Reject",
            secondaryAction: { item, vc in
                _ = try? await vc.api.rejectP2PMerchant(id: item.id, reason: "Rejected by admin")
                vc.showToast("Rejected"); vc.reload()
            },
            rowTitle: { $0.name ?? "Merchant #\($0.id)" },
            rowSubtitle: { subtitle([$0.email, $0.verified.map { $0 ? "verified" : "unverified" }]) },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class PartnersViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Partners",
            load: { try await AdminAPIService().listPartners() },
            primaryTitle: "Approve",
            primaryAction: { item, vc in
                _ = try? await vc.api.approvePartner(id: item.id)
                vc.showToast("Approved"); vc.reload()
            },
            secondaryTitle: "Reject",
            secondaryAction: { item, vc in
                _ = try? await vc.api.rejectPartner(id: item.id, reason: "Rejected by admin")
                vc.showToast("Rejected"); vc.reload()
            },
            rowTitle: { $0.name ?? "Partner #\($0.id)" },
            rowSubtitle: { $0.type ?? "" },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class RewardsViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Rewards",
            load: { try await AdminAPIService().listRewards() },
            primaryTitle: "Toggle",
            primaryAction: { item, vc in
                let next = toggle(item.status)
                _ = try? await vc.api.setRewardStatus(id: item.id, status: next)
                vc.showToast("Status set to \(next)"); vc.reload()
            },
            secondaryTitle: nil, secondaryAction: nil,
            rowTitle: { $0.name ?? "Reward #\($0.id)" },
            rowSubtitle: { subtitle([$0.type, $0.amount]) },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class MarketingViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Marketing",
            load: { try await AdminAPIService().listMarketing() },
            primaryTitle: "Toggle",
            primaryAction: { item, vc in
                let next = toggle(item.status)
                _ = try? await vc.api.setMarketingStatus(id: item.id, status: next)
                vc.showToast("Status set to \(next)"); vc.reload()
            },
            secondaryTitle: nil, secondaryAction: nil,
            rowTitle: { $0.name ?? "Campaign #\($0.id)" },
            rowSubtitle: { $0.campaign ?? "" },
            rowStatus: { $0.status }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class RolesViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Roles",
            load: { try await AdminAPIService().listRoles() },
            primaryTitle: nil, primaryAction: nil,
            secondaryTitle: nil, secondaryAction: nil,
            rowTitle: { $0.name ?? "Role #\($0.id)" },
            rowSubtitle: { subtitle([$0.description, $0.permissions?.joined(separator: ", ").map { "perms: \($0)" }]) },
            rowStatus: { nil }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

final class PermissionsViewController: DomainListViewController {
    init() {
        super.init(config: Config(
            title: "Permissions",
            load: { try await AdminAPIService().listPermissions() },
            primaryTitle: nil, primaryAction: nil,
            secondaryTitle: nil, secondaryAction: nil,
            rowTitle: { $0.name ?? "Permission #\($0.id)" },
            rowSubtitle: { subtitle([$0.resource, $0.action, $0.description]) },
            rowStatus: { nil }
        ))
    }
    required init?(coder: NSCoder) { fatalError() }
}

// Convenience: onramp/offramp often key the asset under "asset" rather than symbol.
extension DomainRecord {
    var assetDisplay: String { symbol ?? name ?? "Asset" }
}
