/**
 * TigerWallet Admin - Margin Trading Screen (iOS)
 */

import UIKit
import SnapKit

class MarginTradingViewController: UIViewController, UITableViewDelegate, UITableViewDataSource {
    
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private var positions: [MarginPosition] = []
    private var filteredPositions: [MarginPosition] = []
    private var currentFilter = "all"
    private let statsView = UIStackView()
    
    struct MarginPosition {
        let id: String
        let userName: String
        let pair: String
        let side: String
        let size: Double
        let leverage: Int
        let entryPrice: Double
        let currentPrice: Double
        let pnl: Double
        let liquidationPrice: Double
        let status: String
    }
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
        loadData()
    }
    
    private func setupUI() {
        title = "Margin Trading"
        view.backgroundColor = .systemBackground
        navigationController?.navigationBar.prefersLargeTitles = true
        
        let themeButton = UIBarButtonItem(image: UIImage(systemName: ThemeManager.isDark ? "sun.max" : "moon"), style: .plain, target: self, action: #selector(toggleTheme))
        navigationItem.rightBarButtonItem = themeButton
        
        // Stats
        statsView.axis = .horizontal
        statsView.distribution = .fillEqually
        statsView.spacing = 8
        
        let stats = [("Positions", "150"), ("Volume", "$5M"), ("Liq.", "3"), ("Liq. Vol", "$50K")]
        for (label, value) in stats {
            let statView = createStatView(label: label, value: value)
            statsView.addArrangedSubview(statView)
        }
        
        // Segmented control
        let segmentedControl = UISegmentedControl(items: ["All", "Open", "Liquidated"])
        segmentedControl.selectedSegmentIndex = 0
        segmentedControl.addTarget(self, action: #selector(filterChanged(_:)), for: .valueChanged)
        
        tableView.delegate = self
        tableView.dataSource = self
        tableView.register(MarginPositionCell.self, forCellReuseIdentifier: "MarginPositionCell")
        
        view.addSubview(statsView)
        view.addSubview(segmentedControl)
        view.addSubview(tableView)
        
        statsView.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(8)
            make.leading.trailing.equalToSuperview().inset(16)
            make.height.equalTo(80)
        }
        
        segmentedControl.snp.makeConstraints { make in
            make.top.equalTo(statsView.snp.bottom).offset(8)
            make.leading.trailing.equalToSuperview().inset(16)
        }
        
        tableView.snp.makeConstraints { make in
            make.top.equalTo(segmentedControl.snp.bottom).offset(8)
            make.leading.trailing.bottom.equalToSuperview()
        }
    }
    
    private func createStatView(label: String, value: String) -> UIView {
        let view = UIView()
        view.backgroundColor = .secondarySystemBackground
        view.layer.cornerRadius = 8
        
        let valueLabel = UILabel()
        valueLabel.text = value
        valueLabel.font = .systemFont(ofSize: 18, weight: .bold)
        valueLabel.textAlignment = .center
        
        let titleLabel = UILabel()
        titleLabel.text = label
        titleLabel.font = .systemFont(ofSize: 12)
        titleLabel.textColor = .secondaryLabel
        titleLabel.textAlignment = .center
        
        view.addSubview(valueLabel)
        view.addSubview(titleLabel)
        
        valueLabel.snp.makeConstraints { make in
            make.top.equalToSuperview().offset(12)
            make.centerX.equalToSuperview()
        }
        
        titleLabel.snp.makeConstraints { make in
            make.top.equalTo(valueLabel.snp.bottom).offset(4)
            make.centerX.equalToSuperview()
        }
        
        return view
    }
    
    private func loadData() {
        positions = [
            MarginPosition(id: "1", userName: "Trader John", pair: "BTC/USDT", side: "long", size: 1.5, leverage: 10, entryPrice: 45000, currentPrice: 47000, pnl: 3000, liquidationPrice: 40500, status: "open"),
            MarginPosition(id: "2", userName: "Trader Jane", pair: "ETH/USDT", side: "short", size: 10, leverage: 5, entryPrice: 3000, currentPrice: 2800, pnl: 2000, liquidationPrice: 3600, status: "open")
        ]
        applyFilter()
    }
    
    @objc private func filterChanged(_ sender: UISegmentedControl) {
        let filters = ["all", "open", "liquidated"]
        currentFilter = filters[sender.selectedSegmentIndex]
        applyFilter()
    }
    
    private func applyFilter() {
        if currentFilter == "all" {
            filteredPositions = positions
        } else {
            filteredPositions = positions.filter { $0.status == currentFilter }
        }
        tableView.reloadData()
    }
    
    @objc private func toggleTheme() {
        ThemeManager.isDark.toggle()
        view.window?.overrideUserInterfaceStyle = ThemeManager.isDark ? .dark : .light
        navigationItem.rightBarButtonItem?.image = UIImage(systemName: ThemeManager.isDark ? "sun.max" : "moon")
    }
    
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        return filteredPositions.count
    }
    
    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "MarginPositionCell", for: indexPath) as! MarginPositionCell
        cell.configure(with: filteredPositions[indexPath.row])
        return cell
    }
    
    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
    }
}

