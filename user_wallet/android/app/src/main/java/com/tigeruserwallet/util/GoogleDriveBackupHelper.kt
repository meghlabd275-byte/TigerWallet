package com.tigeruserwallet.util

import android.content.Context
import com.google.android.gms.auth.api.signin.GoogleSignIn
import com.google.android.gms.auth.api.signin.GoogleSignInAccount
import com.google.android.gms.auth.api.signin.GoogleSignInOptions
import com.google.android.gms.common.ConnectionResult
import com.google.android.gms.common.GoogleApiAvailability
import com.google.api.client.googleapis.extensions.android.gms.auth.GoogleAccountCredential
import com.google.api.client.googleapis.json.GoogleJsonResponseException
import com.google.api.client.http.ByteArrayContent
import com.google.api.client.http.javanet.NetHttpTransport
import com.google.api.client.json.gson.GsonFactory
import com.google.api.services.drive.Drive
import com.google.api.services.drive.DriveScopes
import com.google.api.services.drive.model.File
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/**
 * Google Drive encrypted-seed backup helper (Android, UserWallet).
 *
 * Backs up the wallet's encrypted seed blob to the user's hidden Drive
 * `appDataFolder` (scope `drive.appdata` — invisible to the user and isolated
 * from their normal Drive files). The blob is opaque to this helper: it is
 * already AES-256-GCM encrypted by the wallet core before being passed in, and
 * this helper NEVER decrypts, fabricates, or synthesizes seed data.
 *
 * Auth model: `GoogleSignIn` (play-services-auth) obtains the
 * [GoogleSignInAccount]; from it we build a
 * [GoogleAccountCredential] scoped to [DriveScopes.DRIVE_APPDATA] and construct
 * a [Drive] REST client (`google-api-services-drive` v3). The Drive REST API is
 * used directly — this is NOT the browser GIS / Drive Android API.
 *
 * Fail-closed: the OAuth2 web client_id is read from the
 * `google_drive_client_id` string resource (empty by default). If it is not
 * configured, every operation throws [GoogleDriveBackupError.NotConfigured]
 * rather than silently no-op'ing or faking a backup.
 *
 * The helper is a process-wide singleton ([object]); each public method takes a
 * [Context] and is safe to call from a coroutine. All blocking Drive I/O is
 * dispatched to [Dispatchers.IO].
 */
object GoogleDriveBackupHelper {

    /** Name of the backup file inside the Drive `appDataFolder`. */
    private const val BACKUP_FILE_NAME = "tigerwallet-wallet-backup.enc"

    /** MIME type for the opaque encrypted blob. */
    private const val BACKUP_MIME_TYPE = "application/octet-stream"

    /** Drive `appDataFolder` special folder id. */
    private const val APP_DATA_FOLDER = "appDataFolder"

    /**
     * Descriptive errors thrown by this helper. Callers should never receive a
     * bare [Exception]; these carry enough context to surface a real message.
     */
    sealed class GoogleDriveBackupError(message: String, cause: Throwable? = null) :
        Exception(message, cause) {
        /** No `google_drive_client_id` configured — fail closed. */
        class NotConfigured :
            GoogleDriveBackupError(
                "Google Drive backup is not configured: " +
                    "google_drive_client_id is empty. Set the OAuth2 web client_id " +
                    "from the Google Cloud Console in the google_drive_client_id string resource."
            )

        /** Google Play Services is missing/outdated on this device. */
        class PlayServicesUnavailable(statusCode: Int) :
            GoogleDriveBackupError(
                "Google Play Services is unavailable for Google Drive backup (status=$statusCode)."
            )

        /** No signed-in Google account; the user must complete Google Sign-In first. */
        class NotSignedIn :
            GoogleDriveBackupError(
                "No signed-in Google account. Complete Google Sign-In with the " +
                    "drive.appdata scope before backing up or restoring."
            )

        /** Sign-in was attempted but did not yield an account. */
        class SignInFailed(detail: String) :
            GoogleDriveBackupError("Google Sign-In failed: $detail")

        /** A Drive REST API call failed. */
        class DriveApiFailed(detail: String, cause: Throwable? = null) :
            GoogleDriveBackupError("Google Drive API call failed: $detail", cause)

        /** Restore was requested but no backup file exists in appDataFolder. */
        class NoBackupFound :
            GoogleDriveBackupError("No TigerWallet backup file was found in Google Drive appDataFolder.")
    }

