/**
 * TigerWallet Admin - Crypto Cards Screen (iOS)
 * Complete implementation with dark/light theme support
 */

import UIKit
import SnapKit

class CryptoCardsViewController: UIViewController, UITableViewDelegate, UITableViewDataSource {
    
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private var cards: [CryptoCard] = []
    private var filteredCards: [CryptoCard] = []
    private var currentFilter = "all"
    private let searchController = UISearchController(searchResultsController: nil)
    
    struct CryptoCard {
        let id: String
        let userName: String
        let cardNumber: String
        let currency: String
        let balance: Double
        let limit: Double
        let status: String
        let cardType: String
    }
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
        loadData()
    }
    
    private func setupUI() {
        title = "Crypto Cards"
        view.backgroundColor = .systemBackground
        navigationController?.navigationBar.prefersLargeTitles = true
        
        // Theme toggle
        let themeButton = UIBarButtonItem(image: UIImage(systemName: ThemeManager.isDark ? "sun.max" : "moon"), style: .plain, target: self, action: #selector(toggleTheme))
        navigationItem.rightBarButtonItem = themeButton
        
        // Search
        searchController.searchResultsUpdater = self
        searchController.obscuresBackgroundDuringPresentation = false
        searchController.searchBar.placeholder = "Search cards..."
        navigationItem.searchController = searchController
        definesPresentationContext = true
        
        // Table
        tableView.delegate = self
        tableView.dataSource = self
        tableView.register(CryptoCardCell.self, forCellReuseIdentifier: "CryptoCardCell")
        
        // Segmented control
        let segmentedControl = UISegmentedControl(items: ["All", "Active", "Blocked", "Pending"])
        segmentedControl.selectedSegmentIndex = 0
        segmentedControl.addTarget(self, action: #selector(filterChanged(_:)), for: .valueChanged)
        
        view.addSubview(segmentedControl)
        view.addSubview(tableView)
        
        segmentedControl.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(8)
            make.leading.trailing.equalToSuperview().inset(16)
        }
        
        tableView.snp.makeConstraints { make in
            make.top.equalTo(segmentedControl.snp.bottom).offset(8)
            make.leading.trailing.bottom.equalToSuperview()
        }
    }
    
    private func loadData() {
        // Mock data - in production, call API
        cards = [
            CryptoCard(id: "1", userName: "John Doe", cardNumber: "4532123456789012", currency: "USDT", balance: 5000, limit: 10000, status: "active", cardType: "virtual"),
            CryptoCard(id: "2", userName: "Jane Smith", cardNumber: "4532987654321098", currency: "USDT", balance: 2500, limit: 5000, status: "blocked", cardType: "physical"),
            CryptoCard(id: "3", userName: "Bob Wilson", cardNumber: "4532567890123456", currency: "USDT", balance: 1000, limit: 2000, status: "pending", cardType: "virtual")
        ]
        applyFilter()
    }
    
    @objc private func filterChanged(_ sender: UISegmentedControl) {
        let filters = ["all", "active", "blocked", "pending"]
        currentFilter = filters[sender.selectedSegmentIndex]
        applyFilter()
    }
    
    private func applyFilter() {
        if currentFilter == "all" {
            filteredCards = cards
        } else {
            filteredCards = cards.filter { $0.status == currentFilter }
        }
        tableView.reloadData()
    }
    
    @objc private func toggleTheme() {
        ThemeManager.isDark.toggle()
        view.window?.overrideUserInterfaceStyle = ThemeManager.isDark ? .dark : .light
        navigationItem.rightBarButtonItem?.image = UIImage(systemName: ThemeManager.isDark ? "sun.max" : "moon")
    }
    
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        return filteredCards.count
    }
    
    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "CryptoCardCell", for: indexPath) as! CryptoCardCell
        cell.configure(with: filteredCards[indexPath.row])
        return cell
    }
    
    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
    }
    
    func tableView(_ tableView: UITableView, trailingSwipeActionsConfigurationForRowAt indexPath: IndexPath) -> UISwipeActionsConfiguration? {
        let card = filteredCards[indexPath.row]
        
        if card.status == "active" {
            let block = UIContextualAction(style: .destructive, title: "Block") { _, _, completion in
                completion(true)
            }
            block.backgroundColor = .systemRed
            return UISwipeActionsConfiguration(actions: [block])
        } else {
            let activate = UIContextualAction(style: .normal, title: "Activate") { _, _, completion in
                completion(true)
            }
            activate.backgroundColor = .systemGreen
            return UISwipeActionsConfiguration(actions: [activate])
        }
    }
}

