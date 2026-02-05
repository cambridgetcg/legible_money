/**
 * DMSS-Contract Integration
 *
 * Integrates the Self-Invoking Contracts module with the DMSS Schedule module.
 * This is the bridge that enables contracts to schedule their own invocations.
 *
 * From Patent: "Self-Invoking Smart Contracts are smart contracts that are set up
 * with a schedule message and make use of SPU 131322 to trigger the execution
 * of a smart contract function with predefined parameters."
 *
 * Integration Flow:
 * 1. Contract registers invocation with DMSS
 * 2. SPU monitors schedule conditions
 * 3. When condition satisfied, SPU notifies Invoker
 * 4. Invoker queues and executes the invocation
 * 5. Results update both schedule process and contract state
 *
 * PP HIERARCHY:
 *   TRUTH: Schedule conditions are objectively verifiable
 *   UNDERSTANDING: Clear flow from schedule to execution
 *   BEAUTY: Clean separation of scheduling and execution
 *   JUSTICE: All contracts follow same scheduling rules
 *   CREATIVITY: Novel integration of scheduling with PoT consensus
 */

import { sha512_256Hex } from "../crypto/hashing.js";
import type { Signature } from "../crypto/keys.js";
import type { DID } from "../agents/identity/did.js";
import type {
  ScheduleMessage,
  ScheduleProcess,
  ScheduleCondition,
} from "../schedule/types.js";
import type { ScheduleProcessingUnit } from "../schedule/processing_unit.js";
import type {
  SelfInvokingContract,
  InvocationConfig,
  InvocationResult,
  DeploymentRequest,
  DeploymentResult,
  ContractConfig,
} from "./types.js";
import { DEFAULT_CONTRACT_CONFIG, ContractError } from "./types.js";
import { ContractRegistry } from "./registry.js";
import { SelfInvoker, ContractExecutor } from "./invoker.js";

/**
 * Integration Manager
 * Coordinates between DMSS Schedule system and Self-Invoking Contracts
 */
export class ContractIntegration {
  constructor(
    private registry: ContractRegistry,
    private invoker: SelfInvoker,
    private spu: ScheduleProcessingUnit,
    private config: ContractConfig = DEFAULT_CONTRACT_CONFIG,
  ) {}

  // ============ Contract Deployment ============

  /**
   * Deploy a self-invoking contract and register its schedules
   */
  async deployContract(
    request: DeploymentRequest,
    currentBlock: bigint,
    currentTime: number,
  ): Promise<DeploymentResult> {
    // Generate contract address
    const nonce = BigInt(Date.now()); // In production, use proper nonce
    const address = ContractRegistry.generateAddress(
      request.deployer,
      nonce,
      request.salt,
    );

    // Create contract instance
    const contract: SelfInvokingContract = {
      address,
      creator: request.deployer,
      name: request.metadata.name,
      version: request.metadata.version,
      codeHash: sha512_256Hex(request.bytecode),
      isActive: true,
      invocations: [],
      totalInvocations: 0,
      stateHash: sha512_256Hex("initial"),
      deployedAtBlock: currentBlock,
      gasLimit: this.config.maxSelfInvocationGas,
      prepaidGas: request.prepaidGas,
    };

    // Register contract
    this.registry.register(
      contract,
      request.metadata,
      [], // Tags can be added later
      currentBlock,
    );

    // Create schedule processes for each invocation config
    const scheduleProcessIds: string[] = [];

    for (const invocationSpec of request.invocations) {
      const invocationId = sha512_256Hex(
        `invocation:${address}:${invocationSpec.functionName}:${Date.now()}`,
      );

      // Create invocation config
      const invocation: InvocationConfig = {
        id: invocationId,
        functionName: invocationSpec.functionName,
        functionSelector: invocationSpec.functionSelector,
        parameters: invocationSpec.parameters,
        condition: invocationSpec.condition,
        scheduleType: invocationSpec.scheduleType,
        enabled: invocationSpec.enabled,
        maxInvocations: invocationSpec.maxInvocations,
        invocationCount: 0,
        gasLimit: invocationSpec.gasLimit,
        value: invocationSpec.value,
      };

      // Create schedule message for this invocation
      const scheduleMessage = this.createScheduleMessage(
        contract,
        invocation,
        request.deployer,
        currentTime,
      );

      // Register with SPU
      const scheduleProcess =
        await this.spu.processScheduleMessage(scheduleMessage);

      if (scheduleProcess) {
        invocation.scheduleProcessId = scheduleProcess.id;
        scheduleProcessIds.push(scheduleProcess.id);
      }

      contract.invocations.push(invocation);
    }

    // Update contract in registry with invocations
    this.registry.updateContract(address, contract);

    return {
      address,
      contract,
      scheduleProcessIds,
      transactionId: sha512_256Hex(`deploy:${address}:${currentBlock}`),
      gasUsed: 100000n, // Placeholder
      deployedAtBlock: currentBlock,
    };
  }

