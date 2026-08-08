/**
 * TigerWallet Admin - Dashboard Screen
 * Complete Dark/Light Theme Support
 */

import UIKit
import SnapKit

class DashboardViewController: UIViewController {
    
    private let scrollView = UIScrollView()
    private let contentView = UIView()
    
    private let welcomeCard: UIView = {
        let view = UIView()
        view.backgroundColor = ThemeManager.primaryColor
        view.layer.cornerRadius = 16
        return view
    }()
    
    private let statsGrid: UIStackView = {
        let stack = UIStackView()
        stack.axis = .vertical
        stack.spacing = 12
        stack.distribution = .fillEqually
        return stack
    }()
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
        loadData()
    }
    
    private func setupUI() {
        title = "Dashboard"
        view.backgroundColor = .systemBackground
        navigationController?.navigationBar.prefersLargeTitles = true
        
        view.addSubview(scrollView)
        scrollView.addSubview(contentView)
        
        scrollView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
        }
        
        contentView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }
        
        setupWelcomeCard()
        setupStatsGrid()
    }
    
    private func setupWelcomeCard() {
        contentView.addSubview(welcomeCard)
        
        let titleLabel = UILabel()
        titleLabel.text = "Welcome, Admin!"
        titleLabel.font = .systemFont(ofSize: 20, weight: .bold)
        titleLabel.textColor = .white
        
        let subtitleLabel = UILabel()
        subtitleLabel.text = "Here is what is happening today"
        subtitleLabel.font = .systemFont(ofSize: 14)
        subtitleLabel.textColor = UIColor.white.withAlphaComponent(0.8)
        
        let iconView = UIImageView(image: UIImage(systemName: "chart.bar.fill"))
        iconView.tintColor = .white
        
        welcomeCard.addSubview(titleLabel)
        welcomeCard.addSubview(subtitleLabel)
        welcomeCard.addSubview(iconView)
        
        welcomeCard.snp.makeConstraints { make in
            make.top.equalToSuperview().offset(16)
            make.leading.trailing.equalToSuperview().inset(16)
            make.height.equalTo(120)
        }
        
        titleLabel.snp.makeConstraints { make in
            make.top.leading.equalToSuperview().offset(20)
        }
        
        subtitleLabel.snp.makeConstraints { make in
            make.top.equalTo(titleLabel.snp.bottom).offset(8)
            make.leading.equalToSuperview().offset(20)
        }
        
        iconView.snp.makeConstraints { make in
            make.trailing.bottom.equalToSuperview().inset(20)
            make.width.height.equalTo(40)
        }
    }
    
    private func setupStatsGrid() {
        contentView.addSubview(statsGrid)
        
        let stats = [
            ("Total Users", "12,543", "person.fill"),
            ("Active Users", "8,921", "person.badge.clock"),
            ("Transactions", "456,789", "arrow.left.arrow.right"),
            ("Volume", "$12.5M", "dollarsign.circle.fill"),
            ("Pending KYC", "145", "checkmark.shield"),
            ("Fees", "$234K", "creditcard.fill")
        ]
        
        let topRow = UIStackView()
        topRow.axis = .horizontal
        topRow.spacing = 12
        topRow.distribution = .fillEqually
        
        let bottomRow = UIStackView()
        bottomRow.axis = .horizontal
        bottomRow.spacing = 12
        bottomRow.distribution = .fillEqually
        
        for (index, stat) in stats.enumerated() {
            let card = createStatCard(title: stat.0, value: stat.1, icon: stat.2)
            if index < 3 {
                topRow.addArrangedSubview(card)
            } else {
                bottomRow.addArrangedSubview(card)
            }
        }
        
        statsGrid.addArrangedSubview(topRow)
        statsGrid.addArrangedSubview(bottomRow)
        
        statsGrid.snp.makeConstraints { make in
            make.top.equalTo(welcomeCard.snp.bottom).offset(24)
            make.leading.trailing.equalToSuperview().inset(16)
            make.bottom.equalToSuperview().offset(-24)
        }
    }
    
    private func createStatCard(title: String, value: String, icon: String) -> UIView {
        let card = UIView()
        card.backgroundColor = .secondarySystemBackground
        card.layer.cornerRadius = 12
        
        let iconView = UIImageView(image: UIImage(systemName: icon))
        iconView.tintColor = ThemeManager.primaryColor
        
        let valueLabel = UILabel()
        valueLabel.text = value
        valueLabel.font = .systemFont(ofSize: 20, weight: .bold)
        
        let titleLabel = UILabel()
        titleLabel.text = title
        titleLabel.font = .systemFont(ofSize: 12)
        titleLabel.textColor = .secondaryLabel
        
        card.addSubview(iconView)
        card.addSubview(valueLabel)
        card.addSubview(titleLabel)
        
        iconView.snp.makeConstraints { make in
            make.top.leading.equalToSuperview().offset(12)
            make.width.height.equalTo(24)
        }
        
        valueLabel.snp.makeConstraints { make in
            make.leading.equalToSuperview().offset(12)
            make.bottom.equalTo(titleLabel.snp.top).offset(-2)
        }
        
        titleLabel.snp.makeConstraints { make in
            make.leading.bottom.equalToSuperview().inset(12)
        }
        
        card.snp.makeConstraints { make in
            make.height.equalTo(100)
        }
        
        return card
    }
    
    private func loadData() {
        // Fetch dashboard data from API
    }
}
