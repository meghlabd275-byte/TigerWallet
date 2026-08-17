/**
 * Google Drive encrypted-seed backup for UserWallet (browser extension).
 *
 * Flow:
 *   Backup:  wallet_api POST /wallets/:id/export-encrypted-seed (password) ->
 *            {encrypted_seed, ...} -> upload to Google Drive appDataFolder.
 *   Restore: download backup file from Drive appDataFolder -> wallet_api POST
 *            /wallets/import-encrypted-seed (encrypted_seed, password) ->
 *            re-stores the wallet under the current user.
 *
 * Security: only the AES-256-GCM encrypted seed blob (salt+ciphertext hex) is
 * uploaded to Drive — NEVER the raw mnemonic or seed. The blob is useless
 * without the user's wallet password. Uses the Drive appDataFolder scope
 * (https://www.googleapis.com/auth/drive.appdata) so only this app can see the
 * backup file, not the user's whole Drive.
 *
 * Google OAuth requires a real client_id (Google Cloud Console). Extensions
 * cannot use process.env — the client_id is read from chrome.storage.local key
 * `googleDriveClientId`, falling back to the GOOGLE_DRIVE_CLIENT_ID constant
 * below. Set the constant (or write the storage key) before shipping; fail-closed
 * if neither is set.
 */

// TODO: set this to your Google OAuth client_id (Google Cloud Console) before shipping.
const GOOGLE_DRIVE_CLIENT_ID = '';

const DRIVE_UPLOAD_URL = 'https://www.googleapis.com/upload/drive/v3/files';
const DRIVE_FILES_URL = 'https://www.googleapis.com/drive/v3/files';
const BACKUP_FILENAME = 'tigerwallet-wallet-backup.enc';
// appDataFolder is a private Drive space only the creating app can access.
const BACKUP_FOLDER = 'appDataFolder';

// Read the client_id from chrome.storage.local (set by the user/options page),
// falling back to the hardcoded constant above.
const getClientId = async () => {
  let stored = '';
  try {
    const res = await new Promise((resolve) => {
      if (typeof chrome !== 'undefined' && chrome.storage && chrome.storage.local) {
        chrome.storage.local.get('googleDriveClientId', resolve);
      } else {
        resolve({});
      }
    });
    stored = (res && res.googleDriveClientId) || '';
  } catch {
    stored = '';
  }
  const id = stored || GOOGLE_DRIVE_CLIENT_ID || '';
  if (!id) {
    throw new Error(
      'Google Drive backup is not configured: set googleDriveClientId in chrome.storage.local or the GOOGLE_DRIVE_CLIENT_ID constant'
    );
  }
  return id;
};

// Load the Google Identity Services (GIS) token client dynamically.
const loadGis = () =>
  new Promise((resolve, reject) => {
    if (typeof window === 'undefined') return reject(new Error('browser only'));
    if (window.google && window.google.accounts && window.google.accounts.oauth2) {
      return resolve(window.google);
    }
    const s = document.createElement('script');
    s.src = 'https://accounts.google.com/gsi/client';
    s.async = true;
    s.defer = true;
    s.onload = () => {
      if (window.google && window.google.accounts && window.google.accounts.oauth2) {
        resolve(window.google);
      } else {
        reject(new Error('Google Identity Services failed to load'));
      }
    };
    s.onerror = () => reject(new Error('Google Identity Services blocked'));
    document.head.appendChild(s);
  });

// Request an OAuth2 access token with the drive.appdata scope via GIS popup.
const requestDriveAccessToken = async () => {
  const clientId = await getClientId();
  const google = await loadGis();
  return new Promise((resolve, reject) => {
    const tokenClient = google.accounts.oauth2.initTokenClient({
      client_id: clientId,
      scope: 'https://www.googleapis.com/auth/drive.appdata',
      callback: (resp) => {
        if (resp.access_token) resolve(resp.access_token);
        else reject(new Error(resp.error_description || 'Google auth failed'));
      },
      error_callback: (err) =>
        reject(new Error((err && err.message) || 'Google auth failed')),
    });
    tokenClient.requestAccessToken({ prompt: 'consent' });
  });
};

// Upload the encrypted seed blob to Drive appDataFolder. Returns the Drive file id.
const backupToDrive = async (encryptedSeedBlob) => {
  const token = await requestDriveAccessToken();
  // Search for an existing backup file so we overwrite (not duplicate).
  const listRes = await fetch(
    `${DRIVE_FILES_URL}?spaces=appDataFolder&q=name='${BACKUP_FILENAME}'&fields=files(id,name)`,
    { headers: { Authorization: `Bearer ${token}` } }
  );
  if (!listRes.ok) throw new Error(`Drive list failed (${listRes.status})`);
  const list = await listRes.json();
  const existingId = list.files && list.files[0] && list.files[0].id;

  const metadata = {
    name: BACKUP_FILENAME,
    parents: existingId ? undefined : [BACKUP_FOLDER],
  };
  const boundary = 'tigerwallet-' + Date.now();
  const body =
    `--${boundary}\r\n` +
    'Content-Type: application/json; charset=UTF-8\r\n\r\n' +
    JSON.stringify(metadata) +
    `\r\n--${boundary}\r\n` +
    'Content-Type: application/octet-stream\r\n\r\n' +
    encryptedSeedBlob +
    `\r\n--${boundary}--`;
  const url = existingId
    ? `${DRIVE_UPLOAD_URL}/${existingId}?uploadType=multipart`
    : `${DRIVE_UPLOAD_URL}?uploadType=multipart`;
  const res = await fetch(url, {
    method: existingId ? 'PATCH' : 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': `multipart/related; boundary=${boundary}`,
    },
    body,
  });
  if (!res.ok) {
    const t = await res.text();
    throw new Error(`Drive upload failed (${res.status}): ${t}`);
  }
  const data = await res.json();
  return data.id;
};

// Download the encrypted seed blob from Drive appDataFolder. Returns null if no backup exists.
const restoreFromDrive = async () => {
  const token = await requestDriveAccessToken();
  const listRes = await fetch(
    `${DRIVE_FILES_URL}?spaces=appDataFolder&q=name='${BACKUP_FILENAME}'&fields=files(id,name)`,
    { headers: { Authorization: `Bearer ${token}` } }
  );
  if (!listRes.ok) throw new Error(`Drive list failed (${listRes.status})`);
  const list = await listRes.json();
  const fileId = list.files && list.files[0] && list.files[0].id;
  if (!fileId) return null; // no backup found
  const dlRes = await fetch(`${DRIVE_FILES_URL}/${fileId}?alt=media`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!dlRes.ok) throw new Error(`Drive download failed (${dlRes.status})`);
  return await dlRes.text();
};

// Expose on the global scope when loaded as a classic <script> (extension
// popup.js is not an ES module). No ESM `export`s are used so the file loads
// cleanly as a classic script.
if (typeof window !== 'undefined') {
  window.requestDriveAccessToken = requestDriveAccessToken;
  window.backupToDrive = backupToDrive;
  window.restoreFromDrive = restoreFromDrive;
}
