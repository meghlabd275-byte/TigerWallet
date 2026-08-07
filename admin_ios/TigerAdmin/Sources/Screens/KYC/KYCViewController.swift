/**
 * TigerWallet Admin - KYC Screen
 */

import UIKit
import SnapKit

class KYCViewController: UIViewController, UITableViewDelegate, UITableViewDataSource {
    
    private let segmentedControl = UISegmentedControl(items: ["Pending", "Approved", "Rejected"])
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private var kycRequests: [KycRequest] = []
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
        loadData()
    }
    
    private func setupUI() {
        title = "KYC Verification"
        view.backgroundColor = .systemBackground
        navigationController?.navigationBar.prefersLargeTitles = true
        
        segmentedControl.selectedSegmentIndex = 0
        segmentedControl.addTarget(self, action: #selector(segmentChanged), for: .valueChanged)
        
        tableView.delegate = self
        tableView.dataSource = self
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "KYCCell")
        
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
        kycRequests = (1...15).map { index in
            KycRequest(
                id: UUID().uuidString,
                userId: UUID().uuidString,
                level: index % 3 + 1,
                documentType: "Passport",
                status: index % 3 == 0 ? .pending : (index % 3 == 1 ? .approved : .rejected),
                firstName: "User",
                lastName: "\(index)",
                country: "US",
                createdAt: Date()
            )
        }
        tableView.reloadData()
    }
    
    @objc private func segmentChanged() {
        loadData()
    }
    
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        return kycRequests.count
    }
    
    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = tableView.dequeueReusableCell(withIdentifier: "KYCCell", for: indexPath)
        let request = kycRequests[indexPath.row]
        
        var content = cell.defaultContentConfiguration()
        content.text = "\(request.firstName) \(request.lastName)"
        content.secondaryText = "Level \(request.level) - \(request.country)"
        content.image = UIImage(systemName: "person.text.rectangle")
        
        cell.contentConfiguration = content
        
        if request.status == .pending {
            cell.accessoryType = .disclosureIndicator
        } else {
            cell.accessoryType = .none
        }
        
        return cell
    }
    
    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let request = kycRequests[indexPath.row]
        
        if request.status == .pending {
            showActionSheet(for: request)
        }
    }
    
    private func showActionSheet(for request: KycRequest) {
        let alert = UIAlertController(title: "KYC Action", message: "Choose action for \(request.firstName)", preferredStyle: .actionSheet)
        
        alert.addAction(UIAlertAction(title: "Approve", style: .default) { _ in
            // Approve KYC
        })
        
        alert.addAction(UIAlertAction(title: "Reject", style: .destructive) { _ in
            // Reject KYC
        })
        
        alert.addAction(UIAlertAction(title: "Cancel", style: .cancel))
        
        present(alert, animated: true)
    }
}
