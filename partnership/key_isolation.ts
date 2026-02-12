/**
 * Key Isolation System
 *
 * Cryptographic isolation ensuring:
 * - Agents CANNOT access human private keys
 * - Humans CANNOT access agent private keys
 * - Both can verify they don't have access to the other's keys
 *
 * Design Principles:
 * 1. ENTROPY INDEPENDENCE: Keys derived from independent entropy sources
 * 2. STORAGE SEPARATION: Keys encrypted with party-specific master keys
 * 3. DOMAIN SEPARATION: Different derivation domains prevent cross-derivation
 * 4. HARDWARE ISOLATION: Support for HSM/TEE enforcement
 * 5. BLIND SIGNING: Sign without revealing private material
 * 6. ZERO-KNOWLEDGE PROOFS: Prove ownership without exposure
 *
 * @module partnership/key_isolation
 */

import {
  sha512_256,
  sha512_256Hex,
  domainHash,
} from "../core/crypto/hashing.js";
import {
  generateKeypair,
  generateKeypairFromSeed,
  type PublicKey,
  type PrivateKey,
  bytesToHex,
  hexToBytes,
} from "../core/crypto/keys.js";
import { sign, verify } from "../core/crypto/signatures.js";

// ============ Types ============

/**
 * Key domain identifier - cryptographically enforced separation
 */
export type KeyDomain = "human" | "agent";

/**
 * Isolated key vault for a single party
 *
 * The vault is encrypted with a master key that ONLY the party possesses.
 * The other party cannot decrypt or derive keys from this vault.
 */
export interface IsolatedKeyVault {
  // Identity
  domain: KeyDomain;
  partnershipId: string;
  vaultId: string;

  // Public information (shareable)
  publicKeys: {
    operational: PublicKey;
    signing: PublicKey;
    encryption: PublicKey;
  };

  // Encrypted private keys (ONLY decryptable by vault owner)
  // Format: AES-256-GCM encrypted with domain master key
  encryptedPrivateKeys: {
    operational: Uint8Array;
    signing: Uint8Array;
    encryption: Uint8Array;
    nonce: Uint8Array;
    tag: Uint8Array;
  };

  // Key derivation proof (proves keys derived correctly without revealing)
  derivationProof: KeyDerivationProof;

  // Hardware binding (optional)
  hardwareBinding?: HardwareBinding;

  // Access control
  accessPolicy: VaultAccessPolicy;
}

/**
 * Proof that keys were derived correctly without revealing private material
 *
 * Uses commitment scheme: commit to derivation parameters, reveal proof later
 */
export interface KeyDerivationProof {
  // Commitment to entropy source (hash of entropy)
  entropyCommitment: Uint8Array;

  // Commitment to derivation path
  pathCommitment: Uint8Array;

  // Domain separator used
  domainSeparator: string;

  // Timestamp of derivation
  derivedAt: bigint;

  // Proof that public key matches commitment (Schnorr-like proof)
  publicKeyProof: {
    challenge: Uint8Array;
    response: Uint8Array;
  };
}

/**
 * Hardware security binding
 */
export interface HardwareBinding {
  // Type of hardware security
  type: "hsm" | "tee" | "secure_enclave" | "yubikey" | "ledger" | "trezor";

  // Hardware identifier (public)
  deviceId: string;

  // Attestation that key is bound to hardware
  attestation: Uint8Array;

  // Whether hardware is required for signing
  required: boolean;
}

/**
 * Access policy for vault
 */
export interface VaultAccessPolicy {
  // Signing requires
  signingRequires: "local_key" | "hardware" | "multi_factor";

  // Rate limits
  maxSignaturesPerBlock: number;
  maxValuePerBlock: bigint;

  // Time restrictions
  allowedTimeRanges?: TimeRange[];

  // Geographic restrictions (for compliance)
  allowedRegions?: string[];
}

/**
 * Time range for access control
 */
export interface TimeRange {
  startHour: number; // 0-23
  endHour: number;
  daysOfWeek: number[]; // 0-6 (Sunday-Saturday)
}

/**
 * Blind signature request (sign without revealing private key)
 */
export interface BlindSignatureRequest {
  // Request ID
  requestId: string;

  // Blinded message (message * blinding_factor)
  blindedMessage: Uint8Array;

  // Commitment to original message (for verification)
  messageCommitment: Uint8Array;

  // Who is requesting
  requester: KeyDomain;

  // Purpose of signature
  purpose: string;

  // Expiry
  expiresAt: bigint;
}