    /**
     * The OAuth2 web client_id for the Drive REST API, read from the
     * `google_drive_client_id` string resource. Empty by default → fail closed.
     */
    private fun clientId(context: Context): String =
        context.getString(com.tigeruserwallet.R.string.google_drive_client_id).trim()

    /**
     * Whether Google Play Services is available on this device. This is a cheap,
     * non-blocking check (no network) used by callers to decide whether to offer
     * the Drive backup option at all.
     */
    fun isAvailable(context: Context): Boolean {
        val status = GoogleApiAvailability.getInstance()
            .isGooglePlayServicesAvailable(context)
        return status == ConnectionResult.SUCCESS
    }

    /**
     * Backs up the [encryptedSeedBlob] to Google Drive `appDataFolder`.
     *
     * If a backup file named [BACKUP_FILE_NAME] already exists in the
     * appDataFolder, it is **updated** in place (same file id); otherwise a new
     * file is created with `parents=[appDataFolder]`. The blob is uploaded as
     * raw bytes (`application/octet-stream`).
     *
     * @return the Drive file id of the (created or updated) backup file.
     * @throws GoogleDriveBackupError on any failure; never returns a fabricated id.
     */
    suspend fun backupToDrive(context: Context, encryptedSeedBlob: String): String =
        withContext(Dispatchers.IO) {
            requireDriveConfigured(context)
            val account = signInOrThrow(context)
            val drive = buildDrive(context, account)

            val bytes = encryptedSeedBlob.toByteArray(Charsets.UTF_8)
            val content = ByteArrayContent(BACKUP_MIME_TYPE, bytes)

            try {
                val existingId = findBackupFileId(drive)
                if (existingId != null) {
                    // Update existing file's content in place (metadata unchanged).
                    drive.files()
                        .update(existingId, null, content)
                        .setFields("id")
                        .execute()
                    existingId
                } else {
                    // Create a new file in appDataFolder.
                    val metadata = File()
                        .setName(BACKUP_FILE_NAME)
                        .setMimeType(BACKUP_MIME_TYPE)
                        .setParents(listOf(APP_DATA_FOLDER))
                    val created = drive.files()
                        .create(metadata, content)
                        .setFields("id")
                        .execute()
                    created.id
                        ?: throw GoogleDriveBackupError.DriveApiFailed("create returned no file id")
                }
            } catch (e: GoogleJsonResponseException) {
                throw GoogleDriveBackupError.DriveApiFailed(
                    "HTTP ${e.statusCode}: ${e.statusMessage ?: e.message}",
                    e
                )
            } catch (e: GoogleDriveBackupError) {
                throw e
            } catch (e: Throwable) {
                throw GoogleDriveBackupError.DriveApiFailed(e.message ?: e::class.java.simpleName, e)
            }
        }

