// swift-tools-version: 5.9
// TigerBots — iOS SwiftUI application for the standalone Bots backend.
//
// Xcode on macOS opens this Swift Package directly and builds the
// `TigerBots` executable target. `swift build` on Linux resolves the manifest
// but cannot find SwiftUI/UIKit (Apple-only frameworks), which is expected.
//
// The live feed uses Foundation's URLSessionWebSocketTask, so no third-party
// socket dependency is required.

import PackageDescription

let package = Package(
    name: "TigerBots",
    platforms: [
        .iOS(.v16),
        .macOS(.v13),
    ],
    products: [
        .executable(
            name: "TigerBots",
            targets: ["TigerBots"]
        ),
    ],
    targets: [
        .executableTarget(
            name: "TigerBots",
            path: "App"
        ),
    ]
)