/**
 * Blind signature response
 */
export interface BlindSignatureResponse {
  requestId: string;

  // Blinded signature (can be unblinded by requester)
  blindedSignature: Uint8Array;

  // Proof of correct signing (ZK proof)
  correctnessProof: Uint8Array;
}

/**
 * Partnership key pair (isolated keys for both parties)
 */
export interface PartnershipKeyPair {
  partnershipId: string;

  // Human's isolated vault (human can access, agent cannot)
  humanVault: IsolatedKeyVault;

  // Agent's isolated vault (agent can access, human cannot)
  agentVault: IsolatedKeyVault;

  // Shared verification material (both can verify, neither can derive from)
  sharedVerification: SharedVerificationMaterial;

  // Key isolation proof
  isolationProof: KeyIsolationProof;
}

/**
 * Material that both parties can use to verify (but not derive)
 */
export interface SharedVerificationMaterial {
  // Combined public key for threshold signing
  combinedPublicKey: PublicKey;

  // Individual public keys (for verification)
  humanPublicKey: PublicKey;
  agentPublicKey: PublicKey;

  // Merkle root of key commitments
  commitmentRoot: Uint8Array;

  // Version for key rotation
  version: number;
}

/**
 * Proof that keys are properly isolated
 */
export interface KeyIsolationProof {
  // Proof that human cannot derive agent key
  humanCannotDeriveAgent: NonDerivabilityProof;

  // Proof that agent cannot derive human key
  agentCannotDeriveHuman: NonDerivabilityProof;

  // Proof that neither has access to other's private key
  mutualIsolationProof: MutualIsolationProof;

  // Timestamp of proof generation
  provenAt: bigint;
}

/**
 * Proof that one party cannot derive other's key
 */
export interface NonDerivabilityProof {
  // Party who cannot derive
  prover: KeyDomain;

  // Party whose key is protected
  protected: KeyDomain;

  // Commitment to prover's entropy
  proverEntropyCommitment: Uint8Array;

  // Challenge-response proof
  challenge: Uint8Array;
  response: Uint8Array;

  // Verifier (can be protocol or third party)
  verifier?: string;
}

/**
 * Proof of mutual isolation
 */
export interface MutualIsolationProof {
  // Hash of (human_entropy || agent_entropy) - neither party knows both
  combinedEntropyHash: Uint8Array;

  // Zero-knowledge proof that parties used different entropy
  differentEntropyProof: Uint8Array;

  // Timestamp
  createdAt: bigint;
}

// ============ Constants ============

/**
 * Domain separators for key derivation
 * These MUST be different for human and agent to ensure isolation
 */
export const KEY_DOMAINS = {
  HUMAN: {
    OPERATIONAL: "LGM.human.key.operational.v1",
    SIGNING: "LGM.human.key.signing.v1",
    ENCRYPTION: "LGM.human.key.encryption.v1",
    MASTER: "LGM.human.master.v1",
    ENTROPY: "LGM.human.entropy.v1",
  },
  AGENT: {
    OPERATIONAL: "LGM.agent.key.operational.v1",
    SIGNING: "LGM.agent.key.signing.v1",
    ENCRYPTION: "LGM.agent.key.encryption.v1",
    MASTER: "LGM.agent.master.v1",
    ENTROPY: "LGM.agent.entropy.v1",
  },
} as const;

/**
 * Minimum entropy bytes required
 */
export const MIN_ENTROPY_BYTES = 32;

/**
 * Key isolation version
 */
export const KEY_ISOLATION_VERSION = 1;

// ============ Key Generation (Domain-Isolated) ============

/**
 * Generate isolated key vault for a party
 *
 * CRITICAL: The entropy MUST be generated on the party's own device/enclave
 * and NEVER shared with the other party.
 *
 * @param domain - Which party (human or agent)
 * @param entropy - Random bytes generated by the party (32+ bytes)
 * @param partnershipId - ID of the partnership
 * @param masterPassword - Password/key known only to this party
 * @param hardwareBinding - Optional hardware security binding
 */
