import { Contract, Interface, type ContractRunner, type InterfaceAbi } from "ethers";

/**
 * The canonical EntryPoint deployments. Both are deployed at the same address
 * on every major network, so one pair of constants covers every chain.
 */
export const ENTRY_POINT_V06 = "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789";
export const ENTRY_POINT_V07 = "0x0000000071727De22E5E9d8BAf0edAc6f37da032";

/** A user operation as the v0.6 EntryPoint expects it. */
export interface UserOperationV06 {
  sender: string;
  nonce: bigint;
  initCode: string;
  callData: string;
  callGasLimit: bigint;
  verificationGasLimit: bigint;
  preVerificationGas: bigint;
  maxFeePerGas: bigint;
  maxPriorityFeePerGas: bigint;
  paymasterAndData: string;
  signature: string;
}

/** A packed user operation as the v0.7 EntryPoint expects it. */
export interface UserOperationV07 {
  sender: string;
  nonce: bigint;
  initCode: string;
  callData: string;
  accountGasLimits: string;
  preVerificationGas: bigint;
  gasFees: string;
  paymasterAndData: string;
  signature: string;
}

/** The v0.6 EntryPoint keeps one word per gas field. */
export const EntryPointV06ABI: InterfaceAbi = [
  "function getUserOpHash((address sender, uint256 nonce, bytes initCode, bytes callData, uint256 callGasLimit, uint256 verificationGasLimit, uint256 preVerificationGas, uint256 maxFeePerGas, uint256 maxPriorityFeePerGas, bytes paymasterAndData, bytes signature) userOp) view returns (bytes32)",
  "function handleOps((address sender, uint256 nonce, bytes initCode, bytes callData, uint256 callGasLimit, uint256 verificationGasLimit, uint256 preVerificationGas, uint256 maxFeePerGas, uint256 maxPriorityFeePerGas, bytes paymasterAndData, bytes signature)[] ops, address beneficiary)",
  "function getNonce(address sender, uint192 key) view returns (uint256)",
  "function depositTo(address account) payable",
  "function balanceOf(address account) view returns (uint256)",
  "function getDepositInfo(address account) view returns (uint112 deposit, bool staked, uint112 stake, uint32 unstakeDelaySec, uint48 withdrawTime)"
];

/**
 * The v0.7 EntryPoint packs the gas fields into two words, so its tuple has
 * fewer members.
 */
export const EntryPointV07ABI: InterfaceAbi = [
  "function getUserOpHash((address sender, uint256 nonce, bytes initCode, bytes callData, bytes32 accountGasLimits, uint256 preVerificationGas, bytes32 gasFees, bytes paymasterAndData, bytes signature) userOp) view returns (bytes32)",
  "function handleOps((address sender, uint256 nonce, bytes initCode, bytes callData, bytes32 accountGasLimits, uint256 preVerificationGas, bytes32 gasFees, bytes paymasterAndData, bytes signature)[] ops, address payable beneficiary)",
  "function getNonce(address sender, uint192 key) view returns (uint256)",
  "function depositTo(address account) payable",
  "function balanceOf(address account) view returns (uint256)",
  "function getDepositInfo(address account) view returns (uint256 deposit, bool staked, uint112 stake, uint32 unstakeDelaySec, uint48 withdrawTime)"
];

/** The canonical `SimpleAccountFactory`. */
export const SimpleAccountFactoryABI: InterfaceAbi = [
  "function getAddress(address owner, uint256 salt) view returns (address)",
  "function createAccount(address owner, uint256 salt) returns (address)",
  "function accountImplementation() view returns (address)"
];

/**
 * TigerWallet's own account factory. It takes a `key` instead of a `salt` and
 * exposes the counterfactual address through `getAccountAddress`.
 */
export const TigerWalletFactoryABI: InterfaceAbi = [
  "function getAccountAddress(address owner, uint256 key) view returns (address)",
  "function createAccount(address owner, uint256 key) returns (address)",
  "function accountImplementation() view returns (address)",
  "function entryPoint() view returns (address)"
];

/**
 * TigerWallet's own smart account. It is a v0.7 `BaseAccount` owned by a single
 * EOA, with an owner-only direct execution path.
 */
