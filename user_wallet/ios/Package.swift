// swift-tools-version: 5.9
// TigerUserWallet — iOS SwiftUI user-wallet application.
//
// The app previously shipped 50 Swift sources with no Xcode project and no
// package manifest, so nothing could build it. This manifest is the build
// entry point: Xcode opens the package directly, and `swift build` resolves
// the manifest on Linux (compilation itself needs the Apple frameworks).
//
// The WebSocket live feed uses Foundation's URLSessionWebSocketTask, so no
// third-party socket dependency is required. Google Drive backup is the one
// external dependency: GoogleDriveBackup.swift imports GTMAppAuth, AppAuth
// and GTLRDrive.

import PackageDescription

let package = Package(
    name: "TigerUserWallet",
    platforms: [
        .iOS(.v16),
        .macOS(.v13),
    ],
    products: [
        .executable(
            name: "TigerUserWallet",
            targets: ["TigerUserWallet"]
        ),
    ],
    dependencies: [
        .package(url: "https://github.com/google/GoogleSignIn-iOS.git", from: "7.1.0"),
        // GTMAppAuth is the SPM-capable sibling of gtm-oauth2; the latter ships
        // no Package.swift, so it cannot be a Swift Package dependency.
        .package(url: "https://github.com/google/GTMAppAuth.git", from: "4.1.0"),
        .package(url: "https://github.com/google/google-api-objectivec-client-for-rest.git", from: "3.3.0"),
    ],
    targets: [
        .executableTarget(
            name: "TigerUserWallet",
            dependencies: [
                .product(name: "GoogleSignIn", package: "GoogleSignIn-iOS"),
                .product(name: "GTMAppAuth", package: "GTMAppAuth"),
                // The SPM product is named GoogleAPIClientForREST_Drive; the
                // module it exposes is GTLRDrive, which the sources import.
                .product(name: "GoogleAPIClientForREST_Drive", package: "google-api-objectivec-client-for-rest"),
            ],
            path: "App"
        ),
    ]
)
