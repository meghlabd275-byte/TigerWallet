/**
 * TigerWallet Admin - More Screen
 */

import UIKit
import SnapKit

class MoreViewController: UIViewController, UITableViewDelegate, UITableViewDataSource {
    
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    
    private let sections: [(title: String, items: [(icon: String, name: String)])] = [
        ("Trading", [
            ("dollarsign.circle", "Tokens"),
            ("arrow.left.arrow.right", "Trading Pairs"),
            ("link", "Blockchains")
        ]),
        ("Management", [
            ("building.2", "White Labels"),
            ("questionmark.circle", "Support Tickets")
        ]),
        ("Analytics", [
            ("chart.bar", "Analytics")
        ]),
        ("Settings", [
            ("gear", "Settings"),
            ("info.circle", "About")
        ])
    ]
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
    }
    
    private func setupUI() {
        title = "More"
        view.backgroundColor = .systemBackground
        navigationController?.navigationBar.prefersLargeTitles = true
        
        tableView.delegate = self
        tableView.dataSource = self
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "MoreCell")
        
        view.addSubview(tableView)
        tableView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
        }
    }
    
    func numberOfSections(in tableView: UITableView) -> Int {
        return sections.count
    }
    
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        return sections[section].items.count
    }
    
    func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        return sections[section].title
    }
    
    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "MoreCell", for: indexPath)
        let item = sections[indexPath.section].items[indexPath.row]
        
        var content = cell.defaultContentConfiguration()
        content.text = item.name
        content.image = UIImage(systemName: item.icon)
        content.imageProperties.tintColor = ThemeManager.primaryColor
        
        cell.contentConfiguration = content
        cell.accessoryType = .disclosureIndicator
        return cell
    }
    
    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        
        let item = sections[indexPath.section].items[indexPath.row]
        var targetVC: UIViewController?
        
        switch item.name {
        case "Tokens":
            targetVC = TokensListViewController()
        case "Trading Pairs":
            targetVC = PairsListViewController()
        case "Blockchains":
            targetVC = BlockchainsListViewController()
        case "White Labels":
            targetVC = WhiteLabelsListViewController()
        case "Support Tickets":
            targetVC = TicketsListViewController()
        case "Analytics":
            targetVC = AnalyticsListViewController()
        case "Settings":
            targetVC = SettingsListViewController()
        case "About":
            targetVC = AboutViewController()
        default:
            break
        }
        
        if let vc = targetVC {
            navigationController?.pushViewController(vc, animated: true)
        }
    }
}

// Placeholder ViewControllers
class TokensListViewController: UIViewController {
    override func viewDidLoad() { super.viewDidLoad(); title = "Tokens"; view.backgroundColor = .systemBackground }
}

class PairsListViewController: UIViewController {
    override func viewDidLoad() { super.viewDidLoad(); title = "Trading Pairs"; view.backgroundColor = .systemBackground }
}

class BlockchainsListViewController: UIViewController {
    override func viewDidLoad() { super.viewDidLoad(); title = "Blockchains"; view.backgroundColor = .systemBackground }
}

class WhiteLabelsListViewController: UIViewController {
    override func viewDidLoad() { super.viewDidLoad(); title = "White Labels"; view.backgroundColor = .systemBackground }
}

class TicketsListViewController: UIViewController {
    override func viewDidLoad() { super.viewDidLoad(); title = "Support Tickets"; view.backgroundColor = .systemBackground }
}

class AnalyticsListViewController: UIViewController {
    override func viewDidLoad() { super.viewDidLoad(); title = "Analytics"; view.backgroundColor = .systemBackground }
}

class SettingsListViewController: UIViewController {
    override func viewDidLoad() { super.viewDidLoad(); title = "Settings"; view.backgroundColor = .systemBackground }
}

class AboutViewController: UIViewController {
    override func viewDidLoad() { super.viewDidLoad(); title = "About"; view.backgroundColor = .systemBackground }
}
