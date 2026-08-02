# TigerWallet Android App Configuration

## Google Play Store Information

- **App Name**: TigerWallet
- **Package Name**: com.tigerwallet.android
- **Version**: 1.0.0
- **Version Code**: 1
- **Category**: Finance
- **Content Rating**: Everyone

## Required Permissions

```xml
<!-- AndroidManifest.xml -->
<uses-permission android:name="android.permission.INTERNET"/>
<uses-permission android:name="android.permission.ACCESS_NETWORK_STATE"/>
<uses-permission android:name="android.permission.USE_BIOMETRIC"/>
<uses-permission android:name="android.permission.USE_FINGERPRINT"/>
<uses-permission android:name="android.permission.CAMERA"/>
<uses-permission android:name="android.permission.VIBRATE"/>
<uses-permission android:name="android.permission.RECEIVE_BOOT_COMPLETED"/>
<uses-permission android:name="android.permission.FOREGROUND_SERVICE"/>
<uses-permission android:name="android.permission.POST_NOTIFICATIONS"/>
<uses-permission android:name="android.permission.NFC"/>
<uses-permission android:name="android.permission.BLUETOOTH"/>
<uses-permission android:name="android.permission.BLUETOOTH_CONNECT"/>

<!-- Biometric Feature -->
<uses-feature android:name="android.hardware.fingerprint" android:required="false"/>
<uses-feature android:name="android.hardware.camera" android:required="false"/>
<uses-feature android:name="android.hardware.nfc" android:required="false"/>
```

## build.gradle (Module)

```groovy
plugins {
    id 'com.android.application'
    id 'org.jetbrains.kotlin.android'
    id 'kotlin-kapt'
    id 'com.google.gms.google-services'
}

android {
    namespace 'com.tigerwallet.android'
    compileSdk 34

    defaultConfig {
        applicationId "com.tigerwallet.android"
        minSdk 24
        targetSdk 34
        versionCode 1
        versionName "1.0.0"

        testInstrumentationRunner "androidx.test.runner.AndroidJUnitRunner"

        vectorDrawables {
            useSupportLibrary true
        }

        // Enable multidex
        multiDexEnabled true

        // App Bundle
        bundle {
            density {
                enableSplit = true
            }
            abi {
                enableSplit = true
            }
        }
    }

    buildTypes {
        release {
            minifyEnabled true
            shrinkResources true
            proguardFiles getDefaultProguardFile('proguard-android-optimize.txt'), 'proguard-rules.pro'
            signingConfig signingConfigs.debug
        }
        debug {
            minifyEnabled false
            applicationIdSuffix ".debug"
        }
    }

    compileOptions {
        coreLibraryDesugaringEnabled true
        sourceCompatibility JavaVersion.VERSION_17
        targetCompatibility JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = '17'
    }

    buildFeatures {
        compose true
        buildConfig true
    }

    composeOptions {
        kotlinCompilerExtensionVersion '1.5.1'
    }

    packaging {
        resources {
            excludes += '/META-INF/{AL2.0,LGPL2.1}'
        }
    }
}

dependencies {
    // Core Android
    implementation 'androidx.core:core-ktx:1.12.0'
    implementation 'androidx.lifecycle:lifecycle-runtime-ktx:2.6.2'
    implementation 'androidx.activity:activity-compose:1.8.0'
    
    // Compose
    implementation platform('androidx.compose:compose-bom:2023.10.01')
    implementation 'androidx.compose.ui:ui'
    implementation 'androidx.compose.ui:ui-graphics'
    implementation 'androidx.compose.ui:ui-tooling-preview'
    implementation 'androidx.compose.material3:material3'
    implementation 'androidx.compose.material:material-icons-extended'
    
    // Navigation
    implementation 'androidx.navigation:navigation-compose:2.7.5'
    
    // Hilt
    implementation 'com.google.dagger:hilt-android:2.48'
    kapt 'com.google.dagger:hilt-android-compiler:2.48'
    implementation 'androidx.hilt:hilt-navigation-compose:1.1.0'
    
    // Retrofit
    implementation 'com.squareup.retrofit2:retrofit:2.9.0'
    implementation 'com.squareup.retrofit2:converter-gson:2.9.0'
    implementation 'com.squareup.okhttp3:okhttp:4.12.0'
    implementation 'com.squareup.okhttp3:logging-interceptor:4.12.0'
    
    // Coroutines
    implementation 'org.jetbrains.kotlinx:kotlinx-coroutines-android:1.7.3'
    
    // DataStore
    implementation 'androidx.datastore:datastore-preferences:1.0.0'
    
    // Biometric
    implementation 'androidx.biometric:biometric:1.1.0'
    
    // Security
    implementation 'androidx.security:security-crypto:1.1.0-alpha06'
    
    // WalletConnect
    implementation 'com.walletconnect:android-core:1.14.0'
    implementation 'com.walletconnect:android-sign:1.14.0'
    
    // Web3j
    implementation 'org.web3j:core:4.9.4'
    implementation 'org.web3j:geth:4.9.4'
    
    // CoinGecko API
    implementation 'com.github.coinoid:CoinGecko:1.0.0'
    
    // Coil for images
    implementation 'io.coil-kt:coil-compose:2.5.0'
    
    // Accompanist
    implementation 'com.google.accompanist:accompanist-permissions:0.32.0'
    implementation 'com.google.accompanist:accompanist-systemuicontroller:0.32.0'
    
    // Desugaring
    coreLibraryDesugaring 'com.android.tools:desugar_jdk_libs:2.0.4'
    
    // Testing
    testImplementation 'junit:junit:4.13.2'
    androidTestImplementation 'androidx.test.ext:junit:1.1.5'
    androidTestImplementation 'androidx.test.espresso:espresso-core:3.5.1'
    androidTestImplementation platform('androidx.compose:compose-bom:2023.10.01')
    androidTestImplementation 'androidx.compose.ui:ui-test-junit4'
    debugImplementation 'androidx.compose.ui:ui-tooling'
    debugImplementation 'androidx.compose.ui:ui-test-manifest'
}
```

