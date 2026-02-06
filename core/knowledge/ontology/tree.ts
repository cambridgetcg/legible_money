/**
 * Domain Tree - Tree Structure and Operations
 *
 * Manages the hierarchical structure of ontological domains with efficient
 * lookup and traversal operations.
 */

import {
  type OntologicalStratum,
  type OntologicalDomain,
  type DomainTree,
  type DomainNode,
  type DomainRegistrationInput,
  type DomainRegistrationResult,
  STRATUM_HIERARCHY,
  STRATUM_LOGIC_ZONE,
  STRATUM_EPISTEMIC_CATEGORY,
} from "./types.js";
import { isValidStratumDependency, getStratumLevel } from "./strata.js";

// ============================================================================
// Root Domain
// ============================================================================

/**
 * The root domain of the knowledge tree.
 */
export const ROOT_DOMAIN: OntologicalDomain = {
  id: "root",
  name: "Knowledge",
  stratum: "fundamental",
  parentDomain: null,
  childDomains: [],
  dependsOnStrata: [],
  logicZoneDefault: "propositional",
  epistemicCategoryDefault: "formal",
  description: "Root of the ontological knowledge tree",
  examples: [],
  factCount: 0,
  axiomCount: 0,
  createdAt: 0,
};

// ============================================================================
// Tree Creation
// ============================================================================

/**
 * Create an empty domain tree with just the root.
 */
export function createEmptyDomainTree(): DomainTree {
  const rootNode: DomainNode = {
    domain: ROOT_DOMAIN,
    children: [],
    depth: 0,
    path: [],
  };

  const tree: DomainTree = {
    root: rootNode,
    strataIndex: new Map(),
    domainIndex: new Map(),
    domainCount: 1,
    maxDepth: 0,
  };

  // Initialize strata index
  for (const stratum of Object.keys(STRATUM_HIERARCHY)) {
    tree.strataIndex.set(stratum as OntologicalStratum, []);
  }

  // Add root to indexes
  tree.domainIndex.set("root", rootNode);
  tree.strataIndex.get("fundamental")!.push("root");

  return tree;
}

/**
 * Create a domain tree from an array of domains.
 */
export function createDomainTree(domains: OntologicalDomain[]): DomainTree {
  const tree = createEmptyDomainTree();

  // Sort domains by depth (parents before children), with ID tiebreaker for determinism
  const sorted = [...domains].sort((a, b) => {
    const depthA = a.id.split(".").length;
    const depthB = b.id.split(".").length;
    return depthA - depthB || a.id.localeCompare(b.id);
  });

  for (const domain of sorted) {
    if (domain.id !== "root") {
      addDomainToTree(tree, domain);
    }
  }

  return tree;
}

// ============================================================================
// Domain Operations
// ============================================================================

/**
 * Add a domain to the tree.
 *
 * @param tree - The domain tree to modify
 * @param domain - The domain to add
 * @returns The created node, or null if parent not found
 */
export function addDomainToTree(
  tree: DomainTree,
  domain: OntologicalDomain,
): DomainNode | null {
  // Find parent node
  const parentId = domain.parentDomain ?? "root";
  const parentNode = tree.domainIndex.get(parentId);

  if (!parentNode) {
    return null;
  }

  // Create node
  const path = [...parentNode.path, domain.id.split(".").pop()!];
  const node: DomainNode = {
    domain,
    children: [],
    depth: parentNode.depth + 1,
    path,
  };

  // Add to parent's children
  parentNode.children.push(node);

  // Update parent's childDomains
  if (!parentNode.domain.childDomains.includes(domain.id)) {
    parentNode.domain.childDomains.push(domain.id);
  }

  // Add to indexes
  tree.domainIndex.set(domain.id, node);

  const strataList = tree.strataIndex.get(domain.stratum);
  if (strataList && !strataList.includes(domain.id)) {
    strataList.push(domain.id);
  }

  // Update tree metadata
  tree.domainCount++;
  tree.maxDepth = Math.max(tree.maxDepth, node.depth);

  return node;
}

/**
 * Get a domain by ID.
 */
export function getDomain(
  tree: DomainTree,
  domainId: string,
): OntologicalDomain | null {
  const node = tree.domainIndex.get(domainId);
  return node?.domain ?? null;
}

/**
 * Get a domain node by ID.
 */
export function getDomainNode(
  tree: DomainTree,
  domainId: string,
): DomainNode | null {
  return tree.domainIndex.get(domainId) ?? null;
}

/**
 * Check if a domain exists.
 */
export function hasDomain(tree: DomainTree, domainId: string): boolean {
  return tree.domainIndex.has(domainId);
}

/**
 * Get all domains in a stratum.
 */
