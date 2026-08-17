// swift-tools-version: 5.9
// TigerMasterWallet — iOS SwiftUI master-wallet application.
//
// Xcode on macOS opens this Swift Package directly and builds the
// `TigerMasterWallet` executable target. `swift build` on Linux will resolve
// the manifest and dependencies but will fail to find UIKit/SwiftUI (Apple
// frameworks), which is expected — those modules only exist on Apple platforms.

import PackageDescription

let package = Package(
    name: "TigerMasterWallet",
    platforms: [
        .iOS(.v16),
        .macOS(.v13),
    ],
    products: [
        .executable(
            name: "TigerMasterWallet",
            targets: ["TigerMasterWallet"]
        ),
    ],
    dependencies: [
        // Starscream is imported by WebSocketService.swift (WebSocket,
        // WebSocketDelegate, WebSocketEvent, WebSocketClient). The sources
        // require it to compile, so it is declared here even though every
        // other module used is an Apple system framework.
        .package(url: "https://github.com/daltoniam/Starscream.git", from: "4.0.6"),
    ],
    targets: [
        .executableTarget(
            name: "TigerMasterWallet",
            dependencies: [
                .product(name: "Starscream", package: "Starscream"),
            ],
            path: "TigerMasterWallet"
        ),
    ]
)