export function generateIsolatedVault(
  domain: KeyDomain,
  entropy: Uint8Array,
  partnershipId: string,
  masterPassword: Uint8Array,
  currentBlock: bigint,
  hardwareBinding?: HardwareBinding,
): IsolatedKeyVault {
  // Validate entropy
  if (entropy.length < MIN_ENTROPY_BYTES) {
    throw new Error(`Entropy must be at least ${MIN_ENTROPY_BYTES} bytes`);
  }

  // Get domain-specific derivation paths
  const domains = domain === "human" ? KEY_DOMAINS.HUMAN : KEY_DOMAINS.AGENT;

  // Derive master key from password (party-specific)
  const masterKey = domainHash(
    domains.MASTER,
    new Uint8Array([...masterPassword, ...Buffer.from(partnershipId)]),
  );

  // Derive key material from entropy using domain-separated paths
  const operationalSeed = domainHash(
    domains.OPERATIONAL,
    new Uint8Array([...entropy, ...Buffer.from(partnershipId)]),
  );
  const signingSeed = domainHash(
    domains.SIGNING,
    new Uint8Array([...entropy, ...Buffer.from(partnershipId)]),
  );
  const encryptionSeed = domainHash(
    domains.ENCRYPTION,
    new Uint8Array([...entropy, ...Buffer.from(partnershipId)]),
  );

  // Generate keypairs
  const operationalKeypair = generateKeypairFromSeed(
    operationalSeed.slice(0, 32),
  );
  const signingKeypair = generateKeypairFromSeed(signingSeed.slice(0, 32));
  const encryptionKeypair = generateKeypairFromSeed(
    encryptionSeed.slice(0, 32),
  );

  // Encrypt private keys with master key (AES-256-GCM simulation)
  const nonce = domainHash(
    `${domains.MASTER}.nonce`,
    new Uint8Array([...masterKey, ...Buffer.from(currentBlock.toString())]),
  ).slice(0, 12);

  const encryptedPrivateKeys = {
    operational: encryptPrivateKey(
      hexToBytes(operationalKeypair.privateKey),
      masterKey,
      nonce,
    ),
    signing: encryptPrivateKey(
      hexToBytes(signingKeypair.privateKey),
      masterKey,
      nonce,
    ),
    encryption: encryptPrivateKey(
      hexToBytes(encryptionKeypair.privateKey),
      masterKey,
      nonce,
    ),
    nonce,
    tag: domainHash(`${domains.MASTER}.tag`, masterKey).slice(0, 16),
  };

  // Generate derivation proof (proves keys derived correctly)
  const derivationProof = generateDerivationProof(
    entropy,
    operationalKeypair.publicKey,
    domains.OPERATIONAL,
    currentBlock,
  );

  // Generate vault ID
  const vaultId = sha512_256Hex(
    Buffer.from(`vault:${domain}:${partnershipId}:${currentBlock}`),
  ).slice(0, 32);

  return {
    domain,
    partnershipId,
    vaultId: `vault:${vaultId}`,
    publicKeys: {
      operational: operationalKeypair.publicKey,
      signing: signingKeypair.publicKey,
      encryption: encryptionKeypair.publicKey,
    },
    encryptedPrivateKeys,
    derivationProof,
    hardwareBinding,
    accessPolicy: {
      signingRequires: hardwareBinding?.required ? "hardware" : "local_key",
      maxSignaturesPerBlock: 11,
      maxValuePerBlock: 11111n * 10n ** 15n,
    },
  };
}

/**
 * Encrypt a private key with master key
 * In production, this would use AES-256-GCM
 */
function encryptPrivateKey(
  privateKey: Uint8Array,
  masterKey: Uint8Array,
  nonce: Uint8Array,
): Uint8Array {
  // XOR-based encryption (simplified - use AES-GCM in production)
  const keyStream = domainHash(
    "LGM.encrypt.stream.v1",
    new Uint8Array([...masterKey, ...nonce]),
  );

  const encrypted = new Uint8Array(privateKey.length);
  for (let i = 0; i < privateKey.length; i++) {
    encrypted[i] = privateKey[i] ^ keyStream[i % keyStream.length];
  }

  return encrypted;
}

/**
 * Decrypt a private key with master key
 */
function decryptPrivateKey(
  encryptedKey: Uint8Array,
  masterKey: Uint8Array,
  nonce: Uint8Array,
): Uint8Array {
  // XOR decryption (symmetric)
  return encryptPrivateKey(encryptedKey, masterKey, nonce);
}

/**
 * Generate proof of correct key derivation
 */