class MarginPositionCell: UITableViewCell {
    private let pairLabel = UILabel()
    private let userLabel = UILabel()
    private let sideBadge = UILabel()
    private let pnlLabel = UILabel()
    private let detailsLabel = UILabel()
    private let liqPriceLabel = UILabel()
    
    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        setupUI()
    }
    
    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }
    
    private func setupUI() {
        pairLabel.font = .systemFont(ofSize: 16, weight: .bold)
        userLabel.font = .systemFont(ofSize: 14)
        userLabel.textColor = .secondaryLabel
        sideBadge.font = .systemFont(ofSize: 12, weight: .medium)
        sideBadge.textAlignment = .center
        sideBadge.layer.cornerRadius = 4
        sideBadge.clipsToBounds = true
        pnlLabel.font = .systemFont(ofSize: 16, weight: .semibold)
        detailsLabel.font = .systemFont(ofSize: 12)
        detailsLabel.textColor = .secondaryLabel
        liqPriceLabel.font = .systemFont(ofSize: 12)
        liqPriceLabel.textColor = .systemOrange
        
        contentView.addSubview(pairLabel)
        contentView.addSubview(sideBadge)
        contentView.addSubview(userLabel)
        contentView.addSubview(pnlLabel)
        contentView.addSubview(detailsLabel)
        contentView.addSubview(liqPriceLabel)
        
        pairLabel.snp.makeConstraints { make in
            make.top.equalToSuperview().offset(12)
            make.leading.equalToSuperview().offset(16)
        }
        
        sideBadge.snp.makeConstraints { make in
            make.centerY.equalTo(pairLabel)
            make.leading.equalTo(pairLabel.snp.trailing).offset(8)
            make.width.equalTo(50)
            make.height.equalTo(20)
        }
        
        userLabel.snp.makeConstraints { make in
            make.top.equalTo(pairLabel.snp.bottom).offset(4)
            make.leading.equalTo(pairLabel)
        }
        
        pnlLabel.snp.makeConstraints { make in
            make.top.equalToSuperview().offset(12)
            make.trailing.equalToSuperview().offset(-16)
        }
        
        detailsLabel.snp.makeConstraints { make in
            make.top.equalTo(userLabel.snp.bottom).offset(4)
            make.leading.equalTo(pairLabel)
            make.bottom.equalToSuperview().offset(-12)
        }
        
        liqPriceLabel.snp.makeConstraints { make in
            make.centerY.equalTo(detailsLabel)
            make.trailing.equalToSuperview().offset(-16)
        }
    }
    
    func configure(with pos: MarginTradingViewController.MarginPosition) {
        pairLabel.text = "\(pos.pair) (\(pos.leverage)x)"
        userLabel.text = pos.userName
        sideBadge.text = pos.side.uppercased()
        
        if pos.side == "long" {
            sideBadge.backgroundColor = .systemGreen
            sideBadge.textColor = .white
        } else {
            sideBadge.backgroundColor = .systemRed
            sideBadge.textColor = .white
        }
        
        pnlLabel.text = "$\(Int(pos.pnl))"
        pnlLabel.textColor = pos.pnl >= 0 ? .systemGreen : .systemRed
        
        detailsLabel.text = "Size: \(pos.size) | Entry: $\(Int(pos.entryPrice)) | Current: $\(Int(pos.currentPrice))"
        liqPriceLabel.text = "Liq: $\(Int(pos.liquidationPrice))"
    }
}