## ProGuard Rules

```proguard
# Keep data classes
-keepclassmembers class com.tigerwallet.android.data.model.** { *; }

# Retrofit
-keepattributes Signature
-keepattributes Exceptions
-keepattributes *Annotation*

# OkHttp
-dontwarn okhttp3.**
-dontwarn okio.**

# Web3j
-keep class org.web3j.** { *; }
-keep class org.ethereum.** { *; }

# WalletConnect
-keep class com.walletconnect.** { *; }

# Gson
-keepattributes Signature
-keep class com.google.gson.** { *; }
```

## Deep Link Configuration

```xml
<!-- AndroidManifest.xml -->
<intent-filter>
    <action android:name="android.intent.action.VIEW"/>
    <category android:name="android.intent.category.DEFAULT"/>
    <category android:name="android.intent.category.BROWSABLE"/>
    
    <!-- TigerWallet Deep Links -->
    <data android:scheme="tigerwallet"/>
    <data android:scheme="https" android:host="tigerwallet.io"/>
    
    <!-- WalletConnect -->
    <data android:scheme="wc"/>
</intent-filter>
```

## App Links

```xml
<!-- res/values/asset_links.json -->
[
  {
    "relation": ["delegate_permission/common.handle_all_urls"],
    "target": {
      "namespace": "android_app",
      "package_name": "com.tigerwallet.android",
      "sha256_cert_fingerprints": [
        "SHA256_FINGERPRINT_HERE"
      ]
    }
  }
]
```

## Google Services (google-services.json)

```json
{
  "project_info": {
    "project_number": "123456789",
    "project_id": "tigerwallet",
    "storage_bucket": "tigerwallet.appspot.com"
  },
  "client": [
    {
      "client_info": {
        "mobilesdk_app_id": "1:123456789:android:abc123",
        "android_client_info": {
          "package_name": "com.tigerwallet.android"
        }
      },
      "oauth_client": [],
      "api_key": [
        {
          "current_key": "AIzaSy..."
        }
      ],
      "services": {
        "appinvite_service": {
          "other_platform_oauth_client": []
        }
      }
    }
  ],
  "configuration_version": "1"
}
```

## Push Notifications (FCM)

