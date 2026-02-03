/**
 * Sparse Merkle Tree
 *
 * Authenticated data structure for state storage.
 * Supports efficient inclusion AND exclusion proofs.
 *
 * Properties:
 * - O(log n) updates
 * - O(log n) proofs
 * - Constant-size root regardless of data size
 * - Efficient for sparse data (most leaves empty)
 */

import { blake3Hash, constantTimeEqual } from './hashing.js';
import { bytesToHex, hexToBytes } from './keys.js';

// ============ Types ============

export type MerkleRoot = string;    // Hex-encoded root (32 bytes)
export type MerkleProof = string[]; // Path of sibling hashes

export interface SparseMerkleTree {
  root: MerkleRoot;
  depth: number;  // Tree depth (256 for SHA-256 sized keys)
}

export interface MerkleProofData {
  key: string;
  value: Uint8Array | null;  // null for non-membership proof
  proof: MerkleProof;
  root: MerkleRoot;
}

export interface SMTNode {
  hash: Uint8Array;
  left?: SMTNode;
  right?: SMTNode;
  key?: Uint8Array;
  value?: Uint8Array;
}

// ============ Constants ============

const EMPTY_VALUE = new Uint8Array(32);  // All zeros
const TREE_DEPTH = 256;  // For 256-bit keys

// Precomputed empty subtree hashes
const EMPTY_HASHES: Uint8Array[] = new Array(TREE_DEPTH + 1);

// Initialize empty hashes
(function initEmptyHashes() {
  EMPTY_HASHES[0] = EMPTY_VALUE;
  for (let i = 1; i <= TREE_DEPTH; i++) {
    EMPTY_HASHES[i] = hashNode(EMPTY_HASHES[i - 1], EMPTY_HASHES[i - 1]);
  }
})();

// ============ Sparse Merkle Tree Implementation ============

/**
 * In-memory Sparse Merkle Tree
 */
export class InMemorySMT {
  private nodes: Map<string, SMTNode> = new Map();
  private root: Uint8Array;
  private readonly depth: number;

  constructor(depth: number = TREE_DEPTH) {
    this.depth = depth;
    this.root = EMPTY_HASHES[depth];
  }

  /**
   * Get the current root hash
   */
  getRoot(): MerkleRoot {
    return bytesToHex(this.root);
  }

  /**
   * Get a value by key
   */
  get(key: Uint8Array): Uint8Array | null {
    const keyBits = keyToBits(key, this.depth);
    let node = this.nodes.get(bytesToHex(this.root));

    for (let i = 0; i < this.depth && node; i++) {
      const bit = keyBits[i];
      node = bit ? node.right : node.left;
    }

    if (node && node.value && node.key && constantTimeEqual(node.key, key)) {
      return node.value;
    }
    return null;
  }

  /**
   * Set a key-value pair
   */
  set(key: Uint8Array, value: Uint8Array): MerkleRoot {
    const keyBits = keyToBits(key, this.depth);
    this.root = this.setRecursive(this.root, key, value, keyBits, 0);
    return this.getRoot();
  }

  /**
   * Delete a key
   */
  delete(key: Uint8Array): MerkleRoot {
    return this.set(key, EMPTY_VALUE);
  }

  /**
   * Generate a Merkle proof for a key
   */
  prove(key: Uint8Array): MerkleProofData {
    const keyBits = keyToBits(key, this.depth);
    const proof: Uint8Array[] = [];
    let currentHash = this.root;

    for (let i = 0; i < this.depth; i++) {
      const node = this.nodes.get(bytesToHex(currentHash));

      if (!node) {
        // Empty subtree
        proof.push(EMPTY_HASHES[this.depth - i - 1]);
        currentHash = EMPTY_HASHES[this.depth - i - 1];
      } else {
        const bit = keyBits[i];
        if (bit) {
          proof.push(node.left?.hash || EMPTY_HASHES[this.depth - i - 1]);
          currentHash = node.right?.hash || EMPTY_HASHES[this.depth - i - 1];
        } else {
          proof.push(node.right?.hash || EMPTY_HASHES[this.depth - i - 1]);
          currentHash = node.left?.hash || EMPTY_HASHES[this.depth - i - 1];
        }
      }
    }

    const value = this.get(key);

    return {
      key: bytesToHex(key),
      value,
      proof: proof.map(bytesToHex),
      root: this.getRoot(),
    };
  }

  /**
   * Verify a Merkle proof
   */
  static verify(proofData: MerkleProofData): boolean {
    const key = hexToBytes(proofData.key);
    const keyBits = keyToBits(key, proofData.proof.length);
    const proof = proofData.proof.map(hexToBytes);

    // Start with leaf hash
    let currentHash: Uint8Array;
    if (proofData.value) {
      currentHash = hashLeaf(key, proofData.value);
    } else {
      currentHash = EMPTY_VALUE;
    }

    // Traverse up the tree
    for (let i = proof.length - 1; i >= 0; i--) {
      const sibling = proof[i];
      const bit = keyBits[i];

      if (bit) {
        currentHash = hashNode(sibling, currentHash);
      } else {
        currentHash = hashNode(currentHash, sibling);
      }
    }

    return bytesToHex(currentHash) === proofData.root;
  }

  /**
   * Batch update multiple key-value pairs
   */
  batchSet(entries: Array<{ key: Uint8Array; value: Uint8Array }>): MerkleRoot {
    for (const { key, value } of entries) {
      this.set(key, value);
    }
    return this.getRoot();
  }

  // ============ Private Methods ============

