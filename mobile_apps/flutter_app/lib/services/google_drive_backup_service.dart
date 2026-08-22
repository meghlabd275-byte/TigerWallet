///
/// Google Drive encrypted-seed backup service (R2).
///
/// Uploads the wallet's encrypted seed blob to the signed-in user's Google
/// Drive using the googleapis/drive v3 REST client. Authentication is done via
/// google_sign_in, and the resulting GoogleSignInAuthentication is bridged into
/// an authenticated HTTP client by extension_google_sign_in_as_google_auth_user
/// (which produces a client suitable for passing straight to the DriveApi
/// constructor). The OAuth scope is drive.file — app-created files only, no
/// blanket Drive read access.
///
/// FAIL-CLOSED: this service never fakes a successful backup. If no OAuth web
/// client id has been configured (via --dart-define=GOOGLE_WEB_CLIENT_ID=...),
/// or the user cancels sign-in, or the upload fails, backup() returns an
/// honest error string rather than a fabricated success.
///
/// The content uploaded is the AES-256-GCM encrypted seed blob exported by the
/// canonical Go wallet_api endpoint POST /wallets/:id/export-encrypted-seed
/// (the raw seed / mnemonic is NEVER uploaded). The backup therefore requires
/// a registered backend wallet (wallet_id) and the wallet password.
///

import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;
import 'package:google_sign_in/google_sign_in.dart';
import 'package:googleapis/drive/v3.dart' as drive;
import 'package:extension_google_sign_in_as_google_auth_user/extension_google_sign_in_as_google_auth_user.dart';
import 'package:path_provider/path_provider.dart';
import '../utils/constants.dart';

/// Result of a Google Drive backup attempt.
class GoogleDriveBackupResult {
  final bool success;
  final String? fileId;
  final String? error;

  GoogleDriveBackupResult.success(this.fileId)
      : success = true,
        error = null;

  GoogleDriveBackupResult.failure(this.error)
      : success = false,
        fileId = null;
}

class GoogleDriveBackupService {
  static const String _scopes = 'https://www.googleapis.com/auth/drive.file';

  final FlutterSecureStorage _secureStorage;

  /// OAuth web client id. Configured at build time via
  /// --dart-define=GOOGLE_WEB_CLIENT_ID=<id>. When empty, Drive backup is
  /// considered "not configured" and the service fails closed.
  final String? _webClientId;

  GoogleDriveBackupService({
    FlutterSecureStorage? secureStorage,
    String? webClientId,
  })  : _secureStorage = secureStorage ?? const FlutterSecureStorage(),
        _webClientId = webClientId ??
            const String.fromEnvironment('GOOGLE_WEB_CLIENT_ID');

  /// Whether a Drive backup can be attempted at all (OAuth client configured).
  bool get isConfigured =>
      _webClientId != null && _webClientId!.isNotEmpty;

  /// Export the encrypted seed blob from the backend, then upload it to Google
  /// Drive under the signed-in account. Returns a [GoogleDriveBackupResult] —
  /// never throws; callers surface `error` to the user.
  ///
  /// [walletId] is the backend wallet id (from WalletService.createBackendWallet).
  /// [password] verifies the wallet before the backend hands out the blob.
  Future<GoogleDriveBackupResult> backup({
    required String walletId,
    required String password,
  }) async {
    if (!isConfigured) {
      return GoogleDriveBackupResult.failure(
        'Google Drive backup not configured — set up OAuth in settings',
      );
    }

    // 1) Export the AES-256-GCM encrypted seed blob from the backend.
    final authToken = await _secureStorage.read(key: 'auth_token') ?? '';
    http.Response exportResp;
    try {
      exportResp = await http
          .post(
            Uri.parse(
                '$API_BASE_URL/api/v1/wallets/$walletId/export-encrypted-seed'),
            headers: {
              'Content-Type': 'application/json',
              if (authToken.isNotEmpty) 'Authorization': 'Bearer $authToken',
            },
            body: jsonEncode({'password': password}),
          )
          .timeout(const Duration(seconds: 20));
    } catch (e) {
      return GoogleDriveBackupResult.failure(
        'Could not reach backup backend: $e',
      );
    }
    if (exportResp.statusCode != 200) {
      return GoogleDriveBackupResult.failure(
        'Backend refused to export the encrypted seed (${exportResp.statusCode}): '
        '${exportResp.body}',
      );
    }
    final Map<String, dynamic> blob = jsonDecode(exportResp.body);
    final encryptedSeed = blob['encrypted_seed']?.toString();
    if (encryptedSeed == null || encryptedSeed.isEmpty) {
      return GoogleDriveBackupResult.failure(
        'Backend returned no encrypted_seed blob',
      );
    }

    // 2) Sign in with Google (drive.file scope) and obtain an authenticated
    //    HTTP client for the googleapis Drive client.
    final GoogleSignIn googleSignIn = GoogleSignIn(
      scopes: const [_scopes],
      // clientId is platform-defaulted from the Flutter plugins' native config
      // (Android: google-services.json / iOS: Info.plist). The web client id
      // is used for the server-auth code flow on Android.
      serverClientId: _webClientId,
    );
    try {
      await googleSignIn.signOut();
      final account = await googleSignIn.signIn();
      if (account == null) {
        return GoogleDriveBackupResult.failure(
          'Google sign-in cancelled — backup not uploaded',
        );
      }
      final authClient = await account.authenticatedClient();
      try {
        // 3) Upload the encrypted seed blob as a JSON file to Drive.
        final driveApi = drive.DriveApi(authClient);
        final String fileContent =
            const JsonEncoder.withIndent('  ').convert(blob);
        final Uint8List bytes = Uint8List.fromList(utf8.encode(fileContent));

        // googleapis' Media takes a Stream<List<int>>; write to a temp file.
        final tmpDir = await getTemporaryDirectory();
        final tmpFile = File('${tmpDir.path}/tigerwallet_encrypted_seed.json');
        await tmpFile.writeAsBytes(bytes);

        final media = drive.Media(
          tmpFile.openRead(),
          bytes.length,
          contentType: 'application/json',
        );
        final fileToCreate = drive.File()
          ..name = 'tigerwallet_encrypted_seed.json'
          ..mimeType = 'application/json';
        final created = await driveApi.files.create(
          fileToCreate,
          uploadMedia: media,
          $fields: 'id,name',
        );

        // Best-effort cleanup of the temp file.
        try {
          await tmpFile.delete();
        } catch (_) {}

        if (created.id == null || created.id!.isEmpty) {
          return GoogleDriveBackupResult.failure(
            'Google Drive returned no file id — upload status unknown',
          );
        }
        return GoogleDriveBackupResult.success(created.id);
      } finally {
        authClient.close();
      }
    } catch (e) {
      return GoogleDriveBackupResult.failure(
        'Google Drive backup failed: $e',
      );
    }
  }
}