```kotlin
// FirebaseMessagingService
class TigerWalletMessagingService : FirebaseMessagingService() {
    override fun onNewToken(token: String) {
        super.onNewToken(token)
        // Send token to server
        CoroutineScope(Dispatchers.IO).launch {
            apiService.updatePushToken(token)
        }
    }

    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        super.onMessageReceived(remoteMessage)
        
        val data = remoteMessage.data
        val title = data["title"] ?: "TigerWallet"
        val body = data["body"] ?: ""
        
        showNotification(title, body)
    }

    private fun showNotification(title: String, body: String) {
        val channel = NotificationChannel(
            "tigerwallet_channel",
            "TigerWallet Notifications",
            NotificationManager.IMPORTANCE_HIGH
        )
        
        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.createNotificationChannel(channel)
        
        val notification = NotificationCompat.Builder(this, "tigerwallet_channel")
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setAutoCancel(true)
            .build()
        
        notificationManager.notify(0, notification)
    }
}
```

## Biometric Authentication

```kotlin
@Composable
fun BiometricPrompt() {
    val context = LocalContext.current
    val executor = LocalLifecycleOwner.current as? ComponentActivity
    
    val biometricPrompt = remember {
        BiometricPrompt(
            context as FragmentActivity,
            executor!!,
            object : BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                    // Unlock wallet
                }

                override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                    // Handle error
                }

                override fun onAuthenticationFailed() {
                    // Authentication failed
                }
            }
        )
    }

    val promptInfo = remember {
        BiometricPrompt.PromptInfo.Builder()
            .setTitle("Unlock TigerWallet")
            .setSubtitle("Use your biometric to unlock")
            .setNegativeButtonText("Use PIN")
            .build()
    }

    Button(onClick = { biometricPrompt.authenticate(promptInfo) }) {
        Text("Unlock with Biometric")
    }
}
```

## Fastlane Configuration

```ruby
# Fastfile
default_platform(:android)

platform :android do
  desc "Deploy a new version to Google Play"
  lane :deploy do
    gradle(
      task: "assembleRelease",
      build_type: "Release"
    )

    upload_to_play_store(
      json_key_data: ENV['PLAY_STORE_JSON_KEY'],
      package_name: 'com.tigerwallet.android',
      track: 'production',
      rollout: '0.1'
    )
  end

  desc "Deploy to Internal Testing"
  lane :beta do
    gradle(
      task: "assembleRelease",
      build_type: "Release"
    )

    upload_to_play_store(
      json_key_data: ENV['PLAY_STORE_JSON_KEY'],
      package_name: 'com.tigerwallet.android',
      track: 'internal'
    )
  end

  desc "Build and test"
  lane :test do
    gradle(task: "test")
    gradle(task: "connectedDebugAndroidTest")
  end
end
```

## GitHub Actions CI/CD

```yaml
name: Android CI/CD

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Java
        uses: actions/setup-java@v3
        with:
          java-version: '17'
          distribution: 'temurin'
          
      - name: Setup Android SDK
        uses: android-actions/setup-android@v2
        
      - name: Cache Gradle
        uses: actions/cache@v3
        with:
          path: |
            ~/.gradle/caches
            ~/.gradle/wrapper
          key: ${{ runner.os }}-gradle-${{ hashFiles('**/*.gradle*') }}
          restore-keys: |
            ${{ runner.os }}-gradle-
            
      - name: Build Debug APK
        run: ./gradlew assembleDebug
        
      - name: Upload APK
        uses: actions/upload-artifact@v3
        with:
          name: app-debug
          path: app/build/outputs/apk/debug/app-debug.apk
```

## App Icons

```
Android Icon Sizes:
- mdpi: 48x48
- hdpi: 72x72
- xhdpi: 96x96
- xxhdpi: 144x144
- xxxhdpi: 192x192

Play Store Icon: 512x512 (PNG)
```

## Store Listing

- **Title**: TigerWallet - Web3 Wallet
- **Short Description**: Trade, Swap, Stake across 100+ chains
- **Full Description**: 
  TigerWallet is the ultimate enterprise-grade multichain Web3 wallet. 
  Trade, swap, stake, and manage your crypto assets across 100+ blockchain networks.
  
  Features:
  • Multi-chain support (100+ networks)
  • Instant token swaps
  • Cross-chain bridging
  • NFT management
  • Trading bots
  • DeFi integration
  • Hardware wallet support
  • Biometric security
  
- **Privacy Policy URL**: https://tigerwallet.io/privacy
- **Support URL**: https://tigerwallet.io/support
