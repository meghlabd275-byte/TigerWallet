import { Interface, getAddress, type ContractRunner } from "ethers";

import {
  accountFactoryContract,
  entryPointContract,
  isEntryPointV06,
  packAccountGasLimits,
  packGasFees,
  factoryAddressMethod,
  type FactoryMode,
  type UserOperationV06,
  type UserOperationV07
} from "./contracts";

/** A call the smart account should perform. */
export interface AccountCall {
  to: string;
  value: bigint;
  data: string;
}

const EXECUTE_ABI = [
  "function execute(address dest, uint256 value, bytes func)",
  "function executeBatch(address[] dest, uint256[] value, bytes[] func)"
];

/**
 * Encoding helpers for smart-account calls.
 *
 * The encodings here are the real ones from the canonical `SimpleAccount`; the
 * account decodes them on-chain.
 */
export class SmartAccountEncoder {
  /** Encode a single `execute` call. */
  static encodeExecute(to: string, value: bigint, data = "0x"): string {
    const iface = new Interface(EXECUTE_ABI);
    return iface.encodeFunctionData("execute", [getAddress(to), value, data]);
  }

  /** Encode a batched `executeBatch` call. */
  static encodeExecuteBatch(calls: AccountCall[]): string {
    if (calls.length === 0) {
      throw new Error("a batch must contain at least one call");
    }

    const iface = new Interface(EXECUTE_ABI);
    return iface.encodeFunctionData("executeBatch", [
      calls.map((call) => getAddress(call.to)),
      calls.map((call) => call.value),
      calls.map((call) => call.data)
    ]);
  }
}

/**
 * Reads and writes against the deployed EntryPoint and factory.
 *
 * Every value returned here comes from the chain. Nothing is derived locally,
 * because the factory's proxy bytecode is large and reimplementing its CREATE2
 * inputs off-chain would risk reporting an address the factory never deploys to.
 */
export class SmartAccountService {
  private readonly runner: ContractRunner;
  private readonly factoryMode: FactoryMode;

  constructor(runner: ContractRunner, factoryMode: FactoryMode = "canonical") {
    this.runner = runner;
    this.factoryMode = factoryMode;
  }

  /**
   * The counterfactual account address, read from the deployed factory.
   */
  async getAccountAddress(factory: string, owner: string, salt: bigint): Promise<string> {
    const contract = accountFactoryContract(factory, this.factoryMode, this.runner);
    const address: string = await contract[factoryAddressMethod(this.factoryMode)](
      getAddress(owner),
      salt
    );

    if (address === "0x0000000000000000000000000000000000000000") {
      throw new Error("the factory returned the zero address");
    }

    return address;
  }

  /**
   * The operation hash the EntryPoint will verify the signature against.
   *
   * This is the contract's own view of the hash, so a signature over it is
   * valid on-chain regardless of which EntryPoint version is deployed.
   */
  async getUserOpHash(
    entryPoint: string,
    userOp: UserOperationV06 | UserOperationV07
  ): Promise<string> {
    const contract = entryPointContract(entryPoint, this.runner);
    const hash: string = await contract.getUserOpHash(userOp);
    return hash;
  }

  /**
   * The account's next nonce for a key, read from the EntryPoint.
   *
   * `key` selects a nonce lane; zero is the default lane.
   */
  async getNonce(entryPoint: string, sender: string, key = 0n): Promise<bigint> {
    const contract = entryPointContract(entryPoint, this.runner);
    const nonce: bigint = await contract.getNonce(getAddress(sender), key);
    return nonce;
  }

  /**
   * The EntryPoint deposit credited to an account, in wei.
   */
  async getDeposit(entryPoint: string, account: string): Promise<bigint> {
    const contract = entryPointContract(entryPoint, this.runner);
    const info = await contract.getDepositInfo(getAddress(account));

    // The tuple layout differs between versions, but `deposit` is first in
    // both, so it is read by position.
    const deposit = info[0] ?? info.deposit;
    return BigInt(deposit);
  }

