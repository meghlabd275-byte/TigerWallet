import { ethers, keccak256, toUtf8Bytes } from "ethers";

/**
 * Smart Account Service for ERC-4337
 * Handles encoding/decoding of smart account operations
 */
export class SmartAccountService {
  /**
   * Encode initialize data for smart account
   * @param owner - The EOA address that will own the smart account
   */
  static encodeInitialize(owner: string): string {
    const iface = new ethers.Interface([
      "function initialize(address owner)"
    ]);
    return iface.encodeFunctionData("initialize", [owner]);
  }

  /**
   * Encode execute call for smart account
   * @param to - Target address
   * @param value - Amount of ETH to send
   */
  static encodeExecute(to: string, value: bigint): string {
    const iface = new ethers.Interface([
      "function execute(address dest, uint256 value, bytes calldata func)"
    ]);
    return iface.encodeFunctionData("execute", [to, value, "0x"]);
  }

  /**
   * Encode batch execute calls
   * @param calls - Array of {to, value, data}
   */
  static encodeExecuteBatch(calls: Array<{ to: string; value: bigint; data: string }>): string {
    const iface = new ethers.Interface([
      "function executeBatch(address[] calldata dest, uint256[] calldata value, bytes[] calldata func)"
    ]);
    return iface.encodeFunctionData("executeBatch", [
      calls.map(c => c.to),
      calls.map(c => c.value),
      calls.map(c => c.data)
    ]);
  }

  /**
   * Hash user operation for signing
   * @param userOp - User operation struct
   */
  static hashUserOp(userOp: {
    sender: string;
    nonce: bigint;
    initCode: string;
    callData: string;
    callGasLimit: number;
    verificationGasLimit: number;
    preVerificationGas: number;
    maxFeePerGas: bigint;
    maxPriorityFeePerGas: bigint;
    signature: string;
    paymasterAndData: string;
  }): string {
    const packed = ethers.solidityPacked(
      ["address", "uint256", "bytes32", "bytes32", "uint256", "uint256", "uint256", "uint256", "uint256", "bytes32", "bytes32"],
      [
        userOp.sender,
        userOp.nonce,
        keccak256(userOp.initCode),
        keccak256(userOp.callData),
        userOp.callGasLimit,
        userOp.verificationGasLimit,
        userOp.preVerificationGas,
        userOp.maxFeePerGas,
        userOp.maxPriorityFeePerGas,
        keccak256(userOp.signature),
        keccak256(userOp.paymasterAndData)
      ]
    );
    return keccak256(packed);
  }

  /**
   * Get user operation hash (with domain separator)
   */
  static getUserOpHash(
    userOp: any,
    entryPoint: string,
    chainId: number
  ): string {
    const userOpHash = this.hashUserOp(userOp);
    const domainSeparator = keccak256(
      toUtf8Bytes("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)")
    );
    
    return keccak256(
      ethers.solidityPacked(
        ["bytes32", "bytes32", "bytes32"],
        [
          "0x1901",
          domainSeparator,
          keccak256(
            ethers.solidityPacked(
              ["uint256", "address"],
              [chainId, entryPoint]
            )
          )
        ]
      )
    ) + userOpHash.slice(2);
  }

  /**
   * Calculate smart account address before deployment
   */
  static async getAccountAddress(
    factory: string,
    owner: string,
    salt: number
  ): Promise<string> {
    const initCodeHash = keccak256(
      ethers.solidityPacked(
        ["bytes", "uint256"],
        [ethers.AbiCoder.defaultAbiCoder().encode(["address"], [owner]), salt]
      )
    );
    
    return ethers.getCreate2Address(
      factory,
      ethers.zeroPadValue(ethers.toBeHex(salt), 32),
      initCodeHash
    );
  }
}

/**
 * Paymaster service for sponsored transactions
 */
export class PaymasterService {
  static encodePaymasterData(
    paymaster: string,
    validUntil: number,
    validAfter: number,
    signature: string
  ): string {
    return ethers.solidityPacked(
      ["address", "uint48", "uint48", "bytes"],
      [paymaster, validUntil, validAfter, signature]
    );
  }

  static async validatePaymasterData(
    paymasterData: string,
    userOp: any
  ): Promise<{ valid: boolean; context: string }> {
    return { valid: true, context: "0x" };
  }
}

/**
 * Bundler service for submitting user operations
 */
export class BundlerService {
  private bundlerUrl: string;
  private entryPoint: string;

  constructor(bundlerUrl: string, entryPoint: string) {
    this.bundlerUrl = bundlerUrl;
    this.entryPoint = entryPoint;
  }

  async sendUserOperation(userOp: any): Promise<string> {
    const response = await fetch(this.bundlerUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        method: "eth_sendUserOperation",
        params: [userOp, this.entryPoint],
        id: 1
      })
    });

    const result = await response.json();
    if (result.error) {
      throw new Error(result.error.message);
    }
    return result.result;
  }

  async estimateGas(userOp: any): Promise<{
    callGasLimit: bigint;
    verificationGasLimit: bigint;
    preVerificationGas: bigint;
  }> {
    const response = await fetch(this.bundlerUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        method: "eth_estimateUserOperationGas",
        params: [userOp, this.entryPoint],
        id: 1
      })
    });

    const result = await response.json();
    if (result.error) {
      throw new Error(result.error.message);
    }
    
    return {
      callGasLimit: BigInt(result.result.callGasLimit),
      verificationGasLimit: BigInt(result.result.verificationGasLimit),
      preVerificationGas: BigInt(result.result.preVerificationGas)
    };
  }

  async getUserOpReceipt(userOpHash: string): Promise<any> {
    const response = await fetch(this.bundlerUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        method: "eth_getUserOperationReceipt",
        params: [userOpHash],
        id: 1
      })
    });

    const result = await response.json();
    return result.result;
  }
}
