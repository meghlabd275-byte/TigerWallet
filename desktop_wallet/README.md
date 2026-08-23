# TigerWallet Desktop App Configuration

## Overview

TigerWallet Desktop supports Windows, macOS, and Linux with full functionality.

## Technology Stack

- **Framework**: Flutter Desktop
- **Backend**: Go microservices
- **Database**: PostgreSQL + Redis

## Windows Configuration

### System Requirements

- **OS**: Windows 10 or later (64-bit)
- **RAM**: 4GB minimum, 8GB recommended
- **Storage**: 500MB available space
- **Display**: 1280x720 minimum

### Installer Configuration (NSIS)

```nsis
; installer.nsi
!include "MUI2.nsh"

Name "TigerWallet"
OutFile "TigerWallet-Setup-1.0.0.exe"
InstallDir "$PROGRAMFILES64\TigerWallet"
InstallDirRegKey HKLM "Software\TigerWallet" "InstallDir"

RequestExecutionLevel admin

!define MUI_ICON "assets\icon.ico"
!define MUI_UNICON "assets\icon.ico"

!define MUI_WELCOMEPAGE_TITLE "Welcome to TigerWallet Setup"
!define MUI_WELCOMEPAGE_TEXT "This will install TigerWallet on your computer."

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
    SetOutPath "$INSTDIR"
    File "app\TigerWallet.exe"
    
    ; Create shortcuts
    CreateDirectory "$SMPROGRAMS\TigerWallet"
    CreateShortCut "$SMPROGRAMS\TigerWallet\TigerWallet.lnk" "$INSTDIR\TigerWallet.exe"
    CreateShortCut "$DESKTOP\TigerWallet.lnk" "$INSTDIR\TigerWallet.exe"
    
    ; Registry entries
    WriteRegStr HKLM "Software\TigerWallet" "InstallDir" "$INSTDIR"
    WriteRegStr HKLM "Software\TigerWallet" "Version" "1.0.0"
    
    ; Uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TigerWallet" "DisplayName" "TigerWallet"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TigerWallet" "UninstallString" "$INSTDIR\Uninstall.exe"
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\TigerWallet.exe"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir "$INSTDIR"
    Delete "$SMPROGRAMS\TigerWallet\TigerWallet.lnk"
    RMDir "$SMPROGRAMS\TigerWallet"
    Delete "$DESKTOP\TigerWallet.lnk"
    DeleteRegKey HKLM "Software\TigerWallet"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\TigerWallet"
SectionEnd
```

### MSIX Package

```xml
<!-- AppxManifest.xml -->
<?xml version="1.0" encoding="utf-8"?>
<Package
  xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
  xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
  xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities">
  
  <Identity
    Name="TigerWallet.Desktop"
    Publisher="CN=TigerWallet"
    Version="1.0.0.0"
    ProcessorArchitecture="x64"/>
    
  <Properties>
    <DisplayName>TigerWallet</DisplayName>
    <PublisherDisplayName>TigerWallet</PublisherDisplayName>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>
  
  <Resources>
    <Resource Language="en-us"/>
  </Resources>
  
  <Applications>
    <Application Id="App" Executable="TigerWallet.exe" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements
        DisplayName="TigerWallet"
        Description="Enterprise Web3 Wallet"
        BackgroundColor="#000000"
        Square150x150Logo="Assets\Square150x150Logo.png"
        Square44x44Logo="Assets\Square44x44Logo.png">
        <uap:DefaultTile Wide310x150Logo="Assets\Wide310x150Logo.png"/>
      </uap:VisualElements>
    </Application>
  </Applications>
  
  <Capabilities>
    <rescap:Capability Name="runFullTrust"/>
    <Capability Name="internetClient"/>
  </Capabilities>
</Package>
```

## macOS Configuration

### System Requirements

- **OS**: macOS 11 (Big Sur) or later
- **RAM**: 4GB minimum, 8GB recommended
- **Storage**: 500MB available space
- **Chip**: Apple Silicon (M1/M2) or Intel

