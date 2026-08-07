/**
 * TigerWallet Admin - Users Screen
 */

import UIKit
import SnapKit

class UsersViewController: UIViewController, UITableViewDelegate, UITableViewDataSource {
    
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let searchController = UISearchController(searchResultsController: nil)
    private var users: [User] = []
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
        loadData()
    }
    
    private func setupUI() {
        title = "Users"
        view.backgroundColor = .systemBackground
        navigationController?.navigationBar.prefersLargeTitles = true
        
        searchController.searchResultsUpdater = self
        searchController.obscuresBackgroundDuringPresentation = false
        searchController.searchBar.placeholder = "Search users..."
        navigationItem.searchController = searchController
        definesPresentationContext = true
        
        tableView.delegate = self
        tableView.dataSource = self
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "UserCell")
        
        view.addSubview(tableView)
        tableView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
        }
    }
    
    private func loadData() {
        // Simulated data
        users = (1...20).map { index in
            User(
                id: UUID().uuidString,
                email: "user\(index)@example.com",
                username: "User \(index)",
                role: .admin,
                status: index % 4 == 0 ? .active : (index % 4 == 1 ? .suspended : .banned),
                kycLevel: index % 3 + 1,
                createdAt: Date()
            )
        }
        tableView.reloadData()
    }
    
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        return users.count
    }
    
    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "UserCell", for: indexPath)
        let user = users[indexPath.row]
        
        var content = cell.defaultContentConfiguration()
        content.text = user.username
        content.secondaryText = user.email
        content.image = UIImage(systemName: "person.circle.fill")
        content.imageProperties.tintColor = ThemeManager.primaryColor
        
        cell.contentConfiguration = content
        cell.accessoryType = .disclosureIndicator
        return cell
    }
    
    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let user = users[indexPath.row]
        let detailVC = UserDetailViewController(user: user)
        navigationController?.pushViewController(detailVC, animated: true)
    }
}

// User Detail Screen
class UserDetailViewController: UIViewController {
    private let user: User
    
    init(user: User) {
        self.user = user
        super.init(nibName: nil, bundle: nil)
    }
    
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
    }
    
    private func setupUI() {
        title = "User Details"
        view.backgroundColor = .systemBackground
        
        let scrollView = UIScrollView()
        view.addSubview(scrollView)
        scrollView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
        }
        
        let contentView = UIView()
        scrollView.addSubview(contentView)
        contentView.snp.makeConstraints { make in
            make.edges.equalToSuperview()
            make.width.equalToSuperview()
        }
        
        let stackView = UIStackView()
        stackView.axis = .vertical
        stackView.spacing = 16
        contentView.addSubview(stackView)
        stackView.snp.makeConstraints { make in
            make.edges.equalToSuperview().inset(16)
        }
        
        let fields = [
            ("ID", user.id),
            ("Email", user.email),
            ("Username", user.username),
            ("Role", user.role.rawValue),
            ("Status", user.status.rawValue),
            ("KYC Level", "\(user.kycLevel)")
        ]
        
        for (label, value) in fields {
            let row = createField(label: label, value: value)
            stackView.addArrangedSubview(row)
        }
        
        let buttonStack = UIStackView()
        buttonStack.axis = .horizontal
        buttonStack.spacing = 12
        buttonStack.distribution = .fillEqually
        
        let banButton = UIButton(type: .system)
        banButton.setTitle("Ban", for: .normal)
        banButton.setTitleColor(.white, for: .normal)
        banButton.backgroundColor = ThemeManager.errorColor
        banButton.layer.cornerRadius = 8
        
        let suspendButton = UIButton(type: .system)
        suspendButton.setTitle("Suspend", for: .normal)
        suspendButton.setTitleColor(.white, for: .normal)
        suspendButton.backgroundColor = ThemeManager.warningColor
        suspendButton.layer.cornerRadius = 8
        
        buttonStack.addArrangedSubview(banButton)
        buttonStack.addArrangedSubview(suspendButton)
        
        stackView.addArrangedSubview(buttonStack)
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

extension UsersViewController: UISearchResultsUpdating {
    func updateSearchResults(for searchController: UISearchController) {
        // Filter users
    }
}