extension CryptoCardsViewController: UISearchResultsUpdating {
    func updateSearchResults(for searchController: UISearchController) {
        guard let searchText = searchController.searchBar.text, !searchText.isEmpty else {
            applyFilter()
            return
        }
        filteredCards = cards.filter { $0.userName.lowercased().contains(searchText.lowercased()) }
        tableView.reloadData()
    }
}

class CryptoCardCell: UITableViewCell {
    private let cardNumberLabel = UILabel()
    private let userLabel = UILabel()
    private let balanceLabel = UILabel()
    private let statusBadge = UILabel()
    private let typeIcon = UIImageView()
    
    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        setupUI()
    }
    
    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }
    
    private func setupUI() {
        typeIcon.contentMode = .scaleAspectFit
        cardNumberLabel.font = .systemFont(ofSize: 16, weight: .semibold)
        userLabel.font = .systemFont(ofSize: 14)
        userLabel.textColor = .secondaryLabel
        balanceLabel.font = .systemFont(ofSize: 14)
        statusBadge.font = .systemFont(ofSize: 12, weight: .medium)
        statusBadge.textAlignment = .center
        statusBadge.layer.cornerRadius = 4
        statusBadge.clipsToBounds = true
        
        contentView.addSubview(typeIcon)
        contentView.addSubview(cardNumberLabel)
        contentView.addSubview(userLabel)
        contentView.addSubview(balanceLabel)
        contentView.addSubview(statusBadge)
        
        typeIcon.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(16)
            make.centerY.equalToSuperview()
            make.width.height.equalTo(40)
        }
        
        cardNumberLabel.snp.makeConstraints { make in
            make.top.equalToSuperview().offset(12)
            make.leading.equalTo(typeIcon.snp.trailing).offset(12)
        }
        
        userLabel.snp.makeConstraints { make in
            make.top.equalTo(cardNumberLabel.snp.bottom).offset(4)
            make.leading.equalTo(cardNumberLabel)
        }
        
        balanceLabel.snp.makeConstraints { make in
            make.top.equalTo(userLabel.snp.bottom).offset(4)
            make.leading.equalTo(cardNumberLabel)
            make.bottom.equalToSuperview().offset(-12)
        }
        
        statusBadge.snp.makeConstraints { make in
            make.trailing.equalToSuperview().offset(-16)
            make.centerY.equalToSuperview()
            make.width.equalTo(70)
            make.height.equalTo(24)
        }
    }
    
    func configure(with card: CryptoCardsViewController.CryptoCard) {
        cardNumberLabel.text = "•••• \(card.cardNumber.suffix(4))"
        userLabel.text = "\(card.userName) - \(card.currency) \(card.balance)"
        balanceLabel.text = "Limit: \(card.currency) \(card.limit)"
        
        statusBadge.text = card.status.capitalized
        statusBadge.textColor = .white
        
        switch card.status {
        case "active":
            statusBadge.backgroundColor = .systemGreen
            typeIcon.image = UIImage(systemName: "creditcard.fill")
            typeIcon.tintColor = .systemGreen
        case "blocked":
            statusBadge.backgroundColor = .systemRed
            typeIcon.image = UIImage(systemName: "xmark.circle.fill")
            typeIcon.tintColor = .systemRed
        default:
            statusBadge.backgroundColor = .systemOrange
            typeIcon.image = UIImage(systemName: "clock.fill")
            typeIcon.tintColor = .systemOrange
        }
    }
}