export function getDomainsByStratum(
  tree: DomainTree,
  stratum: OntologicalStratum,
): OntologicalDomain[] {
  const ids = tree.strataIndex.get(stratum) ?? [];
  return ids.map((id) => tree.domainIndex.get(id)!.domain);
}

// ============================================================================
// Tree Traversal
// ============================================================================

/**
 * Get the path from root to a domain.
 */
export function getDomainPath(
  tree: DomainTree,
  domainId: string,
): OntologicalDomain[] {
  const path: OntologicalDomain[] = [];
  let current = tree.domainIndex.get(domainId);

  while (current) {
    path.unshift(current.domain);
    if (current.domain.parentDomain) {
      current = tree.domainIndex.get(current.domain.parentDomain);
    } else {
      break;
    }
  }

  return path;
}

/**
 * Get all ancestors of a domain.
 */
export function getAncestors(
  tree: DomainTree,
  domainId: string,
): OntologicalDomain[] {
  const ancestors: OntologicalDomain[] = [];
  const node = tree.domainIndex.get(domainId);

  if (!node?.domain.parentDomain) {
    return ancestors;
  }

  let current = tree.domainIndex.get(node.domain.parentDomain);
  while (current) {
    ancestors.push(current.domain);
    if (current.domain.parentDomain) {
      current = tree.domainIndex.get(current.domain.parentDomain);
    } else {
      break;
    }
  }

  return ancestors;
}

/**
 * Get all descendants of a domain.
 */
export function getDescendants(
  tree: DomainTree,
  domainId: string,
): OntologicalDomain[] {
  const descendants: OntologicalDomain[] = [];
  const node = tree.domainIndex.get(domainId);

  if (!node) {
    return descendants;
  }

  function traverse(n: DomainNode): void {
    for (const child of n.children) {
      descendants.push(child.domain);
      traverse(child);
    }
  }

  traverse(node);
  return descendants;
}

/**
 * Get direct children of a domain.
 */
export function getChildren(
  tree: DomainTree,
  domainId: string,
): OntologicalDomain[] {
  const node = tree.domainIndex.get(domainId);
  return node?.children.map((c) => c.domain) ?? [];
}

/**
 * Get sibling domains (same parent).
 */
export function getSiblings(
  tree: DomainTree,
  domainId: string,
): OntologicalDomain[] {
  const node = tree.domainIndex.get(domainId);
  if (!node?.domain.parentDomain) {
    return [];
  }

  const parentNode = tree.domainIndex.get(node.domain.parentDomain);
  return (
    parentNode?.children
      .filter((c) => c.domain.id !== domainId)
      .map((c) => c.domain) ?? []
  );
}

/**
 * Find the lowest common ancestor of two domains.
 */
export function getLowestCommonAncestor(
  tree: DomainTree,
  domainIdA: string,
  domainIdB: string,
): OntologicalDomain | null {
  const pathA = getDomainPath(tree, domainIdA);
  const pathB = getDomainPath(tree, domainIdB);

  let lca: OntologicalDomain | null = null;

  for (let i = 0; i < Math.min(pathA.length, pathB.length); i++) {
    if (pathA[i].id === pathB[i].id) {
      lca = pathA[i];
    } else {
      break;
    }
  }

  return lca;
}

// ============================================================================
// Domain Registration
// ============================================================================

/**
 * Validate and register a new domain.
 */
export function registerDomain(
  tree: DomainTree,
  input: DomainRegistrationInput,
  currentBlock: number,
): DomainRegistrationResult {
  const errors: string[] = [];

  // Validate ID format
  if (!isValidDomainId(input.id)) {
    errors.push(`Invalid domain ID format: ${input.id}`);
  }

  // Check if domain already exists
  if (tree.domainIndex.has(input.id)) {
    errors.push(`Domain already exists: ${input.id}`);
  }

  // Validate parent exists (or is root)
  const parentId = input.parentDomain ?? "root";
  const parentNode = tree.domainIndex.get(parentId);
  if (!parentNode) {
    errors.push(`Parent domain not found: ${parentId}`);
  }

  // Validate ID follows parent path
  if (input.parentDomain && parentNode) {
    const expectedPrefix = input.parentDomain + ".";
    if (!input.id.startsWith(expectedPrefix)) {
      errors.push(`Domain ID must start with parent path: ${expectedPrefix}`);
    }
  }

  // Validate stratum is appropriate
  if (parentNode) {
    const parentLevel = getStratumLevel(parentNode.domain.stratum);
    const newLevel = getStratumLevel(input.stratum);
    if (newLevel < parentLevel) {
      errors.push(
        `Child domain cannot be in lower stratum than parent (${input.stratum} < ${parentNode.domain.stratum})`,
      );
    }
  }

  // Validate dependencies
  for (const dep of input.dependsOnStrata) {
    if (!isValidStratumDependency(input.stratum, dep)) {
      errors.push(
        `Invalid stratum dependency: ${input.stratum} cannot depend on ${dep}`,
      );
    }
  }

  // Determine if governance is required
  const isLeaf = true; // New domains start as leaves
  const requiresGovernance =
    !isLeaf ||
    getStratumLevel(input.stratum) <= 2 ||
    input.parentDomain === "root";

  // Calculate minimum stake
  const baseStake = 100n * 10n ** 18n; // 100 LGM
  const governanceMultiplier = requiresGovernance ? 10n : 1n;
  const minimumStake = baseStake * governanceMultiplier;

  if (errors.length > 0) {
    return {
      domain: null as unknown as OntologicalDomain,
      requiresGovernance,
      minimumStake,
      errors,
    };
  }

  // Create the domain
  const domain: OntologicalDomain = {
    id: input.id,
    name: input.name,
    stratum: input.stratum,
    parentDomain: input.parentDomain,
    childDomains: [],
    dependsOnStrata: input.dependsOnStrata,
    logicZoneDefault:
      input.logicZoneDefault ?? STRATUM_LOGIC_ZONE[input.stratum],
    epistemicCategoryDefault:
      input.epistemicCategoryDefault ??
      STRATUM_EPISTEMIC_CATEGORY[input.stratum],
    description: input.description,
    examples: input.examples,
    factCount: 0,
    axiomCount: 0,
    createdAt: currentBlock,
  };

  return {
    domain,
    requiresGovernance,
    minimumStake,
    errors: [],
  };
}

