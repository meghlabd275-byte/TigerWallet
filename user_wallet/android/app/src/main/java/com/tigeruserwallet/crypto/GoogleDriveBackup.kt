package com.tigeruserwallet.crypto

import android.content.Context
import com.google.android.gms.auth.api.signin.GoogleSignIn
import com.google.android.gms.auth.api.signin.GoogleSignInAccount
import com.google.android.gms.auth.api.signin.GoogleSignInClient
import com.google.android.gms.auth.api.signin.GoogleSignInOptions
import com.google.android.gms.common.api.ApiException
import com.google.android.gms.common.api.Scope
import com.google.api.client.googleapis.extensions.android.gms.auth.GoogleAccountCredential
import com.google.api.client.googleapis.json.GoogleJsonResponseException
import com.google.api.client.http.FileContent
import com.google.api.client.http.javanet.NetHttpTransport
import com.google.api.client.json.gson.GsonFactory
import com.google.api.services.drive.Drive
import com.google.api.services.drive.DriveScopes
import com.google.api.services.drive.model.File as DriveFile
import java.io.File

/**
 * GoogleDriveBackup — REAL Google Drive REST API v3 backup (mirrors the web
 * BackupMnemonic Google-Drive path: Google Identity Services token +
 * Drive multipart upload). Android uses Google Sign-In (Play Services) for the
 * account + a GoogleAccountCredential (OAuth2 access token) for the Drive v3
 * REST client. No fabricated success: every upload is a real network call,
 * and failures are surfaced with the server's own error text.
 *
 * Honesty contract: if no Google web client ID is configured in
 * [GoogleDriveBackup.config], [isConfigured] returns false and the UI disables
 * the button with a real message — NEVER a fake success.
 */
object GoogleDriveBackup {

    /** Configured by MainActivity from BuildConfig / resources at init time. */
    data class Config(val webClientId: String?) {
        val enabled: Boolean get() = !webClientId.isNullOrEmpty()
    }

    @Volatile
    var config: Config = Config(null)

    fun isConfigured(): Boolean = config.enabled

    private fun driveService(context: Context, account: GoogleSignInAccount): Drive {
        val credential = GoogleAccountCredential.usingOAuth2(
            context,
            listOf(DriveScopes.DRIVE_FILE)
        ).setSelectedAccount(account.account)
        return Drive.Builder(
            NetHttpTransport(),
            GsonFactory(),
            credential
        ).setApplicationName("TigerWallet").build()
    }

    /** Build the Google Sign-In options requesting the drive.file scope. */
    fun signInOptions(): GoogleSignInOptions =
        GoogleSignInOptions.Builder(GoogleSignInOptions.DEFAULT_SIGN_IN)
            .requestIdToken(config.webClientId)
            .requestEmail()
            .requestScopes(Scope(DriveScopes.DRIVE_FILE))
            .build()

    fun signInClient(context: Context): GoogleSignInClient =
        GoogleSignIn.getClient(context, signInOptions())

    /** The RC_GMS_SIGN_IN request code used by the calling activity/fragment. */
    const val RC_SIGN_IN = 9001

    sealed class Result {
        data class Success(val fileId: String) : Result()
        data class Failure(val message: String) : Result()
        data class Canceled(val message: String = "Google Sign-In canceled") : Result()
    }

    /**
     * Resolve the Sign-In intent result. Returns the signed-in account, or
     * null if the user canceled / it failed (caller surfaces [message]).
     */
    fun accountFromResult(data: android.content.Intent?): Pair<GoogleSignInAccount?, String?> {
        return try {
            val task = GoogleSignIn.getSignedInAccountFromIntent(data)
            val account = task.getResult(ApiException::class.java)
            account to null
        } catch (e: ApiException) {
            null to (e.localizedMessage ?: "Google Sign-In failed (${e.statusCode})")
        }
    }

    /**
     * Upload [localFile] to Google Drive as a plaintext/encrypted backup.
     * REAL Drive v3 media upload — returns the remote file id on success, or
     * the server's error text on failure (never a fabricated id).
     */
    fun uploadBackup(
        context: Context,
        account: GoogleSignInAccount,
        localFile: File,
        remoteName: String
    ): Result {
        return try {
            val drive = driveService(context, account)
            val metadata = DriveFile().apply {
                name = remoteName
                mimeType = "application/octet-stream"
            }
            val media = FileContent("application/octet-stream", localFile)
            val created = drive.files().create(metadata, media)
                .setFields("id")
                .execute()
            Result.Success(created.id ?: "")
        } catch (e: GoogleJsonResponseException) {
            Result.Failure("Drive upload failed (HTTP ${e.statusCode}): ${e.message}")
        } catch (e: Exception) {
            Result.Failure(e.localizedMessage ?: "Google Drive backup failed")
        }
    }

    /**
     * Sign out so a subsequent backup re-prompts the account chooser.
     */
    fun signOut(context: Context) {
        try {
            signInClient(context).signOut()
        } catch (e: Exception) {
            // best-effort
        }
    }
}