  /**
   * Create a schedule message for an invocation
   */
  private createScheduleMessage(
    contract: SelfInvokingContract,
    invocation: InvocationConfig,
    sender: DID,
    currentTime: number,
  ): ScheduleMessage {
    // Build call data (function selector + parameters)
    const callData = new Uint8Array(4 + invocation.parameters.length);
    callData.set(invocation.functionSelector, 0);
    callData.set(invocation.parameters, 4);

    return {
      messageType: "schedule",
      messageId: sha512_256Hex(
        `schedule:${contract.address}:${invocation.id}:${currentTime}`,
      ),
      identifier: new Uint8Array(32), // Would be properly generated
      nonce: BigInt(currentTime),
      senderAddress: sender,
      recipientAddress: contract.address,
      networkFeeUnit: 1n,
      networkFeeLimit: invocation.gasLimit,
      transferValue: invocation.value,
      data: callData,
      scheduleCondition: invocation.condition,
      signature: "" as Signature, // Placeholder
      timestamp: currentTime,
      scheduleType: invocation.scheduleType,
      modifiable: true,
      perpetual: invocation.maxInvocations === undefined,
      maxExecutions: invocation.maxInvocations,
      prepaidFee: contract.prepaidGas,
      linkedClaimId: undefined,
      linkedDisputeId: undefined,
      linkedVestingId: undefined,
    };
  }

  // ============ Invocation Handling ============

  /**
   * Handle schedule process execution (called by SPU)
   * This is the callback when a schedule condition is satisfied
   */
  async handleScheduleExecution(
    process: ScheduleProcess,
    currentBlock: bigint,
    currentTime: number,
    executor: ContractExecutor,
  ): Promise<InvocationResult | null> {
    // Find the contract and invocation for this schedule
    const { contract, invocation } = this.findContractForSchedule(process.id);

    if (!contract || !invocation) {
      // Not a contract-related schedule
      return null;
    }

    // Queue the invocation
    this.invoker.queueInvocation(
      contract,
      invocation.id,
      process.id,
      { satisfied: true }, // SPU already verified condition
      currentBlock,
    );

    // Execute immediately (or wait for batch processing)
    const context = {
      blockNumber: currentBlock,
      timestamp: currentTime,
      proposer: process.creator,
      gasPrice: 1n,
      recursionDepth: 0,
    };

    const results = await this.invoker.processPendingInvocations(
      context,
      executor,
    );
    return results[0] ?? null;
  }

  /**
   * Find contract and invocation for a schedule process
   */
  private findContractForSchedule(scheduleProcessId: string): {
    contract?: SelfInvokingContract;
    invocation?: InvocationConfig;
  } {
    for (const contract of this.registry.getAllContracts()) {
      for (const invocation of contract.invocations) {
        if (invocation.scheduleProcessId === scheduleProcessId) {
          return { contract, invocation };
        }
      }
    }
    return {};
  }

  // ============ Dynamic Schedule Management ============