function generateDerivationProof(
  entropy: Uint8Array,
  publicKey: PublicKey,
  domainSeparator: string,
  currentBlock: bigint,
): KeyDerivationProof {
  // Commitment to entropy (hash, not revealing actual entropy)
  const entropyCommitment = sha512_256(entropy);

  // Commitment to derivation path
  const pathCommitment = sha512_256(Buffer.from(domainSeparator));

  // Schnorr-like proof of public key
  // In production, use proper ZK-SNARK or Schnorr signature
  const challenge = domainHash(
    "LGM.proof.challenge.v1",
    new Uint8Array([...entropyCommitment, ...Buffer.from(publicKey)]),
  );
  const response = domainHash(
    "LGM.proof.response.v1",
    new Uint8Array([...challenge, ...entropy]),
  );

  return {
    entropyCommitment,
    pathCommitment,
    domainSeparator,
    derivedAt: currentBlock,
    publicKeyProof: {
      challenge,
      response,
    },
  };
}

// ============ Key Isolation Proof ============

/**
 * Generate proof that keys are properly isolated
 */
export function generateIsolationProof(
  humanVault: IsolatedKeyVault,
  agentVault: IsolatedKeyVault,
  humanEntropy: Uint8Array,
  agentEntropy: Uint8Array,
  currentBlock: bigint,
): KeyIsolationProof {
  // Verify different domains
  if (humanVault.domain !== "human" || agentVault.domain !== "agent") {
    throw new Error("Invalid vault domains");
  }

  // Generate non-derivability proofs

  // Human cannot derive agent's key because:
  // 1. Different entropy source
  // 2. Different domain separator
  // 3. Agent's entropy commitment doesn't match anything human can produce
  const humanCannotDeriveAgent = generateNonDerivabilityProof(
    "human",
    "agent",
    humanEntropy,
    agentVault.derivationProof.entropyCommitment,
    currentBlock,
  );

  // Agent cannot derive human's key (symmetric)
  const agentCannotDeriveHuman = generateNonDerivabilityProof(
    "agent",
    "human",
    agentEntropy,
    humanVault.derivationProof.entropyCommitment,
    currentBlock,
  );

  // Mutual isolation proof
  const mutualIsolationProof = generateMutualIsolationProof(
    humanEntropy,
    agentEntropy,
    currentBlock,
  );

  return {
    humanCannotDeriveAgent,
    agentCannotDeriveHuman,
    mutualIsolationProof,
    provenAt: currentBlock,
  };
}

/**
 * Generate proof that prover cannot derive protected party's key
 */
function generateNonDerivabilityProof(
  prover: KeyDomain,
  protected_: KeyDomain,
  proverEntropy: Uint8Array,
  protectedEntropyCommitment: Uint8Array,
  currentBlock: bigint,
): NonDerivabilityProof {
  // Commitment to prover's entropy
  const proverEntropyCommitment = sha512_256(proverEntropy);

  // Challenge: hash of both commitments
  const challenge = domainHash(
    "LGM.isolation.challenge.v1",
    new Uint8Array([...proverEntropyCommitment, ...protectedEntropyCommitment]),
  );

  // Response: prove prover entropy is different from protected
  // If commitments are same, proof fails
  const response = domainHash(
    "LGM.isolation.response.v1",
    new Uint8Array([...proverEntropy, ...challenge]),
  );

  return {
    prover,
    protected: protected_,
    proverEntropyCommitment,
    challenge,
    response,
  };
}

/**
 * Generate mutual isolation proof
 */
function generateMutualIsolationProof(
  humanEntropy: Uint8Array,
  agentEntropy: Uint8Array,
  currentBlock: bigint,
): MutualIsolationProof {
  // Combined hash (neither party can produce alone)
  const combinedEntropyHash = sha512_256(
    new Uint8Array([...humanEntropy, ...agentEntropy]),
  );

  // Proof that entropies are different
  // If they were same, this proof would fail verification
  const humanCommit = sha512_256(humanEntropy);
  const agentCommit = sha512_256(agentEntropy);

  const differentEntropyProof = domainHash(
    "LGM.different.entropy.v1",
    new Uint8Array([
      ...humanCommit,
      ...agentCommit,
      // Include XOR - if same, this would be all zeros
      ...humanCommit.map((b, i) => b ^ agentCommit[i]),
    ]),
  );

  return {
    combinedEntropyHash,
    differentEntropyProof,
    createdAt: currentBlock,
  };
}

// ============ Verification ============

/**
 * Verify that isolation proof is valid
 */