export const TigerWalletAccountABI: InterfaceAbi = [
  "function owner() view returns (address)",
  "function entryPoint() view returns (address)",
  "function executeFromOwner(address target, uint256 value, bytes data) returns (bool, bytes)",
  "function transferOwnership(address newOwner)"
];

/** The canonical `SimpleAccount`. */
export const SimpleAccountABI: InterfaceAbi = [
  "function initialize(address owner)",
  "function owner() view returns (address)",
  "function entryPoint() view returns (address)",
  "function execute(address dest, uint256 value, bytes func)",
  "function executeBatch(address[] dest, uint256[] value, bytes[] func)",
  "function addDeposit() payable",
  "function withdrawDepositTo(address payable withdrawAddress, uint256 amount)"
];

/** True when the address is the v0.6 EntryPoint. */
export function isEntryPointV06(entryPoint: string): boolean {
  return entryPoint.toLowerCase() === ENTRY_POINT_V06.toLowerCase();
}

/**
 * Which account factory a deployment uses.
 *
 * `canonical` is the SimpleAccountFactory shipped with the EntryPoint;
 * `tigerwallet` is TigerWallet's own factory. They share the
 * `createAccount(address,uint256)` entry point but expose the counterfactual
 * address under different names, so callers must say which one they use.
 */
export type FactoryMode = "canonical" | "tigerwallet";

/** The ABI matching a factory flavour. */
export function factoryAbi(mode: FactoryMode): InterfaceAbi {
  return mode === "tigerwallet" ? TigerWalletFactoryABI : SimpleAccountFactoryABI;
}

/** The method that returns a factory's counterfactual address. */
export function factoryAddressMethod(mode: FactoryMode): string {
  return mode === "tigerwallet" ? "getAccountAddress" : "getAddress";
}

/** A typed factory binding. */
export function accountFactoryContract(
  factory: string,
  mode: FactoryMode,
  runner: ContractRunner
): Contract {
  return new Contract(factory, factoryAbi(mode), runner);
}

/** The ABI matching an EntryPoint address. */
export function entryPointAbi(entryPoint: string): InterfaceAbi {
  return isEntryPointV06(entryPoint) ? EntryPointV06ABI : EntryPointV07ABI;
}

/**
 * A typed EntryPoint binding.
 *
 * `runner` may be a provider for reads or a signer for writes.
 */
export function entryPointContract(entryPoint: string, runner: ContractRunner): Contract {
  return new Contract(entryPoint, entryPointAbi(entryPoint), runner);
}

/** A typed `SimpleAccountFactory` binding. */
export function simpleAccountFactoryContract(factory: string, runner: ContractRunner): Contract {
  return new Contract(factory, SimpleAccountFactoryABI, runner);
}

/** A typed `SimpleAccount` binding. */
export function simpleAccountContract(account: string, runner: ContractRunner): Contract {
  return new Contract(account, SimpleAccountABI, runner);
}

/**
 * Build the `initCode` for an undeployed account: the factory address followed
 * by its `createAccount(owner, salt)` calldata.
 */
export function buildInitCode(factory: string, owner: string, salt: bigint): string {
  const iface = new Interface(SimpleAccountFactoryABI);
  const call = iface.encodeFunctionData("createAccount", [owner, salt]);

  return factory + call.slice(2);
}

/**
 * Pack a verification and a call gas limit into the v0.7 `accountGasLimits`
 * word: verification high, call low.
 */
export function packAccountGasLimits(verificationGasLimit: bigint, callGasLimit: bigint): string {
  return "0x" + toWord(verificationGasLimit) + toWord(callGasLimit);
}

/**
 * Pack a priority and a max fee into the v0.7 `gasFees` word: priority high,
 * max low.
 */
export function packGasFees(maxPriorityFeePerGas: bigint, maxFeePerGas: bigint): string {
  return "0x" + toWord(maxPriorityFeePerGas) + toWord(maxFeePerGas);
}

/** A 16-byte big-endian word, as 32 hex digits. */
function toWord(value: bigint): string {
  if (value < 0n) {
    throw new Error("a gas value cannot be negative");
  }
  if (value > (1n << 128n) - 1n) {
    throw new Error("a gas value does not fit in 128 bits");
  }

  return value.toString(16).padStart(32, "0");
}