### Info.plist

```xml
<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>TigerWallet</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.tigerwallet.desktop</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>TigerWallet</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>NSMainStoryboardFile</key>
    <string>Main</string>
    <key>NSPrincipalClass</key>
    <string>NSApplication</string>
    <key>NSHumanReadableCopyright</key>
    <string>Copyright © 2024 TigerWallet. All rights reserved.</string>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.finance</string>
    <key>NSFaceIDUsageDescription</key>
    <string>TigerWallet uses Face ID to secure your wallet.</string>
</dict>
</plist>
```

### Entitlements

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.app-sandbox</key>
    <false/>
    <key>com.apple.security.network.client</key>
    <true/>
    <key>com.apple.security.files.user-selected.read-write</key>
    <true/>
</dict>
</plist>
```

### DMG Configuration

```ruby
# TigerWallet.dmg
title = "TigerWallet"
volume_icon = "Assets/AppIcon.icns"
background = "Assets/background.png"
window_position = { :x => 100, :y => 100 }
window_size = { :width => 500, :height => 400 }

packages = [
  "TigerWallet.app"
]

# Shortcuts
dmg_dir = "TigerWallet-1.0.0"
Dir.mkdir(dmg_dir)
FileUtils.cp_r("TigerWallet.app", dmg_dir)
```

## Linux Configuration

### System Requirements

- **OS**: Ubuntu 20.04+, Fedora 36+, Debian 11+
- **RAM**: 4GB minimum
- **Storage**: 500MB
- **Display**: 1280x720

### AppImage Configuration

```yaml
# AppImage.yml
appId: com.tigerwallet.desktop
runtime: org.freedesktop.Platform
runtimeVersion: '23.08'
sdk: org.freedesktop.Sdk
command: tigerwallet

sections:
  - name: TigerWallet
    path: TigerWallet/
    type: binary

# Desktop entry
desktop:
  exec: tigerwallet
  name: TigerWallet
  comment: Enterprise Web3 Wallet
  categories: Network;Finance;
  keywords: wallet;crypto;ethereum;bitcoin;
  icon: assets/icon.png
```

### Flatpak Configuration

```xml
<?xml version="1.0" encoding="UTF-8"?>
<package version="1.0">
  <id>com.tigerwallet.desktop</id>
  <runtime>org.freedesktop.Platform</runtime>
  <runtime-version>23.08</runtime-version>
  <sdk>org.freedesktop.Sdk</sdk>
  <command>tigerwallet</command>
  
  <name>TigerWallet</name>
  <summary>Enterprise Web3 Wallet</summary>
  <description>Trade, swap, stake across 100+ chains</description>
  
  <categories>
    <category>Network</category>
    <category>Finance</category>
  </categories>
  
  <content type="binary">files/</content>
  
  <launchable type="desktop-id">tigerwallet.desktop</launchable>
  
  <provides>
    <binary>tigerwallet</binary>
  </provides>
</package>
```

### Desktop Entry

```desktop
# tigerwallet.desktop
[Desktop Entry]
Name=TigerWallet
Comment=Enterprise Web3 Wallet
Exec=tigerwallet %U
Icon=tigerwallet
Terminal=false
Type=Application
Categories=Network;Finance;Cryptocurrency;
Keywords=wallet;crypto;ethereum;bitcoin;blockchain;defi;
StartupWMClass=tigerwallet
MimeType=x-scheme-handler/tigerwallet;x-scheme-handler/wc;
```

## Flutter Desktop Implementation

### pubspec.yaml

```yaml
name: tigerwallet_desktop
description: TigerWallet Desktop Application
version: 1.0.0
publish_to: 'https://tigerwallet.io'

environment:
  sdk: '>=3.0.0 <4.0.0'