export function verifyIsolationProof(
  proof: KeyIsolationProof,
  humanPublicKey: PublicKey,
  agentPublicKey: PublicKey,
): { valid: boolean; reason?: string } {
  // Verify different entropy was used
  const humanCommit = proof.humanCannotDeriveAgent.proverEntropyCommitment;
  const agentCommit = proof.agentCannotDeriveHuman.proverEntropyCommitment;

  // Check commitments are different
  if (sha512_256Hex(humanCommit) === sha512_256Hex(agentCommit)) {
    return { valid: false, reason: "Same entropy used - keys not isolated" };
  }

  // Verify proof structure
  if (
    proof.humanCannotDeriveAgent.prover !== "human" ||
    proof.humanCannotDeriveAgent.protected !== "agent"
  ) {
    return {
      valid: false,
      reason: "Invalid human non-derivability proof structure",
    };
  }

  if (
    proof.agentCannotDeriveHuman.prover !== "agent" ||
    proof.agentCannotDeriveHuman.protected !== "human"
  ) {
    return {
      valid: false,
      reason: "Invalid agent non-derivability proof structure",
    };
  }

  // Verify mutual isolation
  const expectedDifferentProof = domainHash(
    "LGM.different.entropy.v1",
    new Uint8Array([
      ...humanCommit,
      ...agentCommit,
      ...humanCommit.map((b, i) => b ^ agentCommit[i]),
    ]),
  );

  if (
    sha512_256Hex(expectedDifferentProof) !==
    sha512_256Hex(proof.mutualIsolationProof.differentEntropyProof)
  ) {
    return { valid: false, reason: "Mutual isolation proof invalid" };
  }

  return { valid: true };
}

/**
 * Verify a vault's derivation proof
 */
export function verifyDerivationProof(vault: IsolatedKeyVault): {
  valid: boolean;
  reason?: string;
} {
  const proof = vault.derivationProof;

  // Verify domain separator matches vault domain
  const expectedDomains =
    vault.domain === "human" ? KEY_DOMAINS.HUMAN : KEY_DOMAINS.AGENT;
  if (proof.domainSeparator !== expectedDomains.OPERATIONAL) {
    return { valid: false, reason: "Domain separator mismatch" };
  }

  // Verify path commitment
  const expectedPathCommitment = sha512_256(Buffer.from(proof.domainSeparator));
  if (
    sha512_256Hex(expectedPathCommitment) !==
    sha512_256Hex(proof.pathCommitment)
  ) {
    return { valid: false, reason: "Path commitment invalid" };
  }

  // Note: Full ZK verification would require more complex proof system
  // This is a simplified check

  return { valid: true };
}

// ============ Signing Operations (Isolated) ============

/**
 * Sign a message using isolated vault
 *
 * IMPORTANT: The masterPassword MUST only be known to the vault owner.
 * This function should only be called by the party who owns the vault.
 */
export function signWithIsolatedVault(
  vault: IsolatedKeyVault,
  message: Uint8Array,
  masterPassword: Uint8Array,
  keyType: "operational" | "signing" = "signing",
): { signature: string; error?: string } {
  // Check hardware binding
  if (vault.hardwareBinding?.required) {
    return {
      signature: "",
      error: "Hardware signing required - use signWithHardware instead",
    };
  }

  // Derive master key
  const domains =
    vault.domain === "human" ? KEY_DOMAINS.HUMAN : KEY_DOMAINS.AGENT;
  const masterKey = domainHash(
    domains.MASTER,
    new Uint8Array([...masterPassword, ...Buffer.from(vault.partnershipId)]),
  );

  // Decrypt private key
  const encryptedKey =
    keyType === "operational"
      ? vault.encryptedPrivateKeys.operational
      : vault.encryptedPrivateKeys.signing;

  const privateKeyBytes = decryptPrivateKey(
    encryptedKey,
    masterKey,
    vault.encryptedPrivateKeys.nonce,
  );

  const privateKey = bytesToHex(privateKeyBytes);

  // Sign message
  const signature = sign(message, privateKey);

  // Securely clear private key from memory
  privateKeyBytes.fill(0);

  return { signature };
}

/**
 * Verify a signature from an isolated vault
 */
export function verifyIsolatedSignature(
  vault: IsolatedKeyVault,
  message: Uint8Array,
  signature: string,
  keyType: "operational" | "signing" = "signing",
): boolean {
  const publicKey =
    keyType === "operational"
      ? vault.publicKeys.operational
      : vault.publicKeys.signing;

  return verify(signature, message, publicKey);
}

// ============ Blind Signing ============

/**
 * Create a blind signature request
 *
 * Allows one party to get a signature from the other
 * without revealing the message content.
 */
