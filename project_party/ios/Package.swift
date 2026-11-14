// swift-tools-version: 5.9
// TigerProjectParty — iOS SwiftUI application for the standalone ProjectParty
// token-listing / launchpad backend.
//
// Xcode on macOS opens this Swift Package directly and builds the
// `TigerProjectParty` executable target. `swift build` on Linux resolves the
// manifest but cannot find SwiftUI (an Apple-only framework), which is
// expected.
//
// The app has no third-party dependencies: every network call is URLSession.

import PackageDescription

let package = Package(
    name: "TigerProjectParty",
    platforms: [
        .iOS(.v16),
        .macOS(.v13),
    ],
    products: [
        .executable(
            name: "TigerProjectParty",
            targets: ["TigerProjectParty"]
        ),
    ],
    targets: [
        .executableTarget(
            name: "TigerProjectParty",
            path: "App"
        ),
    ]
)