  /**
   * Add a new invocation to an existing contract
   */
  async addInvocation(
    contractAddress: string,
    invocationSpec: Omit<
      InvocationConfig,
      "id" | "invocationCount" | "scheduleProcessId"
    >,
    sender: DID,
    _currentBlock: bigint,
    currentTime: number,
  ): Promise<InvocationConfig> {
    const contract = this.registry.getContract(contractAddress);
    if (!contract) {
      throw new ContractError(
        `Contract ${contractAddress} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    // Verify sender is creator
    if (contract.creator !== sender) {
      throw new ContractError(
        "Only contract creator can add invocations",
        "UNAUTHORIZED",
      );
    }

    const invocationId = sha512_256Hex(
      `invocation:${contractAddress}:${invocationSpec.functionName}:${currentTime}`,
    );

    const invocation: InvocationConfig = {
      id: invocationId,
      ...invocationSpec,
      invocationCount: 0,
    };

    // Create schedule
    const scheduleMessage = this.createScheduleMessage(
      contract,
      invocation,
      sender,
      currentTime,
    );

    const scheduleProcess =
      await this.spu.processScheduleMessage(scheduleMessage);

    if (scheduleProcess) {
      invocation.scheduleProcessId = scheduleProcess.id;
    }

    // Update contract
    contract.invocations.push(invocation);
    this.registry.updateContract(contractAddress, contract);

    return invocation;
  }

  /**
   * Disable an invocation
   */
  async disableInvocation(
    contractAddress: string,
    invocationId: string,
    sender: DID,
  ): Promise<void> {
    const contract = this.registry.getContract(contractAddress);
    if (!contract) {
      throw new ContractError(
        `Contract ${contractAddress} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    if (contract.creator !== sender) {
      throw new ContractError(
        "Only contract creator can disable invocations",
        "UNAUTHORIZED",
      );
    }

    const invocation = contract.invocations.find((i) => i.id === invocationId);
    if (!invocation) {
      throw new ContractError(
        `Invocation ${invocationId} not found`,
        "INVOCATION_NOT_FOUND",
      );
    }

    // Disable invocation
    invocation.enabled = false;

    // Cancel schedule process if exists
    if (invocation.scheduleProcessId) {
      // Would call SPU to cancel the schedule
      // this.spu.cancelProcess(invocation.scheduleProcessId);
    }

    this.registry.updateContract(contractAddress, contract);
  }

  /**
   * Update invocation condition
   */
  async updateInvocationCondition(
    contractAddress: string,
    invocationId: string,
    newCondition: ScheduleCondition,
    sender: DID,
    _currentBlock: bigint,
    _currentTime: number,
  ): Promise<void> {
    const contract = this.registry.getContract(contractAddress);
    if (!contract) {
      throw new ContractError(
        `Contract ${contractAddress} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    if (contract.creator !== sender) {
      throw new ContractError(
        "Only contract creator can update invocations",
        "UNAUTHORIZED",
      );
    }

    const invocation = contract.invocations.find((i) => i.id === invocationId);
    if (!invocation) {
      throw new ContractError(
        `Invocation ${invocationId} not found`,
        "INVOCATION_NOT_FOUND",
      );
    }

    // Update condition
    invocation.condition = newCondition;

    // Would update the schedule process with SPU
    // this.spu.updateProcessCondition(invocation.scheduleProcessId, newCondition);

    this.registry.updateContract(contractAddress, contract);
  }

  // ============ Gas Management ============

  /**
   * Add prepaid gas to a contract
   */
  addPrepaidGas(contractAddress: string, amount: bigint, _sender: DID): void {
    const contract = this.registry.getContract(contractAddress);
    if (!contract) {
      throw new ContractError(
        `Contract ${contractAddress} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    contract.prepaidGas += amount;
    this.registry.updateContract(contractAddress, contract);
  }

  /**
   * Get gas balance for a contract
   */
  getGasBalance(contractAddress: string): bigint {
    const contract = this.registry.getContract(contractAddress);
    if (!contract) {
      throw new ContractError(
        `Contract ${contractAddress} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    return contract.prepaidGas;
  }

  // ============ PoT-Specific Integration ============

  /**
   * Create a verification deadline schedule for a claim
   */
  async createVerificationDeadline(
    contractAddress: string,
    claimId: string,
    deadlineMs: number,
    currentBlock: bigint,
    currentTime: number,
  ): Promise<string> {
    const contract = this.registry.getContract(contractAddress);
    if (!contract) {
      throw new ContractError(
        `Contract ${contractAddress} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    // Find the verification timeout function
    const entry = this.registry.getEntry(contractAddress);
    const timeoutFn = entry?.metadata.abi.functions.find(
      (f) =>
        f.name === "onVerificationTimeout" ||
        f.name === "handleVerificationTimeout",
    );

    if (!timeoutFn) {
      throw new ContractError(
        "Contract does not have verification timeout function",
        "INVALID_SELECTOR",
      );
    }

    // Create invocation for the deadline
    const invocation = await this.addInvocation(
      contractAddress,
      {
        functionName: timeoutFn.name,
        functionSelector: hexToUint8Array(timeoutFn.selector),
        parameters: encodeClaimId(claimId),
        condition: {
          time: {
            type: "absolute",
            executeAt: currentTime + deadlineMs,
          },
        },
        scheduleType: "verification_deadline",
        enabled: true,
        maxInvocations: 1,
        gasLimit: 100000n,
        value: 0n,
      },
      contract.creator,
      currentBlock,
      currentTime,
    );

    return invocation.id;
  }

  /**
   * Create a challenge window close schedule
   */
  async createChallengeDeadline(
    contractAddress: string,
    disputeId: string,
    deadlineMs: number,
    currentBlock: bigint,
    currentTime: number,
  ): Promise<string> {
    const contract = this.registry.getContract(contractAddress);
    if (!contract) {
      throw new ContractError(
        `Contract ${contractAddress} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    // Create schedule for challenge window close
    const invocation = await this.addInvocation(
      contractAddress,
      {
        functionName: "onChallengeWindowClose",
        functionSelector: hexToUint8Array("0x00000000"), // Placeholder
        parameters: encodeDisputeId(disputeId),
        condition: {
          time: {
            type: "absolute",
            executeAt: currentTime + deadlineMs,
          },
        },
        scheduleType: "challenge_deadline",
        enabled: true,
        maxInvocations: 1,
        gasLimit: 100000n,
        value: 0n,
      },
      contract.creator,
      currentBlock,
      currentTime,
    );

    return invocation.id;
  }

  /**
   * Create a periodic confidence decay schedule
   */
  async createConfidenceDecaySchedule(
    contractAddress: string,
    claimId: string,
    intervalMs: number,
    currentBlock: bigint,
    currentTime: number,
  ): Promise<string> {
    const contract = this.registry.getContract(contractAddress);
    if (!contract) {
      throw new ContractError(
        `Contract ${contractAddress} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    const invocation = await this.addInvocation(
      contractAddress,
      {
        functionName: "updateConfidenceDecay",
        functionSelector: hexToUint8Array("0x00000000"), // Placeholder
        parameters: encodeClaimId(claimId),
        condition: {
          time: {
            type: "periodic",
            interval: intervalMs,
            startTime: currentTime,
          },
        },
        scheduleType: "confidence_decay",
        enabled: true,
        // No max invocations - perpetual
        gasLimit: 50000n,
        value: 0n,
      },
      contract.creator,
      currentBlock,
      currentTime,
    );

    return invocation.id;
  }

  // ============ Query Methods ============

  /**
   * Get all schedule processes for a contract
   */
  getSchedulesForContract(contractAddress: string): string[] {
    const contract = this.registry.getContract(contractAddress);
    if (!contract) {
      return [];
    }

    return contract.invocations
      .map((i) => i.scheduleProcessId)
      .filter((id): id is string => id !== undefined);
  }

  /**
   * Get all contracts with pending schedules
   */
  getContractsWithPendingSchedules(): SelfInvokingContract[] {
    return this.registry
      .getAllContracts()
      .filter((contract) =>
        contract.invocations.some((i) => i.enabled && i.scheduleProcessId),
      );
  }
}

// ============ Helper Functions ============

function hexToUint8Array(hex: string): Uint8Array {
  const cleanHex = hex.startsWith("0x") ? hex.slice(2) : hex;
  const bytes = new Uint8Array(cleanHex.length / 2);
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(cleanHex.substr(i * 2, 2), 16);
  }
  return bytes;
}

function encodeClaimId(claimId: string): Uint8Array {
  const encoder = new TextEncoder();
  return encoder.encode(claimId);
}

function encodeDisputeId(disputeId: string): Uint8Array {
  const encoder = new TextEncoder();
  return encoder.encode(disputeId);
}
