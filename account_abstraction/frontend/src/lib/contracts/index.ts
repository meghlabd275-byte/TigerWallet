import { ethers } from "ethers";

export interface UserOperation {
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
}

export const EntryPointABI = [
  "function getUserOpHash((address sender, uint256 nonce, bytes initCode, bytes callData, uint256 callGasLimit, uint256 verificationGasLimit, uint256 preVerificationGas, uint256 maxFeePerGas, uint256 maxPriorityFeePerGas, bytes signature, uint256 paymasterAndData)) view returns (bytes32)",
  "function handleOps((address sender, uint256 nonce, bytes initCode, bytes callData, uint256 callGasLimit, uint256 verificationGasLimit, uint256 preVerificationGas, uint256 maxFeePerGas, uint256 maxPriorityFeePerGas, bytes signature, uint256 paymasterAndData)[], address beneficiary)",
  "function depositTo(address account)",
  "function getDepositInfo(address account) view returns (tuple(uint256 deposit, bool staked))"
];

export const SmartAccountABI = [
  "function initialize(address owner)",
  "function owner() view returns (address)",
  "function nonce() view returns (uint256)",
  "function execute(address dest, uint256 value, bytes calldata func)",
  "function executeBatch(address[] dest, uint256[] value, bytes[] func)"
];

export const AccountFactoryABI = [
  "function getInitCode(address owner) view returns (bytes)",
  "function createAccount(address owner, uint256 salt) returns (address)",
  "function getAccountAddress(address owner, uint256 salt) view returns (address)"
];

export class EntryPoint__factory {
  static connect(address: string, signerOrProvider: any) {
    return {
      address,
      interface: new ethers.Interface(EntryPointABI),
      getUserOpHash: async (userOp: UserOperation) => "0x" + "0".repeat(64)
    };
  }
}

export class SmartAccountFactory__factory {
  static connect(address: string, signerOrProvider: any) {
    return {
      address,
      interface: new ethers.Interface(AccountFactoryABI),
      getInitCode: async (owner: string) => "0x",
      createAccount: async (owner: string, salt: number) => ({ wait: async () => ({ status: 1 }) }),
      getAccountAddress: async (owner: string, salt: number) => "0x" + "0".repeat(40)
    };
  }
}
