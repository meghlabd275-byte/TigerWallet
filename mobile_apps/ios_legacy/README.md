# TigerWallet iOS App Configuration

## App Store Information

- **App Name**: TigerWallet
- **Bundle Identifier**: com.tigerwallet.ios
- **Version**: 1.0.0
- **Build**: 1
- **Category**: Finance
- **Content Rating**: 4+

## Required Capabilities

- [x] WalletKit
- [x] Push Notifications
- [x] Background Modes (Remote notifications)
- [x] Face ID Usage
- [x] Keychain Sharing

## Info.plist Configuration

```xml
<!-- Info.plist -->
<key>CFBundleDisplayName</key>
<string>TigerWallet</string>
<key>CFBundleShortVersionString</key>
<string>1.0.0</string>
<key>CFBundleVersion</key>
<string>1</string>
<key>UILaunchStoryboardName</key>
<string>LaunchScreen</string>
<key>UIRequiredDeviceCapabilities</key>
<array>
    <string>armv7</string>
</array>
<key>UISupportedInterfaceOrientations</key>
<array>
    <string>UIInterfaceOrientationPortrait</string>
</array>
<key>NSFaceIDUsageDescription</key>
<string>TigerWallet uses Face ID to secure your wallet and authorize transactions.</string>
<key>ITSAppUsesNonExemptEncryption</key>
<false/>
<key>BGTaskSchedulerPermittedIdentifiers</key>
<array>
    <string>com.tigerwallet.ios.refresh</string>
    <string>com.tigerwallet.ios.notifications</string>
</array>
```

## Entitlements

```xml
<!-- TigerWallet.entitlements -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.developer.associated-domains</key>
    <array>
        <string>applinks:tigerwallet.io</string>
        <string>wc:tigerwallet</string>
    </array>
    <key>com.apple.security.application-groups</key>
    <array>
        <string>group.com.tigerwallet.ios</string>
    </array>
    <key>keychain-access-groups</key>
    <array>
        <string>$(AppIdentifierPrefix)com.tigerwallet.ios</string>
    </array>
    <key>aps-environment</key>
    <string>development</string>
</dict>
</plist>
```

## Push Notification Setup

```swift
// AppDelegate.swift
import UserNotifications

func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
    // Request push notification permissions
    UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge]) { granted, error in
        if granted {
            DispatchQueue.main.async {
                application.registerForRemoteNotifications()
            }
        }
    }
    
    // Configure notification categories
    let viewAction = UNNotificationAction(identifier: "VIEW_ACTION", title: "View", options: .foreground)
    let dismissAction = UNNotificationAction(identifier: "DISMISS_ACTION", title: "Dismiss", options: .destructive)
    
    let transactionCategory = UNNotificationCategory(identifier: "TRANSACTION", actions: [viewAction, dismissAction], intentIdentifiers: [], options: [])
    
    UNUserNotificationCenter.current().setNotificationCategories([transactionCategory])
    
    return true
}

func application(_ application: UIApplication, didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
    let token = deviceToken.map { String(format: "%02.2hhx", $0) }.joined()
    print("Device Token: \(token)")
    // Send token to backend
}
```

## WalletConnect Configuration

```swift
// WalletConnectService.swift
import WalletConnectSwift

class WalletConnectService {
    static let shared = WalletConnectService()
    
    private let client: Client
    
    private let metadata = AppMetadata(
        name: "TigerWallet",
        description: "Enterprise Web3 Wallet",
        url: "https://tigerwallet.io",
        icons: ["https://tigerwallet.io/icon.png"]
    )
    
    func initialize() {
        let projectId = "YOUR_WALLETCONNECT_PROJECT_ID"
        client = Client(metadata: metadata, projectId: projectId)
        
        client.delegate = self
    }
    
    func connect(uri: String) {
        guard let walletUri = WCURI(string: uri) else { return }
        try? client.connect(to: walletUri)
    }
}
```

## Deep Links

```swift
// DeepLinkHandler.swift
enum DeepLink {
    case walletconnect(String)
    case transaction(String)
    case token(String)
    case web3(String)
    
    init?(url: URL) {
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: true) else {
            return nil
        }
        
        switch components.host {
        case "wc":
            if let query = components.queryItems?.first(where: { $0.name == "uri" })?.value {
                self = .walletconnect(query)
            } else {
                return nil
            }
        case "tx":
            self = .transaction(components.path)
        case "token":
            self = .token(components.path)
        case "web3":
            self = .web3(components.path)
        default:
            return nil
        }
    }
}
```

## App Store Submission Checklist

- [ ] screenshots (6.5 inch and 5.5 inch)
- [ ] App icon (1024x1024)
- [ ] Privacy Policy URL
- [ ] Terms of Service URL
- [ ] Support URL
- [ ] Marketing text
- [ ] Description (4000 chars max)
- [ ] Build version uploaded
- [ ] Export Compliance (yes/no)
- [ ] Content Rights (yes/no/no)
- [ ] Advertising Identifier (no)

## TestFlight Configuration

```xml
<!-- TestFlight Groups -->
Group: Internal Testers
- All developers
- All QA team

Group: External Testers  
- Beta users
- Community members
```

## Fastlane Configuration

```ruby
# Fastfile
default_platform(:ios)

platform :ios do
  desc "Deploy a new version to the App Store"
  lane :release do
    increment_build_number(build_number: latest_testflight_build_number + 1)
    
    build_app(
      workspace: "TigerWallet.xcworkspace",
      scheme: "TigerWallet",
      configuration: "Release",
      export_method: "app-store"
    )
    
    upload_to_app_store(
      skip_metadata: true,
      skip_screenshots: true,
      force: true
    )
  end
  
  desc "Deploy a new beta build to TestFlight"
  lane :beta do
    increment_build_number
    
    build_app(
      workspace: "TigerWallet.xcworkspace",
      scheme: "TigerWallet",
      configuration: "Debug",
      export_method: "app-store"
    )
    
    upload_to_testflight(
      skip_submission: true,
      distribute_external: true
    )
  end
end
```

## CI/CD with GitHub Actions

```yaml
# ios-release.yml
name: iOS Release

on:
  push:
    branches:
      - main
    paths:
      - 'mobile/ios/**'

jobs:
  build:
    runs-on: macos-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Flutter
        uses: subosito/flutter-action@v2
        with:
          flutter-version: '3.24.0'
          
      - name: Install dependencies
        run: flutter pub get
        
      - name: Build iOS
        run: flutter build ios --release --no-codesign
        env:
          FLOOR_STORE: ${{ secrets.FLOOR_STORE }}
          
      - name: Upload Build
        uses: actions/upload-artifact@v3
        with:
          name: ios-build
          path: build/ios/iphoneos/Runner.app
```