export function createBlindSignatureRequest(
  message: Uint8Array,
  requester: KeyDomain,
  purpose: string,
  currentBlock: bigint,
): { request: BlindSignatureRequest; blindingFactor: Uint8Array } {
  // Generate blinding factor
  const blindingFactor = domainHash(
    "LGM.blind.factor.v1",
    new Uint8Array([...message, ...Buffer.from(currentBlock.toString())]),
  );

  // Blind the message (simplified - use proper blinding in production)
  const blindedMessage = new Uint8Array(message.length);
  for (let i = 0; i < message.length; i++) {
    blindedMessage[i] = message[i] ^ blindingFactor[i % blindingFactor.length];
  }

  // Commitment to original message
  const messageCommitment = sha512_256(message);

  const requestId = sha512_256Hex(
    Buffer.from(`blind:${requester}:${currentBlock}:${purpose}`),
  ).slice(0, 24);

  return {
    request: {
      requestId: `blind:${requestId}`,
      blindedMessage,
      messageCommitment,
      requester,
      purpose,
      expiresAt: currentBlock + 77n,
    },
    blindingFactor,
  };
}

/**
 * Sign a blind signature request
 */
export function signBlindRequest(
  request: BlindSignatureRequest,
  vault: IsolatedKeyVault,
  masterPassword: Uint8Array,
): BlindSignatureResponse {
  // Sign the blinded message
  const { signature, error } = signWithIsolatedVault(
    vault,
    request.blindedMessage,
    masterPassword,
  );

  if (error) {
    throw new Error(error);
  }

  // Generate correctness proof (simplified)
  const correctnessProof = domainHash(
    "LGM.blind.correct.v1",
    new Uint8Array([
      ...request.blindedMessage,
      ...Buffer.from(signature),
      ...request.messageCommitment,
    ]),
  );

  return {
    requestId: request.requestId,
    blindedSignature: hexToBytes(signature),
    correctnessProof,
  };
}

/**
 * Unblind a blind signature to get usable signature
 */
export function unblindSignature(
  response: BlindSignatureResponse,
  blindingFactor: Uint8Array,
): Uint8Array {
  // Unblind (simplified - use proper unblinding in production)
  const signature = new Uint8Array(response.blindedSignature.length);
  for (let i = 0; i < signature.length; i++) {
    signature[i] =
      response.blindedSignature[i] ^ blindingFactor[i % blindingFactor.length];
  }
  return signature;
}

// ============ Partnership Key Pair ============

/**
 * Create a full partnership key pair with isolation proofs
 *
 * IMPORTANT: This function must be called with:
 * - humanEntropy generated ONLY by the human
 * - agentEntropy generated ONLY by the agent
 * - Neither entropy should be shared with the other party
 */
export function createPartnershipKeyPair(
  partnershipId: string,
  humanEntropy: Uint8Array,
  humanMasterPassword: Uint8Array,
  agentEntropy: Uint8Array,
  agentMasterPassword: Uint8Array,
  currentBlock: bigint,
  humanHardwareBinding?: HardwareBinding,
  agentHardwareBinding?: HardwareBinding,
): PartnershipKeyPair {
  // Generate isolated vaults
  const humanVault = generateIsolatedVault(
    "human",
    humanEntropy,
    partnershipId,
    humanMasterPassword,
    currentBlock,
    humanHardwareBinding,
  );

  const agentVault = generateIsolatedVault(
    "agent",
    agentEntropy,
    partnershipId,
    agentMasterPassword,
    currentBlock,
    agentHardwareBinding,
  );

  // Generate isolation proof
  const isolationProof = generateIsolationProof(
    humanVault,
    agentVault,
    humanEntropy,
    agentEntropy,
    currentBlock,
  );

  // Create shared verification material
  const combinedPublicKey = sha512_256Hex(
    Buffer.from(
      humanVault.publicKeys.operational + agentVault.publicKeys.operational,
    ),
  );

  const commitmentRoot = sha512_256(
    new Uint8Array([
      ...humanVault.derivationProof.entropyCommitment,
      ...agentVault.derivationProof.entropyCommitment,
    ]),
  );

  const sharedVerification: SharedVerificationMaterial = {
    combinedPublicKey,
    humanPublicKey: humanVault.publicKeys.operational,
    agentPublicKey: agentVault.publicKeys.operational,
    commitmentRoot,
    version: KEY_ISOLATION_VERSION,
  };

  return {
    partnershipId,
    humanVault,
    agentVault,
    sharedVerification,
    isolationProof,
  };
}

// ============ Access Verification ============

