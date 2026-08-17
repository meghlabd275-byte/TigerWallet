# ProGuard / R8 rules for TigerMasterWallet release builds.
# Keep web3j crypto + okhttp/okio internals intact; cryptography and reflection
# must not be shrunk.

-keep class org.web3j.** { *; }
-keep class okhttp3.** { *; }
-keep class okio.** { *; }

# Keep EncryptedSharedPreferences MasterKey builder entry points.
-keep class androidx.security.crypto.** { *; }