dependencies:
  flutter:
    sdk: flutter
  flutter_hooks: ^0.20.0
  hooks_riverpod: ^2.4.0
  window_manager: ^0.3.7
  flutter_secure_storage: ^9.0.0
  http: ^1.1.0
  shared_preferences: ^2.2.0
  intl: ^0.18.0
  url_launcher: ^6.2.0
  window_manager: ^0.3.7
  
dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^3.0.0

flutter:
  uses-material-design: true
  
  assets:
    - assets/icons/
    - assets/images/
```

### Window Manager Setup

```dart
import 'package:flutter/material.dart';
import 'package:window_manager/window_manager.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  await windowManager.ensureInitialized();
  
  WindowOptions windowOptions = const WindowOptions(
    size: Size(1400, 900),
    minimumSize: Size(1024, 768),
    center: true,
    backgroundColor: Colors.transparent,
    skipTaskbar: false,
    titleBarStyle: TitleBarStyle.normal,
    title: 'TigerWallet',
  );
  
  await windowManager.waitUntilReadyToShow(windowOptions, () async {
    await windowManager.show();
    await windowManager.focus();
  });
  
  runApp(const TigerWalletApp());
}
```

## Auto-Update Configuration

### Update Server

```go
// update_server.go
package main

import (
    "encoding/json"
    "net/http"
)

type UpdateInfo struct {
    Version     string `json:"version"`
    ReleaseDate string `json:"releaseDate"`
    DownloadURL struct {
        Windows string `json:"windows"`
        macOS   string `json:"macos"`
        Linux   string `json:"linux"`
    } `json:"downloadURL"`
    ReleaseNotes string `json:"releaseNotes"`
    MinVersion  string `json:"minVersion"`
}

func getUpdateInfo(w http.ResponseWriter, r *http.Request) {
    update := UpdateInfo{
        Version:     "1.0.0",
        ReleaseDate: "2024-01-01",
        DownloadURL: struct {
            Windows string `json:"windows"`
            macOS   string `json:"macos"`
            Linux   string `json:"linux"`
        }{
            Windows: "https://releases.tigerwallet.io/v1.0.0/TigerWallet-Setup.exe",
            macOS:   "https://releases.tigerwallet.io/v1.0.0/TigerWallet.dmg",
            Linux:   "https://releases.tigerwallet.io/v1.0.0/TigerWallet.AppImage",
        },
        ReleaseNotes: "- Bug fixes\n- Performance improvements\n- New features",
        MinVersion:  "1.0.0",
    }
    
    json.NewEncoder(w).Encode(update)
}

func main() {
    http.HandleFunc("/api/v1/update", getUpdateInfo)
    http.ListenAndServe(":8080", nil)
}
```

## GitHub Actions - Desktop Builds

```yaml
name: Desktop CI/CD

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    strategy:
      matrix:
        include:
          - os: windows-latest
            artifact: app-windows
          - os: macos-latest
            artifact: app-macos
          - os: ubuntu-latest
            artifact: app-linux

    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Flutter
        uses: subosito/flutter-action@v2
        with:
          flutter-version: '3.24.0'
          
      - name: Install dependencies
        run: flutter pub get
        
      - name: Build
        run: |
          flutter config --enable-linux-desktop
          flutter build ${{ matrix.artifact }}
          
      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: ${{ matrix.artifact }}
          path: build/${{ matrix.artifact }}/

  release:
    needs: build
    runs-on: ubuntu-latest
    
    steps:
      - name: Download artifacts
        uses: actions/download-artifact@v3
        
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            app-windows/*.exe
            app-macos/*.dmg
            app-linux/*.AppImage
          draft: true
```

## Platform-Specific Features

### Windows Features
- NSIS installer
- MSIX package
- Start menu integration
- System tray
- Windows Hello integration

### macOS Features
- DMG distribution
- App Store ready
- Apple Silicon optimized
- Touch Bar support
- Keychain integration

### Linux Features
- AppImage (portable)
- Flatpak
- DEB/RPM packages
- System tray
- Desktop notifications
