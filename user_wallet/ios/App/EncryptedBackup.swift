import Foundation
import CryptoKit
import CommonCrypto
import UIKit

// EncryptedBackup — the offline encrypted recovery-phrase backup (mirrors the
// web BackupMnemonic "Download encrypted backup" path).
//
// Format (byte layout identical to the web blob): salt(16) || iv(12) ||
// ciphertext (AES-256-GCM sealed). The 256-bit key is derived from the wallet
// password via PBKDF2-SHA256 with 600,000 iterations (CommonCrypto
// CCKeyDerivationPBKDF). The file is written to the app Documents directory and
// shared via UIActivityViewController.
//
// Real CryptoKit AES.GCM.seal + CommonCrypto PBKDF2. No mocks.

enum EncryptedBackup {
    static let saltLength = 16
    static let ivLength = 12
    static let iterations: Int = 600_000
    static let keyLength = 32  // AES-256

    enum BackupError: LocalizedError {
        case deriveFailed
        case sealFailed
        case writeFailed

        var errorDescription: String? {
            switch self {
            case .deriveFailed: return "Failed to derive encryption key"
            case .sealFailed: return "Failed to encrypt recovery phrase"
            case .writeFailed: return "Failed to write backup file"
            }
        }
    }

    /// Derive a 256-bit key from the wallet password + salt via PBKDF2-SHA256
    /// (600k iterations) using CommonCrypto.CCKeyDerivationPBKDF.
    static func deriveKey(password: String, salt: Data) throws -> SymmetricKey {
        let passwordData = Data(password.utf8)
        var derived = Data(count: keyLength)
        let status = derived.withUnsafeMutableBytes { (d: UnsafeMutableRawBufferPointer) -> Int32 in
            salt.withUnsafeBytes { (s: UnsafeRawBufferPointer) -> Int32 in
                passwordData.withUnsafeBytes { (p: UnsafeRawBufferPointer) -> Int32 in
                    CCKeyDerivationPBKDF(
                        CCPBKDFAlgorithm(kCCPBKDF2),
                        p.bindMemory(to: CChar.self).baseAddress,
                        passwordData.count,
                        s.bindMemory(to: UInt8.self).baseAddress,
                        salt.count,
                        CCPseudoRandomAlgorithm(kCCPRFHmacAlgSHA256),
                        UInt32(iterations),
                        d.bindMemory(to: UInt8.self).baseAddress,
                        keyLength
                    )
                }
            }
        }
        guard status == kCCSuccess else { throw BackupError.deriveFailed }
        return SymmetricKey(data: derived)
    }

    /// Encrypt the mnemonic. Layout matches the web blob exactly:
    /// salt(16) || iv(12) || ciphertext+tag (AES-256-GCM, CryptoKit).
    static func encrypt(mnemonic: String, walletPassword: String) throws -> Data {
        var saltBytes = [UInt8](repeating: 0, count: saltLength)
        let saltStatus = saltBytes.withUnsafeMutableBufferPointer { ptr in
            SecRandomCopyBytes(kSecRandomDefault, ptr.count, ptr.baseAddress!)
        }
        guard saltStatus == errSecSuccess else { throw BackupError.deriveFailed }
        let salt = Data(saltBytes)

        let key = try deriveKey(password: walletPassword, salt: salt)
        let sealed: AES.GCM.SealedBox
        do {
            // .combined == nonce(12) || ciphertext || tag — exactly the web
            // (iv || ciphertext-with-appended-tag) layout.
            sealed = try AES.GCM.seal(Data(mnemonic.utf8), using: key)
        } catch {
            throw BackupError.sealFailed
        }
        var out = Data()
        out.append(salt)
        out.append(sealed.combined)
        return out
    }

    /// Write the encrypted blob to the app Documents directory and return its
    /// URL. The file name mirrors the web download name.
    static func writeEncryptedBackup(mnemonic: String, walletId: String,
                                     walletPassword: String) throws -> URL {
        let blob = try encrypt(mnemonic: mnemonic, walletPassword: walletPassword)
        let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first
            ?? FileManager.default.temporaryDirectory
        let fileName = "tigerwallet-backup-\(String(walletId.prefix(8))).enc"
        let url = docs.appendingPathComponent(fileName)
        do {
            try blob.write(to: url, options: .atomic)
        } catch {
            throw BackupError.writeFailed
        }
        return url
    }
}

// ShareSheet — UIKit bridge to present a UIActivityViewController for sharing
// the encrypted backup file from SwiftUI.
struct ShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