  private setRecursive(
    nodeHash: Uint8Array,
    key: Uint8Array,
    value: Uint8Array,
    keyBits: boolean[],
    depth: number
  ): Uint8Array {
    if (depth === this.depth) {
      // Leaf node
      if (constantTimeEqual(value, EMPTY_VALUE)) {
        return EMPTY_HASHES[0];
      }
      const leafHash = hashLeaf(key, value);
      this.nodes.set(bytesToHex(leafHash), {
        hash: leafHash,
        key,
        value,
      });
      return leafHash;
    }

    const node = this.nodes.get(bytesToHex(nodeHash));
    const bit = keyBits[depth];

    let leftHash: Uint8Array;
    let rightHash: Uint8Array;

    if (!node) {
      leftHash = EMPTY_HASHES[this.depth - depth - 1];
      rightHash = EMPTY_HASHES[this.depth - depth - 1];
    } else {
      leftHash = node.left?.hash || EMPTY_HASHES[this.depth - depth - 1];
      rightHash = node.right?.hash || EMPTY_HASHES[this.depth - depth - 1];
    }

    if (bit) {
      rightHash = this.setRecursive(rightHash, key, value, keyBits, depth + 1);
    } else {
      leftHash = this.setRecursive(leftHash, key, value, keyBits, depth + 1);
    }

    // Check for empty subtree
    if (
      constantTimeEqual(leftHash, EMPTY_HASHES[this.depth - depth - 1]) &&
      constantTimeEqual(rightHash, EMPTY_HASHES[this.depth - depth - 1])
    ) {
      return EMPTY_HASHES[this.depth - depth];
    }

    const newHash = hashNode(leftHash, rightHash);

    this.nodes.set(bytesToHex(newHash), {
      hash: newHash,
      left: { hash: leftHash },
      right: { hash: rightHash },
    });

    return newHash;
  }
}

// ============ Helper Functions ============

/**
 * Hash a leaf node (key-value pair)
 */
function hashLeaf(key: Uint8Array, value: Uint8Array): Uint8Array {
  // Prefix with 0x00 for leaf
  const data = new Uint8Array(1 + key.length + value.length);
  data[0] = 0x00;
  data.set(key, 1);
  data.set(value, 1 + key.length);
  return blake3Hash(data);
}

/**
 * Hash two child nodes
 */
function hashNode(left: Uint8Array, right: Uint8Array): Uint8Array {
  // Prefix with 0x01 for internal node
  const data = new Uint8Array(1 + left.length + right.length);
  data[0] = 0x01;
  data.set(left, 1);
  data.set(right, 1 + left.length);
  return blake3Hash(data);
}

/**
 * Convert key to array of bits (path through tree)
 */
function keyToBits(key: Uint8Array, depth: number): boolean[] {
  const bits: boolean[] = [];
  for (let i = 0; i < depth; i++) {
    const byteIndex = Math.floor(i / 8);
    const bitIndex = 7 - (i % 8);
    bits.push(((key[byteIndex] || 0) >> bitIndex & 1) === 1);
  }
  return bits;
}

// ============ Compact Merkle Tree (for sequences) ============

/**
 * Compact Merkle Tree for sequential data (blocks, transactions)
 */
export class CompactMerkleTree {
  private leaves: Uint8Array[] = [];
  private layers: Uint8Array[][] = [];

  /**
   * Add a leaf to the tree
   */
  append(data: Uint8Array): number {
    const leaf = hashLeaf(new Uint8Array([]), data);
    this.leaves.push(leaf);
    this.rebuildTree();
    return this.leaves.length - 1;
  }

  /**
   * Get the root hash
   */
  getRoot(): MerkleRoot {
    if (this.layers.length === 0) {
      return bytesToHex(EMPTY_VALUE);
    }
    return bytesToHex(this.layers[this.layers.length - 1][0]);
  }

  /**
   * Get proof for leaf at index
   */
  prove(index: number): MerkleProof {
    if (index >= this.leaves.length) {
      throw new Error('Index out of bounds');
    }

    const proof: Uint8Array[] = [];
    let idx = index;

    for (let layer = 0; layer < this.layers.length - 1; layer++) {
      const siblingIdx = idx % 2 === 0 ? idx + 1 : idx - 1;
      if (siblingIdx < this.layers[layer].length) {
        proof.push(this.layers[layer][siblingIdx]);
      } else {
        proof.push(EMPTY_VALUE);
      }
      idx = Math.floor(idx / 2);
    }

    return proof.map(bytesToHex);
  }

  /**
   * Verify a proof
   */
  static verify(leaf: Uint8Array, index: number, proof: MerkleProof, root: MerkleRoot): boolean {
    let hash = hashLeaf(new Uint8Array([]), leaf);
    let idx = index;

    for (const siblingHex of proof) {
      const sibling = hexToBytes(siblingHex);
      if (idx % 2 === 0) {
        hash = hashNode(hash, sibling);
      } else {
        hash = hashNode(sibling, hash);
      }
      idx = Math.floor(idx / 2);
    }

    return bytesToHex(hash) === root;
  }

  private rebuildTree(): void {
    this.layers = [this.leaves.slice()];

    while (this.layers[this.layers.length - 1].length > 1) {
      const currentLayer = this.layers[this.layers.length - 1];
      const nextLayer: Uint8Array[] = [];

      for (let i = 0; i < currentLayer.length; i += 2) {
        const left = currentLayer[i];
        const right = currentLayer[i + 1] || EMPTY_VALUE;
        nextLayer.push(hashNode(left, right));
      }

      this.layers.push(nextLayer);
    }
  }
}

// ============ Exports ============

export {
  EMPTY_VALUE,
  TREE_DEPTH,
  EMPTY_HASHES,
  hashLeaf,
  hashNode,
  keyToBits,
};