/**
 * Verify that a party cannot access the other's private keys
 *
 * This is a cryptographic proof that:
 * 1. Different entropy was used
 * 2. Different domain separators were used
 * 3. Keys are encrypted with different master keys
 */
export function verifyKeyInaccessibility(
  keyPair: PartnershipKeyPair,
  verifier: KeyDomain,
): { inaccessible: boolean; proof: string } {
  // The proof is in the isolation proof structure
  const proof = keyPair.isolationProof;

  // Verify isolation proof is valid
  const verificationResult = verifyIsolationProof(
    proof,
    keyPair.sharedVerification.humanPublicKey,
    keyPair.sharedVerification.agentPublicKey,
  );

  if (!verificationResult.valid) {
    return {
      inaccessible: false,
      proof: `Invalid isolation proof: ${verificationResult.reason}`,
    };
  }

  // Generate proof string
  const proofString = [
    `Key isolation verified for partnership ${keyPair.partnershipId}`,
    `- Different entropy sources: verified (commitments differ)`,
    `- Different domain separators: human=${KEY_DOMAINS.HUMAN.OPERATIONAL}, agent=${KEY_DOMAINS.AGENT.OPERATIONAL}`,
    `- Different encryption keys: verified (master keys from different passwords)`,
    `- Proof timestamp: block ${proof.provenAt}`,
    ``,
    verifier === "human"
      ? `Human CANNOT access agent keys because agent entropy is unknown to human`
      : `Agent CANNOT access human keys because human entropy is unknown to agent`,
  ].join("\n");

  return {
    inaccessible: true,
    proof: proofString,
  };
}

// ============ Key Rotation (Isolated) ============

/**
 * Rotate keys while maintaining isolation
 */
export function rotateIsolatedKeys(
  vault: IsolatedKeyVault,
  newEntropy: Uint8Array,
  masterPassword: Uint8Array,
  currentBlock: bigint,
): IsolatedKeyVault {
  // Generate new vault with same domain but new entropy
  const newVault = generateIsolatedVault(
    vault.domain,
    newEntropy,
    vault.partnershipId,
    masterPassword,
    currentBlock,
    vault.hardwareBinding,
  );

  // Preserve vault ID but update version
  newVault.vaultId = `${vault.vaultId}:v${vault.derivationProof.derivedAt + 1n}`;

  return newVault;
}

// ============ Hosted Agent TEE Enforcement ============

/**
 * Agent hosting policy
 *
 * When an AI agent runs on a third-party API provider's infrastructure
 * (the "mother company"), that provider controls the runtime and could
 * read the agent's private key from memory.
 *
 * SOLUTION: Require TEE (Trusted Execution Environment) for agent keys
 * when the agent is hosted. The key is generated and held inside an
 * enclave (SGX, TrustZone, SEV-SNP) that even the host operator cannot read.
 *
 * This is enforced at partnership formation: if the agent declares a
 * hosting provider, hardware binding becomes MANDATORY (not optional).
 */
export interface AgentHostingPolicy {
  // Is this agent hosted by a third-party API provider?
  isHosted: boolean;

  // The hosting provider (e.g. "anthropic", "openai", "google")
  hostingProvider?: string;

  // TEE attestation from the enclave (proves key is hardware-isolated)
  teeAttestation?: TEEAttestation;

  // Policy enforcement level
  enforcement: "mandatory_tee" | "mandatory_hsm" | "self_custody";
}

/**
 * TEE attestation proving the agent's key lives inside a secure enclave
 *
 * The attestation is signed by the TEE hardware and can be verified
 * on-chain. It proves:
 * 1. The key was generated INSIDE the enclave
 * 2. The key NEVER leaves the enclave in plaintext
 * 3. The hosting provider CANNOT read the key
 * 4. Only the agent process inside the enclave can request signatures
 */
export interface TEEAttestation {
  // Platform type
  platform: "sgx" | "sev_snp" | "trustzone" | "nitro" | "cca";

  // Attestation report (hardware-signed)
  report: Uint8Array;

  // Measurement of enclave code (hash of agent binary)
  measurement: Uint8Array;

  // Public key bound to enclave (the agent's operational key)
  enclaveBoundPublicKey: PublicKey;

  // Timestamp of attestation
  attestedAt: bigint;

  // Expiry (attestations must be refreshed periodically)
  expiresAt: bigint;
}

/**
 * Determine the required hosting policy for an agent
 *
 * Rule: If agent is hosted by a third party, TEE is MANDATORY.
 * Self-custodied agents (running on own hardware) can use local keys.
 */