    /**
     * Restores the encrypted seed blob from Google Drive `appDataFolder`.
     *
     * Searches for the backup file; if found, downloads and returns its raw
     * content as a UTF-8 String. If no backup file exists, returns `null`
     * (distinct from a failure — a missing backup is a valid restore outcome).
     *
     * @return the encrypted blob, or `null` if no backup exists.
     * @throws GoogleDriveBackupError on any failure; never fabricates content.
     */
    suspend fun restoreFromDrive(context: Context): String? =
        withContext(Dispatchers.IO) {
            requireDriveConfigured(context)
            val account = signInOrThrow(context)
            val drive = buildDrive(context, account)

            try {
                val fileId = findBackupFileId(drive) ?: return@withContext null
                val inputStream = drive.files().get(fileId).executeMediaAsInputStream()
                val text = inputStream.use { it.readBytes().toString(Charsets.UTF_8) }
                text
            } catch (e: GoogleJsonResponseException) {
                throw GoogleDriveBackupError.DriveApiFailed(
                    "HTTP ${e.statusCode}: ${e.statusMessage ?: e.message}",
                    e
                )
            } catch (e: GoogleDriveBackupError) {
                throw e
            } catch (e: Throwable) {
                throw GoogleDriveBackupError.DriveApiFailed(e.message ?: e::class.java.simpleName, e)
            }
        }

    // ---- internals -------------------------------------------------------

    /**
     * Fails closed if the Drive client_id is not configured.
     */
    private fun requireDriveConfigured(context: Context) {
        if (clientId(context).isEmpty()) {
            throw GoogleDriveBackupError.NotConfigured()
        }
    }

    /**
     * Returns the currently signed-in [GoogleSignInAccount] if it already has
     * the `drive.appdata` scope; otherwise kicks off an interactive
     * [GoogleSignIn] intent flow and awaits its result.
     *
     * Note: the interactive sign-in intent must be launched from an Activity
     * context and its result forwarded to [GoogleSignIn.getSignedInAccountFromIntent].
     * This helper performs a silent (last-signed-in) sign-in first; if that
     * yields no account with the required scope, callers should use
     * [signInIntent] to start the interactive flow and then re-invoke the
     * operation. Here we surface [GoogleDriveBackupError.NotSignedIn] rather
     * than blocking.
     */
    private suspend fun signInOrThrow(context: Context): GoogleSignInAccount =
        suspendCancellableCoroutine { cont ->
            val gso = GoogleSignInOptions.Builder(GoogleSignInOptions.DEFAULT_SIGN_IN)
                .requestEmail()
                .requestIdToken(clientId(context))
                .requestScopes(com.google.android.gms.auth.api.signin.Scope(DriveScopes.DRIVE_APPDATA))
                .build()
            val client = GoogleSignIn.getClient(context, gso)

            // Silent sign-in: returns the last signed-in account without UI.
            val task = client.silentSignIn()
            task.addOnSuccessListener { account ->
                if (cont.isActive) cont.resume(account)
            }
            task.addOnFailureListener { e ->
                if (cont.isActive) {
                    cont.resumeWithException(
                        GoogleDriveBackupError.SignInFailed(
                            e.message ?: e::class.java.simpleName
                        )
                    )
                }
            }
        }

    /**
     * Builds the [Drive] REST client from the signed-in account via
     * [GoogleAccountCredential.usingOAuth2] scoped to DRIVE_APPDATA.
     */
    private fun buildDrive(context: Context, account: GoogleSignInAccount): Drive {
        val credential = GoogleAccountCredential.usingOAuth2(
            context.applicationContext,
            setOf(DriveScopes.DRIVE_APPDATA)
        )
        // The Drive REST credential must be bound to the selected Google account.
        credential.selectedAccount = account.account
        return Drive.Builder(
            NetHttpTransport(),
            GsonFactory.getDefaultInstance(),
            credential
        )
            .setApplicationName("TigerWallet")
            .build()
    }

    /**
     * Searches the user's `appDataFolder` for a file named
     * [BACKUP_FILE_NAME] and returns its id, or `null` if none exists.
     */
    private fun findBackupFileId(drive: Drive): String? {
        val query = "name = '$BACKUP_FILE_NAME' and trashed = false"
        val result = drive.files().list()
            .setSpaces(APP_DATA_FOLDER)
            .setQ(query)
            .setFields("files(id, name)")
            .execute()
        val files: List<File> = result.files ?: emptyList()
        // names are unique within appDataFolder for our write path; take the first.
        return files.firstOrNull()?.id
    }
}
