/**
 * TigerWallet Admin - Transactions Screen
 */

import UIKit
import SnapKit

class TransactionsViewController: UIViewController, UITableViewDelegate, UITableViewDataSource {
    
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private var transactions: [Transaction] = []
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
        loadData()
    }
    
    private func setupUI() {
        title = "Transactions"
        view.backgroundColor = .systemBackground
        navigationController?.navigationBar.prefersLargeTitles = true
        
        tableView.delegate = self
        tableView.dataSource = self
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "TransactionCell")
        
        view.addSubview(tableView)
        tableView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
        }
    }
    
    private func loadData() {
        let types: [Transaction.TransactionType] = [.deposit, .withdraw, .transfer, .swap]
        let statuses: [Transaction.TransactionStatus] = [.confirmed, .pending, .failed, .flagged]
        
        transactions = (1...30).map { index in
            Transaction(
                id: UUID().uuidString,
                userId: UUID().uuidString,
                type: types[index % 4],
                amount: "\(index * 100)",
                currency: "USDT",
                status: statuses[index % 4],
                fromAddress: "0xabcd...1234",
                toAddress: "0xefgh...5678",
                txHash: "0x\(String(index, radix: 16))",
                timestamp: Date()
            )
        }
        tableView.reloadData()
    }
    
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        return transactions.count
    }
    
    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "TransactionCell", for: indexPath)
        let tx = transactions[indexPath.row]
        
        var content = cell.defaultContentConfiguration()
        content.text = "\(tx.type.rawValue.capitalized) - $\(tx.amount)"
        content.secondaryText = "Hash: \(tx.txHash ?? "N/A")"
        
        let iconName: String
        switch tx.type {
        case .deposit: iconName = "arrow.down.circle.fill"
        case .withdraw: iconName = "arrow.up.circle.fill"
        case .transfer: iconName = "arrow.left.arrow.right.circle.fill"
        case .swap: iconName = "arrow.triangle.2.circlepath.circle.fill"
        }
        content.image = UIImage(systemName: iconName)
        
        cell.contentConfiguration = content
        cell.accessoryType = .disclosureIndicator
        return cell
    }
    
    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let tx = transactions[indexPath.row]
        let detailVC = TransactionDetailViewController(transaction: tx)
        navigationController?.pushViewController(detailVC, animated: true)
    }
}

class TransactionDetailViewController: UIViewController {
    private let transaction: Transaction
    
    init(transaction: Transaction) {
        self.transaction = transaction
        super.init(nibName: nil, bundle: nil)
    }
    
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
    
    override func viewDidLoad() {
        super.viewDidLoad()
        title = "Transaction Details"
        view.backgroundColor = .systemBackground
        
        let scrollView = UIScrollView()
        view.addSubview(scrollView)
        scrollView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
        }
        
        let stackView = UIStackView()
        stackView.axis = .vertical
        stackView.spacing = 16
        stackView.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 16, right: 16)
        stackView.isLayoutMarginsRelativeArrangement = true
        
        scrollView.addSubview(stackView)
        stackView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }
        
        let fields = [
            ("ID", transaction.id),
            ("Type", transaction.type.rawValue),
            ("Amount", "\(transaction.amount) \(transaction.currency)"),
            ("Status", transaction.status.rawValue),
            ("From", transaction.fromAddress ?? "N/A"),
            ("To", transaction.toAddress ?? "N/A"),
            ("Hash", transaction.txHash ?? "N/A")
        ]
        
        for (label, value) in fields {
            let row = createField(label: label, value: value)
            stackView.addArrangedSubview(row)
        }
        
        let flagButton = UIButton(type: .system)
        flagButton.setTitle("Flag Transaction", for: .normal)
        flagButton.setTitleColor(.white, for: .normal)
        flagButton.backgroundColor = ThemeManager.errorColor
        flagButton.layer.cornerRadius = 8
        stackView.addArrangedSubview(flagButton)
    }
    
    private func createField(label: String, value: String) -> UIView {
        let container = UIView()
        container.backgroundColor = .secondarySystemBackground
        container.layer.cornerRadius = 8
        
        let labelView = UILabel()
        labelView.text = label
        labelView.textColor = .secondaryLabel
        labelView.font = .systemFont(ofSize: 14)
        
        let valueView = UILabel()
        valueView.text = value
        valueView.font = .systemFont(ofSize: 16, weight: .semibold)
        
        container.addSubview(labelView)
        container.addSubview(valueView)
        
        labelView.snp.makeConstraints { make in
            make.top.leading.equalToSuperview().offset(12)
        }
        
        valueView.snp.makeConstraints { make in
            make.top.equalTo(labelView.snp.bottom).offset(4)
            make.leading.bottom.equalToSuperview().inset(12)
        }
        
        return container
    }
}