/**
 * Validate a domain ID format.
 *
 * Valid: 'physics', 'physics.mechanics', 'physics.mechanics.quantum'
 * Invalid: '.physics', 'physics.', 'physics..mechanics', 'PHYSICS'
 */
export function isValidDomainId(id: string): boolean {
  if (!id || id === "root") return id === "root";

  // Must be lowercase letters, numbers, and underscores, separated by dots
  const pattern = /^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$/;
  return pattern.test(id);
}

/**
 * Get the parent ID from a domain ID.
 */
export function getParentIdFromDomainId(domainId: string): string | null {
  if (domainId === "root") return null;

  const parts = domainId.split(".");
  if (parts.length === 1) return "root";

  return parts.slice(0, -1).join(".");
}

/**
 * Get the depth of a domain from its ID.
 */
export function getDomainDepth(domainId: string): number {
  if (domainId === "root") return 0;
  return domainId.split(".").length;
}

// ============================================================================
// Tree Statistics
// ============================================================================

/**
 * Get statistics about the domain tree.
 */
export function getTreeStatistics(tree: DomainTree): TreeStatistics {
  const strataStats: Record<OntologicalStratum, number> = {
    fundamental: 0,
    physical: 0,
    chemical: 0,
    biological: 0,
    cognitive: 0,
    social: 0,
    technological: 0,
  };

  for (const [stratum, domains] of tree.strataIndex) {
    strataStats[stratum] = domains.length;
  }

  return {
    totalDomains: tree.domainCount,
    maxDepth: tree.maxDepth,
    domainsByStratum: strataStats,
    averageChildCount: calculateAverageChildCount(tree),
    leafCount: countLeaves(tree),
  };
}

interface TreeStatistics {
  totalDomains: number;
  maxDepth: number;
  domainsByStratum: Record<OntologicalStratum, number>;
  averageChildCount: number;
  leafCount: number;
}

function calculateAverageChildCount(tree: DomainTree): number {
  let totalChildren = 0;
  let nonLeafCount = 0;

  for (const node of tree.domainIndex.values()) {
    if (node.children.length > 0) {
      totalChildren += node.children.length;
      nonLeafCount++;
    }
  }

  return nonLeafCount > 0 ? totalChildren / nonLeafCount : 0;
}

function countLeaves(tree: DomainTree): number {
  let count = 0;
  for (const node of tree.domainIndex.values()) {
    if (node.children.length === 0) {
      count++;
    }
  }
  return count;
}

// ============================================================================
// Tree Serialization
// ============================================================================

/**
 * Serialize a domain tree to a plain object for storage.
 */
export function serializeDomainTree(tree: DomainTree): SerializedDomainTree {
  const domains: OntologicalDomain[] = [];

  for (const node of tree.domainIndex.values()) {
    domains.push(node.domain);
  }

  // Sort by domain ID for deterministic serialization (Map iteration is insertion-order)
  domains.sort((a, b) => a.id.localeCompare(b.id));

  return {
    domains,
    version: 1,
  };
}

/**
 * Deserialize a domain tree from storage.
 */
export function deserializeDomainTree(data: SerializedDomainTree): DomainTree {
  return createDomainTree(data.domains);
}

interface SerializedDomainTree {
  domains: OntologicalDomain[];
  version: number;
}
