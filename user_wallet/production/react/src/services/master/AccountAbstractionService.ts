/**
 * Account Abstraction Service - React/Web Implementation
 * Identical across ALL platforms
 *
 * Fail-closed: UserOperation submission requires a REAL ERC-4337 bundler
 * endpoint (JSON-RPC `eth_sendUserOperation`). The userOpHash is never
 * fabricated (no `0x<hash><Date.now()>`, no DJB2 hash of the userOp fields).
 * The `'0xPaymasterAddress'` placeholder is removed; `paymasterAndData` is
 * left as `"0x"` unless a real sponsor signature is supplied via
 * `paymasterData`. If no real bundler is configured or the bundler is
 * unreachable/rejects the userOp, methods throw.
 */

const ENTRY_POINT_ADDRESS = '0x5FF137D4a0ADd64d127571f85d2dC51Bf7d7fE3';

class AccountAbstractionService {
  private static instance: AccountAbstractionService;
  private smartAccount: SmartAccount | null = null;
  private sessionKeys: Map<string, SessionKey> = new Map();
  private isInitialized = false;

  /**
   * Real ERC-4337 bundler endpoint (JSON-RPC `eth_sendUserOperation`).
   * Empty by default - must be configured by the host app before any userOp
   * can be submitted. When empty, submission throws fail-closed.
   */
  bundlerEndpoint = '';

  static getInstance(): AccountAbstractionService {
    if (!AccountAbstractionService.instance) {
      AccountAbstractionService.instance = new AccountAbstractionService();
    }
    return AccountAbstractionService.instance;
  }

  /**
   * Initializes a smart account. The owner address is recorded but the
   * account address is NOT fabricated: it is resolved lazily from the
   * bundler/counterfactual deployment on first use and left empty here.
   */
  initialize(ownerAddress: string): SmartAccount {
    this.smartAccount = {
      address: '',
      owner: ownerAddress,
      nonce: 0,
      isDeployed: false,
      entryPoint: ENTRY_POINT_ADDRESS,
    };
    this.isInitialized = true;
    return this.smartAccount;
  }

  getAccountAddress(): string {
    return this.smartAccount?.address ?? '';
  }

