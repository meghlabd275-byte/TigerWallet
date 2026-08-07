/**
 * TigerWallet Admin - App Coordinator
 */

import UIKit
import XCoordinator

class AppCoordinator: Coordinator {
    var childCoordinators: [Coordinator] = []
    var rootViewController: UIViewController
    
    private let window: UIWindow
    private var isLoggedIn = false
    
    init(window: UIWindow) {
        self.window = window
        self.rootViewController = UINavigationController()
    }
    
    func start() {
        // Check authentication status
        if AuthService.shared.isLoggedIn {
            showMain()
        } else {
            showLogin()
        }
        window.rootViewController = rootViewController
    }
    
    private func showLogin() {
        let loginVC = LoginViewController()
        loginVC.onLoginSuccess = { [weak self] in
            self?.showMain()
        }
        rootViewController.setViewControllers([loginVC], animated: false)
    }
    
    private func showMain() {
        let tabBarController = MainTabBarController()
        rootViewController.setViewControllers([tabBarController], animated: true)
    }
}

// Main Tab Bar Controller
class MainTabBarController: UITabBarController {
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupTabs()
    }
    
    private func setupTabs() {
        let dashboardVC = DashboardViewController()
        dashboardVC.tabBarItem = UITabBarItem(title: "Dashboard", image: UIImage(systemName: "house"), selectedImage: UIImage(systemName: "house.fill"))
        
        let usersVC = UsersViewController()
        usersVC.tabBarItem = UITabBarItem(title: "Users", image: UIImage(systemName: "person.2"), selectedImage: UIImage(systemName: "person.2.fill"))
        
        let kycVC = KYCViewController()
        kycVC.tabBarItem = UITabBarItem(title: "KYC", image: UIImage(systemName: "checkmark.shield"), selectedImage: UIImage(systemName: "checkmark.shield.fill"))
        
        let transactionsVC = TransactionsViewController()
        transactionsVC.tabBarItem = UITabBarItem(title: "Transactions", image: UIImage(systemName: "arrow.left.arrow.right"), selectedImage: UIImage(systemName: "arrow.left.arrow.right.circle.fill"))
        
        let moreVC = MoreViewController()
        moreVC.tabBarItem = UITabBarItem(title: "More", image: UIImage(systemName: "ellipsis"), selectedImage: UIImage(systemName: "ellipsis.circle.fill"))
        
        let dashboardNav = UINavigationController(rootViewController: dashboardVC)
        let usersNav = UINavigationController(rootViewController: usersVC)
        let kycNav = UINavigationController(rootViewController: kycVC)
        let transactionsNav = UINavigationController(rootViewController: transactionsVC)
        let moreNav = UINavigationController(rootViewController: moreVC)
        
        viewControllers = [dashboardNav, usersNav, kycNav, transactionsNav, moreNav]
    }
}
