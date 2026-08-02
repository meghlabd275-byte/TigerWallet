//
//  TigerWallet iOS - Push Notification Service
//  Production-ready push notifications using APNs
//

import Foundation
import UserNotifications
import UIKit

// MARK: - Push Notification Manager
@available(iOS 10.0, *)
class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
    
    static let shared = PushNotificationManager()
    
    private let notificationCenter = UNUserNotificationCenter.current()
    private var deviceToken: String?
    
    // Notification categories
    enum NotificationCategory: String {
        case transaction = "TRANSACTION"
        case security = "SECURITY"
        case priceAlert = "PRICE_ALERT"
        case general = "GENERAL"
    }
    
    private override init() {
        super.init()
        notificationCenter.delegate = self
    }
    
    // MARK: - Registration
    
    func requestAuthorization(completion: @escaping (Bool) -> Void) {
        notificationCenter.requestAuthorization(options: [.alert, .sound, .badge]) { granted, error in
            if granted {
                DispatchQueue.main.async {
                    UIApplication.shared.registerForRemoteNotifications()
                }
            }
            completion(granted)
        }
    }
    
    func registerDeviceToken(_ deviceToken: Data) {
        let tokenParts = deviceToken.map { String(format: "%02.2hhx", $0) }
        self.deviceToken = tokenParts.joined()
        
        // Send token to server
        sendTokenToServer(tokenParts.joined())
    }
    
    // MARK: - Setup Categories
    
    func setupNotificationCategories() {
        // Transaction actions
        let viewAction = UNNotificationAction(
            identifier: "VIEW_TRANSACTION",
            title: "View Transaction",
            options: .foreground
        )
        
        let transactionCategory = UNNotificationCategory(
            identifier: NotificationCategory.transaction.rawValue,
            actions: [viewAction],
            intentIdentifiers: [],
            options: []
        )
        
        // Security actions
        let secureAction = UNNotificationAction(
            identifier: "SECURE_ACCOUNT",
            title: "Secure Account",
            options: .foreground
        )
        
        let securityCategory = UNNotificationCategory(
            identifier: NotificationCategory.security.rawValue,
            actions: [secureAction],
            intentIdentifiers: [],
            options: []
        )
        
        // Price alert actions
        let viewPriceAction = UNNotificationAction(
            identifier: "VIEW_PRICE",
            title: "View Chart",
            options: .foreground
        )
        
        let priceCategory = UNNotificationCategory(
            identifier: NotificationCategory.priceAlert.rawValue,
            actions: [viewPriceAction],
            intentIdentifiers: [],
            options: []
        )
        
        notificationCenter.setNotificationCategories([
            transactionCategory,
            securityCategory,
            priceCategory
        ])
    }
    
    // MARK: - Handle Token
    
    private func sendTokenToServer(_ token: String) {
        guard let url = URL(string: "\(APIClient.shared.baseURL)/push/register") else { return }
        
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        let body: [String: Any] = [
            "token": token,
            "platform": "ios",
            "app_version": Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        ]
        
        request.httpBody = try? JSONSerialization.data(withJSONObject: body)
        
        URLSession.shared.dataTask(with: request) { _, response, error in
            if let error = error {
                print("Failed to register push token: \(error)")
            }
        }.resume()
    }
    
    // MARK: - UNUserNotificationCenterDelegate
    
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        // Show notification even when app is in foreground
        completionHandler([.banner, .sound, .badge])
    }
    
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        let userInfo = response.notification.request.content.userInfo
        
        switch response.actionIdentifier {
        case "VIEW_TRANSACTION":
            handleTransactionNotification(userInfo)
        case "SECURE_ACCOUNT":
            handleSecurityNotification(userInfo)
        case "VIEW_PRICE":
            handlePriceAlertNotification(userInfo)
        case UNNotificationDefaultActionIdentifier:
            handleDefaultAction(userInfo)
        default:
            break
        }
        
        completionHandler()
    }
    
    // MARK: - Handle Notifications
    
    private func handleTransactionNotification(_ userInfo: [AnyHashable: Any]) {
        guard let txHash = userInfo["tx_hash"] as? String else { return }
        
        NotificationCenter.default.post(
            name: .transactionNotificationReceived,
            object: nil,
            userInfo: ["tx_hash": txHash]
        )
    }
    
    private func handleSecurityNotification(_ userInfo: [AnyHashable: Any]) {
        NotificationCenter.default.post(
            name: .securityNotificationReceived,
            object: nil,
            userInfo: userInfo
        )
    }
    
    private func handlePriceAlertNotification(_ userInfo: [AnyHashable: Any]) {
        guard let symbol = userInfo["token_symbol"] as? String else { return }
        
        NotificationCenter.default.post(
            name: .priceAlertNotificationReceived,
            object: nil,
            userInfo: ["symbol": symbol]
        )
    }
    
    private func handleDefaultAction(_ userInfo: [AnyHashable: Any]) {
        if let type = userInfo["type"] as? String {
            switch type {
            case NotificationCategory.transaction.rawValue:
                handleTransactionNotification(userInfo)
            case NotificationCategory.security.rawValue:
                handleSecurityNotification(userInfo)
            case NotificationCategory.priceAlert.rawValue:
                handlePriceAlertNotification(userInfo)
            default:
                break
            }
        }
    }
}

// MARK: - Notification Names

extension Notification.Name {
    static let transactionNotificationReceived = Notification.Name("transactionNotificationReceived")
    static let securityNotificationReceived = Notification.Name("securityNotificationReceived")
    static let priceAlertNotificationReceived = Notification.Name("priceAlertNotificationReceived")
}

// MARK: - Biometric Authentication

enum BiometricType {
    case none
    case touchID
    case faceID
}

class BiometricAuthService {
    
    static let shared = BiometricAuthService()
    
    private let context = LAContext()
    
    var biometricType: BiometricType {
        var error: NSError?
        guard context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error) else {
            return .none
        }
        
        switch context.biometryType {
        case .touchID:
            return .touchID
        case .faceID:
            return .faceID
        case .opticID:
            return .faceID // Treat opticID similar to faceID
        default:
            return .none
        }
    }
    
    var isBiometricAvailable: Bool {
        var error: NSError?
        return context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error)
    }
    
    func authenticate(reason: String, completion: @escaping (Result<Bool, Error>) -> Void) {
        let newContext = LAContext()
        newContext.localizedCancelTitle = "Cancel"
        
        var error: NSError?
        guard newContext.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error) else {
            completion(.failure(error ?? BiometricError.notAvailable))
            return
        }
        
        newContext.evaluatePolicy(
            .deviceOwnerAuthenticationWithBiometrics,
            localizedReason: reason
        ) { success, error in
            DispatchQueue.main.async {
                if success {
                    completion(.success(true))
                } else if let error = error {
                    completion(.failure(error))
                } else {
                    completion(.failure(BiometricError.unknown))
                }
            }
        }
    }
    
    enum BiometricError: Error {
        case notAvailable
        case unknown
    }
}

// Import LocalAuthentication for biometric auth
import LocalAuthentication
