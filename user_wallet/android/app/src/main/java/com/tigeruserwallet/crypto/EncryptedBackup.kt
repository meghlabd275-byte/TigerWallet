package com.tigeruserwallet.crypto

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.util.Base64
import androidx.core.content.FileProvider
import java.io.File
import java.io.FileOutputStream
import java.security.SecureRandom
import javax.crypto.Cipher
import javax.crypto.SecretKeyFactory
import javax.crypto.spec.GCMParameterSpec
import javax.crypto.spec.PBEKeySpec
import javax.crypto.spec.SecretKeySpec

/**
 * EncryptedBackup — REAL offline encrypted-backup file fallback (mirrors the
 * web BackupMnemonic "Download encrypted backup" path: WebCrypto PBKDF2 +
 * AES-GCM). Pure javax.crypto — no fabricated security.
 *
 * Layout of the written file (and the analogous share intent):
 *   magic(8) || version(1) || salt(16) || iv(12) || ciphertext(+16 GCM tag)
 *
 * Fail-closed: a wrong password never yields plaintext (GCM auth tag fails).
 */
object EncryptedBackup {
    private const val MAGIC = "TWBK\x01\x00\x00\x00".toByteArray(Charsets.US_ASCII)
    private const val PBKDF2_ITERS = 600_000
    private const val PBKDF2_ALGORITHM = "PBKDF2WithHmacSHA256"
    private const val AES_TRANSFORM = "AES/GCM/NoPadding"
    private const val GCM_TAG_BITS = 128
    private const val SALT_LEN = 16
    private const val IV_LEN = 12

    data class Result(val uri: Uri, val file: File)

    /** Derive an AES-256 key from the wallet password via PBKDF2 (SHA-256, 600k). */
    private fun deriveKey(password: String, salt: ByteArray): SecretKeySpec {
        val spec = PBEKeySpec(password.toCharArray(), salt, PBKDF2_ITERS, 256)
        val factory = SecretKeyFactory.getInstance(PBKDF2_ALGORITHM)
        return SecretKeySpec(factory.generateSecret(spec).encoded, "AES")
    }

    private fun randBytes(n: Int): ByteArray {
        val b = ByteArray(n)
        SecureRandom().nextBytes(b)
        return b
    }

    /**
     * Encrypt [plaintext] (the mnemonic) with [password], write the result to
     * the app's external files dir, and return the file + a content Uri for
     * sharing via a real ACTION_SEND Intent. NEVER fabricates success: any IO
     * / crypto failure throws.
     */
    fun writeEncrypted(context: Context, walletId: String, password: String, plaintext: String): Result {
        val salt = randBytes(SALT_LEN)
        val key = deriveKey(password, salt)
        val iv = randBytes(IV_LEN)
        val cipher = Cipher.getInstance(AES_TRANSFORM)
        cipher.init(Cipher.ENCRYPT_MODE, key, GCMParameterSpec(GCM_TAG_BITS, iv))
        val ct = cipher.doFinal(plaintext.toByteArray(Charsets.UTF_8))

        val outDir = File(context.getExternalFilesDir(null), "backups").apply { mkdirs() }
        val shortId = walletId.take(8).ifEmpty { "wallet" }
        val outFile = File(outDir, "tigerwallet-backup-$shortId-${System.currentTimeMillis()}.enc")
        FileOutputStream(outFile).use { fos ->
            fos.write(MAGIC)
            fos.write(salt)
            fos.write(iv)
            fos.write(ct)
        }
        val authority = context.packageName + ".fileprovider"
        val uri = FileProvider.getUriForFile(context, authority, outFile)
        return Result(uri, outFile)
    }

    /** Build a real ACTION_SEND share intent for an encrypted backup Uri. */
    fun shareIntent(uri: Uri): Intent =
        Intent(Intent.ACTION_SEND).apply {
            type = "application/octet-stream"
            putExtra(Intent.EXTRA_STREAM, uri)
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        }

    /** Build the file name (without path) used for the share subject. */
    fun backupFileName(walletId: String): String =
        "tigerwallet-backup-${walletId.take(8).ifEmpty { "wallet" }}.enc"

    // ---- base64 helpers (used by callers that prefer a string blob form) ----
    fun encryptToBase64(password: String, plaintext: String): String {
        val salt = randBytes(SALT_LEN)
        val key = deriveKey(password, salt)
        val iv = randBytes(IV_LEN)
        val cipher = Cipher.getInstance(AES_TRANSFORM)
        cipher.init(Cipher.ENCRYPT_MODE, key, GCMParameterSpec(GCM_TAG_BITS, iv))
        val ct = cipher.doFinal(plaintext.toByteArray(Charsets.UTF_8))
        return Base64.encodeToString(salt + iv + ct, Base64.NO_WRAP)
    }
}
