/**
 * TigerWallet Admin - Login Screen
 * Complete Dark/Light Theme Support
 */

import UIKit
import SnapKit

class LoginViewController: UIViewController {
    
    var onLoginSuccess: (() -> Void)?
    
    private let logoImageView: UIImageView = {
        let iv = UIImageView()
        iv.image = UIImage(systemName: "adminpanelsettings")
        iv.tintColor = ThemeManager.primaryColor
        iv.contentMode = .scaleAspectFit
        return iv
    }()
    
    private let titleLabel: UILabel = {
        let label = UILabel()
        label.text = "Welcome Back"
        label.font = .systemFont(ofSize: 28, weight: .bold)
        label.textAlignment = .center
        return label
    }()
    
    private let subtitleLabel: UILabel = {
        let label = UILabel()
        label.text = "Sign in to continue to TigerAdmin"
        label.font = .systemFont(ofSize: 16)
        label.textColor = .secondaryLabel
        label.textAlignment = .center
        return label
    }()
    
    private let emailTextField: UITextField = {
        let tf = UITextField()
        tf.placeholder = "Email Address"
        tf.borderStyle = .roundedRect
        tf.keyboardType = .emailAddress
        tf.autocapitalizationType = .none
        tf.leftView = UIView(frame: CGRect(x: 0, y: 0, width: 16, height: 0))
        tf.leftViewMode = .always
        return tf
    }()
    
    private let passwordTextField: UITextField = {
        let tf = UITextField()
        tf.placeholder = "Password"
        tf.isSecureTextEntry = true
        tf.borderStyle = .roundedRect
        tf.leftView = UIView(frame: CGRect(x: 0, y: 0, width: 16, height: 0))
        tf.leftViewMode = .always
        return tf
    }()
    
    private let loginButton: UIButton = {
        let btn = UIButton(type: .system)
        btn.setTitle("Sign In", for: .normal)
        btn.setTitleColor(.white, for: .normal)
        btn.backgroundColor = ThemeManager.primaryColor
        btn.titleLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
        btn.layer.cornerRadius = 12
        return btn
    }()
    
    private let themeToggleButton: UIButton = {
        let btn = UIButton(type: .system)
        btn.setImage(UIImage(systemName: "moon.fill"), for: .normal)
        return btn
    }()
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
        setupActions()
    }
    
    private func setupUI() {
        view.backgroundColor = .systemBackground
        
        view.addSubview(logoImageView)
        view.addSubview(titleLabel)
        view.addSubview(subtitleLabel)
        view.addSubview(emailTextField)
        view.addSubview(passwordTextField)
        view.addSubview(loginButton)
        view.addSubview(themeToggleButton)
        
        logoImageView.snp.makeConstraints { make in
            make.top.equalTo(view.safeAreaLayoutGuide).offset(60)
            make.centerX.equalToSuperview()
            make.width.height.equalTo(80)
        }
        
        titleLabel.snp.makeConstraints { make in
            make.top.equalTo(logoImageView.snp.bottom).offset(24)
            make.leading.trailing.equalToSuperview().inset(24)
        }
        
        subtitleLabel.snp.makeConstraints { make in
            make.top.equalTo(titleLabel.snp.bottom).offset(8)
            make.leading.trailing.equalToSuperview().inset(24)
        }
        
        emailTextField.snp.makeConstraints { make in
            make.top.equalTo(subtitleLabel.snp.bottom).offset(40)
            make.leading.trailing.equalToSuperview().inset(24)
            make.height.equalTo(50)
        }
        
        passwordTextField.snp.makeConstraints { make in
            make.top.equalTo(emailTextField.snp.bottom).offset(16)
            make.leading.trailing.equalToSuperview().inset(24)
            make.height.equalTo(50)
        }
        
        loginButton.snp.makeConstraints { make in
            make.top.equalTo(passwordTextField.snp.bottom).offset(24)
            make.leading.trailing.equalToSuperview().inset(24)
            make.height.equalTo(56)
        }
        
        themeToggleButton.snp.makeConstraints { make in
            make.top.equalTo(loginButton.snp.bottom).offset(32)
            make.centerX.equalToSuperview()
        }
    }
    
    private func setupActions() {
        loginButton.addTarget(self, action: #selector(loginTapped), for: .touchUpInside)
        themeToggleButton.addTarget(self, action: #selector(toggleTheme), for: .touchUpInside)
    }
    
    @objc private func loginTapped() {
        guard let email = emailTextField.text, !email.isEmpty,
              let password = passwordTextField.text, !password.isEmpty else {
            return
        }
        
        loginButton.isEnabled = false
        loginButton.setTitle("Loading...", for: .normal)
        
        AuthService.shared.login(email: email, password: password) { [weak self] result in
            DispatchQueue.main.async {
                self?.loginButton.isEnabled = true
                self?.loginButton.setTitle("Sign In", for: .normal)
                
                switch result {
                case .success:
                    self?.onLoginSuccess?()
                case .failure(let error):
                    print("Login failed: \(error)")
                }
            }
        }
    }
    
    @objc private func toggleTheme() {
        let currentTheme = ThemeManager.shared.currentTheme
        switch currentTheme {
        case .system:
            ThemeManager.shared.currentTheme = .light
            themeToggleButton.setImage(UIImage(systemName: "sun.max.fill"), for: .normal)
        case .light:
            ThemeManager.shared.currentTheme = .dark
            themeToggleButton.setImage(UIImage(systemName: "moon.fill"), for: .normal)
        case .dark:
            ThemeManager.shared.currentTheme = .system
            themeToggleButton.setImage(UIImage(systemName: "circle.lefthalf.filled"), for: .normal)
        }
    }
}