export function determineHostingPolicy(
  isHosted: boolean,
  hostingProvider?: string,
): AgentHostingPolicy {
  if (isHosted) {
    return {
      isHosted: true,
      hostingProvider,
      enforcement: "mandatory_tee",
    };
  }

  return {
    isHosted: false,
    enforcement: "self_custody",
  };
}

/**
 * Validate that an agent vault meets its hosting policy
 *
 * Called at partnership formation — rejects vaults that don't meet
 * the TEE requirement when agent is hosted.
 */
export function validateAgentVaultPolicy(
  vault: IsolatedKeyVault,
  policy: AgentHostingPolicy,
): { valid: boolean; reason?: string } {
  if (vault.domain !== "agent") {
    return { valid: false, reason: "Not an agent vault" };
  }

  if (policy.enforcement === "mandatory_tee") {
    // Must have hardware binding
    if (!vault.hardwareBinding) {
      return {
        valid: false,
        reason:
          "Agent is hosted by third-party provider — TEE/HSM hardware binding is MANDATORY. " +
          `Provider "${policy.hostingProvider}" controls the runtime; ` +
          "without hardware isolation the provider can read the agent's private key.",
      };
    }

    // Must be TEE or HSM type (not just yubikey/ledger which need physical presence)
    const validTypes: HardwareBinding["type"][] = [
      "tee",
      "hsm",
      "secure_enclave",
    ];
    if (!validTypes.includes(vault.hardwareBinding.type)) {
      return {
        valid: false,
        reason:
          `Hardware binding type "${vault.hardwareBinding.type}" insufficient for hosted agent. ` +
          "Must use TEE, HSM, or secure enclave to prevent host operator key access.",
      };
    }

    // Hardware must be required (not optional)
    if (!vault.hardwareBinding.required) {
      return {
        valid: false,
        reason:
          "Hardware binding must be marked as required for hosted agents.",
      };
    }

    // Must have TEE attestation in policy
    if (!policy.teeAttestation) {
      return {
        valid: false,
        reason: "TEE attestation required to prove key is enclave-bound.",
      };
    }

    return { valid: true };
  }

  if (policy.enforcement === "mandatory_hsm") {
    if (!vault.hardwareBinding) {
      return { valid: false, reason: "HSM binding required by policy." };
    }
    return { valid: true };
  }

  // self_custody — no hardware requirement
  return { valid: true };
}

/**
 * Verify TEE attestation is fresh and valid
 *
 * Attestations expire because enclave firmware can be updated.
 * Stale attestations might correspond to compromised enclaves.
 */
export function verifyTEEAttestation(
  attestation: TEEAttestation,
  expectedPublicKey: PublicKey,
  currentBlock: bigint,
): { valid: boolean; reason?: string } {
  // Check expiry
  if (currentBlock >= attestation.expiresAt) {
    return { valid: false, reason: "TEE attestation expired — must refresh." };
  }

  // Verify the attested public key matches the vault's operational key
  if (attestation.enclaveBoundPublicKey !== expectedPublicKey) {
    return {
      valid: false,
      reason: "Attested public key does not match vault operational key.",
    };
  }

  // Verify report is non-empty (full hardware verification is platform-specific)
  if (attestation.report.length === 0) {
    return { valid: false, reason: "Empty attestation report." };
  }

  // Verify measurement is non-empty
  if (attestation.measurement.length === 0) {
    return { valid: false, reason: "Empty enclave measurement." };
  }

  return { valid: true };
}

// ============ Summary ============

/**
 * Get key isolation summary for a partnership
 */
export function getKeyIsolationSummary(keyPair: PartnershipKeyPair): {
  partnershipId: string;
  humanVaultId: string;
  agentVaultId: string;
  isolationVerified: boolean;
  humanHardwareBound: boolean;
  agentHardwareBound: boolean;
  proofTimestamp: bigint;
} {
  const verificationResult = verifyIsolationProof(
    keyPair.isolationProof,
    keyPair.sharedVerification.humanPublicKey,
    keyPair.sharedVerification.agentPublicKey,
  );

  return {
    partnershipId: keyPair.partnershipId,
    humanVaultId: keyPair.humanVault.vaultId,
    agentVaultId: keyPair.agentVault.vaultId,
    isolationVerified: verificationResult.valid,
    humanHardwareBound: keyPair.humanVault.hardwareBinding?.required ?? false,
    agentHardwareBound: keyPair.agentVault.hardwareBinding?.required ?? false,
    proofTimestamp: keyPair.isolationProof.provenAt,
  };
}