  /**
   * Build a user operation for the deployed EntryPoint version.
   *
   * The gas limits and fees must be filled by the bundler; the values supplied
   * here are the ones the caller already knows.
   */
  buildUserOp(params: {
    entryPoint: string;
    sender: string;
    nonce: bigint;
    callData: string;
    initCode?: string;
    callGasLimit: bigint;
    verificationGasLimit: bigint;
    preVerificationGas: bigint;
    maxFeePerGas: bigint;
    maxPriorityFeePerGas: bigint;
    paymasterAndData?: string;
    signature?: string;
  }): UserOperationV06 | UserOperationV07 {
    const {
      entryPoint,
      sender,
      nonce,
      callData,
      initCode = "0x",
      callGasLimit,
      verificationGasLimit,
      preVerificationGas,
      maxFeePerGas,
      maxPriorityFeePerGas,
      paymasterAndData = "0x",
      signature = "0x"
    } = params;

    if (isEntryPointV06(entryPoint)) {
      return {
        sender: getAddress(sender),
        nonce,
        initCode,
        callData,
        callGasLimit,
        verificationGasLimit,
        preVerificationGas,
        maxFeePerGas,
        maxPriorityFeePerGas,
        paymasterAndData,
        signature
      };
    }

    return {
      sender: getAddress(sender),
      nonce,
      initCode,
      callData,
      accountGasLimits: packAccountGasLimits(verificationGasLimit, callGasLimit),
      preVerificationGas,
      gasFees: packGasFees(maxPriorityFeePerGas, maxFeePerGas),
      paymasterAndData,
      signature
    };
  }

  /**
   * Encode paymaster data in the layout the canonical paymasters expect:
   * the paymaster address, then a 48-bit valid-until, a 48-bit valid-after and
   * the signature.
   */
  static encodePaymasterData(
    paymaster: string,
    validUntil: bigint,
    validAfter: bigint,
    signature: string
  ): string {
    if (validUntil < 0n || validUntil > (1n << 48n) - 1n) {
      throw new Error("validUntil does not fit in 48 bits");
    }
    if (validAfter < 0n || validAfter > (1n << 48n) - 1n) {
      throw new Error("validAfter does not fit in 48 bits");
    }
    if (signature === "0x") {
      throw new Error("a paymaster signature is required");
    }

    const until = validUntil.toString(16).padStart(12, "0");
    const after = validAfter.toString(16).padStart(12, "0");

    return getAddress(paymaster) + until + after + signature.slice(2);
  }
}

/** A JSON-RPC error returned by a bundler. */
interface JsonRpcError {
  code: number;
  message: string;
}

/**
 * A client for an ERC-4337 bundler.
 *
 * The bundler is a JSON-RPC server, so calls go over the wire with the method
 * names the specification defines. A bundler error is surfaced as an exception
 * carrying the node's own message; a failed request never becomes a fabricated
 * result.
 */
export class BundlerService {
  private readonly bundlerUrl: string;
  private readonly entryPoint: string;

  constructor(bundlerUrl: string, entryPoint: string) {
    if (!bundlerUrl) {
      throw new Error("a bundler URL is required");
    }
    if (!entryPoint) {
      throw new Error("an EntryPoint address is required");
    }

    this.bundlerUrl = bundlerUrl;
    this.entryPoint = entryPoint;
  }

  private async call<T>(method: string, params: unknown[]): Promise<T> {
    const response = await fetch(this.bundlerUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        method,
        params
      })
    });

    if (!response.ok) {
      throw new Error(`the bundler returned HTTP ${response.status}`);
    }

    const payload: { result?: T; error?: JsonRpcError } = await response.json();

    if (payload.error) {
      throw new Error(`the bundler rejected the request: ${payload.error.message}`);
    }
    if (payload.result === undefined) {
      throw new Error("the bundler returned no result");
    }

    return payload.result;
  }

  /** Submit a user operation and return its hash. */
  async sendUserOperation(userOp: UserOperationV06 | UserOperationV07): Promise<string> {
    return this.call<string>("eth_sendUserOperation", [userOp, this.entryPoint]);
  }

  /** Ask the bundler to estimate the operation's gas limits. */
  async estimateUserOperationGas(
    userOp: UserOperationV06 | UserOperationV07
  ): Promise<{ callGasLimit: bigint; verificationGasLimit: bigint; preVerificationGas: bigint }> {
    const estimate = await this.call<{
      callGasLimit: string;
      verificationGasLimit: string;
      preVerificationGas: string;
    }>("eth_estimateUserOperationGas", [userOp, this.entryPoint]);

    return {
      callGasLimit: BigInt(estimate.callGasLimit),
      verificationGasLimit: BigInt(estimate.verificationGasLimit),
      preVerificationGas: BigInt(estimate.preVerificationGas)
    };
  }

  /** Fetch a user operation receipt, or null while it is not yet mined. */
  async getUserOperationReceipt(userOpHash: string): Promise<unknown | null> {
    return this.call<unknown | null>("eth_getUserOperationReceipt", [userOpHash]);
  }
}