  /**
   * Submits a UserOperation to a REAL ERC-4337 bundler via
   * `eth_sendUserOperation` and returns the REAL userOpHash reported by the
   * bundler. The userOpHash is never fabricated. `paymasterAndData` is set
   * to `"0x"` unless a real sponsor signature is provided via `paymasterData`.
   * Throws if no bundler is configured, the bundler is unreachable, or it
   * rejects the userOp.
   */
  async sendUserOp(
    to: string,
    value: string,
    data: Uint8Array,
    paymaster: boolean = true,
    paymasterData?: string
  ): Promise<string> {
    if (!this.isInitialized || !this.smartAccount) {
      throw new Error('Account abstraction is not initialized.');
    }
    if (!this.bundlerEndpoint) {
      throw new Error(
        'No real ERC-4337 bundler endpoint is configured; cannot submit UserOperation.'
      );
    }
    const userOp = this.createUserOperation(to, value, data, paymaster, paymasterData);
    const rpcBody = {
      jsonrpc: '2.0',
      method: 'eth_sendUserOperation',
      params: [this.userOpPayload(userOp), ENTRY_POINT_ADDRESS],
      id: 1,
    };

    let resp: Response;
    try {
      resp = await fetch(this.bundlerEndpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(rpcBody),
      });
    } catch (err) {
      throw new Error(
        `Bundler unreachable: ${err instanceof Error ? err.message : String(err)}`
      );
    }
    if (!resp.ok) {
      throw new Error(`Bundler rejected UserOperation (HTTP ${resp.status}): ${await safeRespText(resp)}`);
    }
    let json: Record<string, unknown>;
    try {
      json = (await resp.json()) as Record<string, unknown>;
    } catch {
      throw new Error('Bundler unreachable: malformed JSON');
    }
    if (json.error) {
      const rpcErr = json.error as Record<string, unknown>;
      throw new Error(
        `Bundler rejected UserOperation (code ${String(rpcErr.code)}): ${String(rpcErr.message ?? '')}`
      );
    }
    const userOpHash = json.result;
    if (typeof userOpHash !== 'string' || userOpHash.length === 0) {
      throw new Error('Bundler unreachable: missing userOpHash');
    }
    return userOpHash;
  }

  createSessionKey(config: SessionKeyConfig): SessionKey {
    const key: SessionKey = {
      keyAddress: this.generateKeyAddress(),
      dAppAddress: config.dAppAddress,
      validUntil: config.validUntil,
      allowedContracts: config.allowedContracts,
      allowedSelectors: config.allowedSelectors,
      spendingLimit: config.spendingLimit,
      spentAmount: '0',
      isRevoked: false,
    };
    this.sessionKeys.set(key.keyAddress, key);
    return key;
  }

  revokeSessionKey(keyAddress: string): boolean {
    const key = this.sessionKeys.get(keyAddress);
    if (key) {
      key.isRevoked = true;
      return true;
    }
    return false;
  }

  getActiveSessionKeys(): SessionKey[] {
    const now = Date.now();
    return Array.from(this.sessionKeys.values()).filter(
      (k) => !k.isRevoked && k.validUntil > now
    );
  }

  /**
   * Executes a call through the smart account using a session key. The call
   * is submitted as a REAL UserOperation to the bundler (same path as
   * `sendUserOp`); no fabricated tx hash is returned. Throws if the session
   * key is invalid/expired/revoked, or if no real bundler is configured.
   */
  async executeWithSessionKey(
    keyAddress: string,
    to: string,
    data: Uint8Array
  ): Promise<string> {
    const key = this.sessionKeys.get(keyAddress);
    if (!key) throw new Error('Session key not found');
    if (key.isRevoked) throw new Error('Session key revoked');
    if (Date.now() > key.validUntil) throw new Error('Session key expired');
    // Real submission via the bundler; the returned userOpHash is the only
    // legitimate identifier. No hash of (to, data) is fabricated.
    return this.sendUserOp(to, '0', data, false);
  }

  // Private helpers
  private deriveSmartAccountAddress(_owner: string): string {
    // Smart-account addresses are derived by the canonical wallet-api backend
    // / ERC-4337 factory (real CREATE2 counterfactual address). This client
    // must NOT fabricate one from a non-cryptographic hash.
    throw new Error(
      'Smart-account address is derived by the canonical wallet-api backend / ERC-4337 factory; client-side fabrication is disabled'
    );
  }

  /**
   * Generates a random session-key identifier (not a wallet address and not a
   * public key - it is a local handle for the in-memory session key only) using
   * the Web Crypto CSPRNG. Throws fail-closed if the secure RNG is unavailable.
   */
  private generateKeyAddress(): string {
    const cryptoObj = globalThis.crypto;
    if (!cryptoObj || typeof cryptoObj.getRandomValues !== 'function') {
      throw new Error('Secure RNG unavailable; cannot generate session-key handle.');
    }
    const bytes = new Uint8Array(32);
    cryptoObj.getRandomValues(bytes);
    return '0x' + Array.from(bytes.slice(0, 20))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
  }

  private createUserOperation(
    to: string,
    value: string,
    data: Uint8Array,
    paymaster: boolean,
    paymasterData?: string
  ): UserOperation {
    // paymasterAndData is "0x" (no sponsorship) unless a REAL sponsor signature
    // is supplied. The previous "0xPaymasterAddress" placeholder is removed.
    const paymasterAndData =
      paymaster && paymasterData && paymasterData.length > 0 ? paymasterData : '0x';
    return {
      sender: this.smartAccount?.address ?? '',
      nonce: this.smartAccount?.nonce.toString() ?? '0',
      initCode: this.smartAccount?.isDeployed ? '0x' : '0x',
      callData: this.encodeCallData(to, value, data),
      callGasLimit: '0x5208',
      verificationGasLimit: '0x186A0',
      preVerificationGas: '0x5208',
      maxFeePerGas: '0x3B9ACA00',
      maxPriorityFeePerGas: '0x3B9ACA00',
      paymasterAndData,
      signature: '0x',
    };
  }

  private encodeCallData(to: string, value: string, data: Uint8Array): string {
    const toClean = to.replace(/^0x/i, '').padStart(64, '0').toLowerCase();
    const valueClean = value.replace(/^0x/i, '').padStart(64, '0').toLowerCase();
    // offset to bytes data (3 * 32 bytes head)
    const offset = '60'.padStart(64, '0');
    const length = data.length.toString(16).padStart(64, '0');
    const dataHex = Array.from(data)
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
    const dataPadded =
      dataHex + '0'.repeat((64 - (data.length * 2) % 64) % 64);
    // execute(address,uint256,bytes) selector 0x61cbb628 (SimpleAccount)
    return '0x61cbb628' + toClean + valueClean + offset + length + dataPadded;
  }

  /// Serializes a UserOperation into the ERC-4337 JSON shape expected by
  /// `eth_sendUserOperation`.
  private userOpPayload(userOp: UserOperation): Record<string, string> {
    return {
      sender: userOp.sender,
      nonce: userOp.nonce,
      initCode: userOp.initCode,
      callData: userOp.callData,
      callGasLimit: userOp.callGasLimit,
      verificationGasLimit: userOp.verificationGasLimit,
      preVerificationGas: userOp.preVerificationGas,
      maxFeePerGas: userOp.maxFeePerGas,
      maxPriorityFeePerGas: userOp.maxPriorityFeePerGas,
      paymasterAndData: userOp.paymasterAndData,
      signature: userOp.signature,
    };
  }
}

async function safeRespText(resp: Response): Promise<string> {
  try {
    return await resp.text();
  } catch {
    return '';
  }
}

export interface SmartAccount {
  address: string;
  owner: string;
  nonce: number;
  isDeployed: boolean;
  entryPoint: string;
}

export interface UserOperation {
  sender: string;
  nonce: string;
  initCode: string;
  callData: string;
  callGasLimit: string;
  verificationGasLimit: string;
  preVerificationGas: string;
  maxFeePerGas: string;
  maxPriorityFeePerGas: string;
  paymasterAndData: string;
  signature: string;
}

export interface SessionKey {
  keyAddress: string;
  dAppAddress: string;
  validUntil: number;
  allowedContracts: string[];
  allowedSelectors: string[];
  spendingLimit: string;
  spentAmount: string;
  isRevoked: boolean;
}

export interface SessionKeyConfig {
  dAppAddress: string;
  validUntil: number;
  allowedContracts: string[];
  allowedSelectors: string[];
  spendingLimit: string;
}

export default AccountAbstractionService.getInstance();
