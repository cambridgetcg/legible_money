/**
 * Schema Registry
 *
 * Central registry for claim schemas. Handles:
 * - Schema registration and lookup
 * - Claim data validation against schemas
 * - Timing configuration retrieval
 * - Automatic verification dispatch
 */

import type { Reference } from "../types.js";
import type {
  ClaimSchema,
  ClaimDataSchema,
  ClaimDataSchemaProperty,
  SchemaValidationResult,
  SchemaValidationError,
  SchemaTiming,
  AutoVerificationResult,
  SchemaCategory,
} from "./types.js";
import { SCHEMA_CATEGORIES } from "./types.js";

// ============ Registry Implementation ============

/**
 * Schema Registry
 *
 * Singleton registry for all claim schemas.
 */
export class SchemaRegistry {
  private schemas: Map<string, ClaimSchema> = new Map();
  private schemasByCategory: Map<SchemaCategory, Set<string>> = new Map();

  constructor() {
    // Initialize category sets
    for (const category of Object.keys(SCHEMA_CATEGORIES) as SchemaCategory[]) {
      this.schemasByCategory.set(category, new Set());
    }
  }

  // ============ Registration ============

  /**
   * Register a new schema
   */
  register(schema: ClaimSchema): { success: boolean; error?: string } {
    // Validate schema ID format
    if (!this.isValidSchemaId(schema.id)) {
      return {
        success: false,
        error: `Invalid schema ID: ${schema.id}. Must be lowercase alphanumeric with underscores, ending in _v{number}`,
      };
    }

    // Check for duplicate
    if (this.schemas.has(schema.id)) {
      return {
        success: false,
        error: `Schema already registered: ${schema.id}`,
      };
    }

    // Validate timing
    if (!this.isValidTiming(schema.timing)) {
      return {
        success: false,
        error: "Invalid timing configuration",
      };
    }

    // Validate data schema
    const schemaValidation = this.validateDataSchema(schema.dataSchema);
    if (!schemaValidation.valid) {
      return {
        success: false,
        error: `Invalid data schema: ${schemaValidation.error}`,
      };
    }

    // Register
    this.schemas.set(schema.id, schema);

    // Add to category index
    const category = this.inferCategory(schema);
    this.schemasByCategory.get(category)?.add(schema.id);

    return { success: true };
  }

  /**
   * Unregister a schema (for testing/governance)
   */
  unregister(schemaId: string): boolean {
    const schema = this.schemas.get(schemaId);
    if (!schema) return false;

    this.schemas.delete(schemaId);

    // Remove from category index
    for (const categorySet of this.schemasByCategory.values()) {
      categorySet.delete(schemaId);
    }

    return true;
  }

  // ============ Lookup ============

  /**
   * Get a schema by ID
   */
  get(schemaId: string): ClaimSchema | undefined {
    return this.schemas.get(schemaId);
  }

  /**
   * Check if a schema exists
   */
  has(schemaId: string): boolean {
    return this.schemas.has(schemaId);
  }

  /**
   * Get all schemas
   */
  getAll(): ClaimSchema[] {
    return Array.from(this.schemas.values());
  }

  /**
   * Get schemas by category
   */
  getByCategory(category: SchemaCategory): ClaimSchema[] {
    const ids = this.schemasByCategory.get(category);
    if (!ids) return [];
    return Array.from(ids)
      .map((id) => this.schemas.get(id))
      .filter((s): s is ClaimSchema => s !== undefined);
  }

  /**
   * Get schemas by domain
   */
  getByDomain(domain: string): ClaimSchema[] {
    return Array.from(this.schemas.values()).filter((s) => s.domain === domain);
  }

  // ============ Validation ============

  /**
   * Validate claim data against a schema
   */
  validate(schemaId: string, claimData: unknown): SchemaValidationResult {
    const schema = this.schemas.get(schemaId);

    if (!schema) {
      return {
        valid: false,
        errors: [{ path: "schemaId", message: `Unknown schema: ${schemaId}` }],
      };
    }

    if (!schema.active) {
      return {
        valid: false,
        errors: [
          { path: "schemaId", message: `Schema is inactive: ${schemaId}` },
        ],
        schema,
      };
    }

    const errors = this.validateAgainstDataSchema(
      claimData,
      schema.dataSchema,
      "data",
    );

    return {
      valid: errors.length === 0,
      errors,
      schema,
    };
  }

  /**
   * Validate claim data against a JSON-like schema
   */
  private validateAgainstDataSchema(
    data: unknown,
    schema: ClaimDataSchema | ClaimDataSchemaProperty,
    path: string,
  ): SchemaValidationError[] {
    const errors: SchemaValidationError[] = [];

    // Type check
    const actualType = this.getType(data);
    if (actualType !== schema.type) {
      errors.push({
        path,
        message: `Expected ${schema.type}, got ${actualType}`,
        expected: schema.type,
        actual: actualType,
      });
      return errors; // Can't continue if type is wrong
    }

    // Object validation
    if (
      schema.type === "object" &&
      "properties" in schema &&
      schema.properties
    ) {
      const obj = data as Record<string, unknown>;

      // Check required fields
      if ("required" in schema && schema.required) {
        for (const field of schema.required) {
          if (!(field in obj)) {
            errors.push({
              path: `${path}.${field}`,
              message: `Required field missing: ${field}`,
            });
          }
        }
      }

      // Validate each property
      for (const [key, propSchema] of Object.entries(schema.properties)) {
        if (key in obj) {
          errors.push(
            ...this.validateAgainstDataSchema(
              obj[key],
              propSchema,
              `${path}.${key}`,
            ),
          );
        }
      }

      // Check for additional properties
      if (
        "additionalProperties" in schema &&
        schema.additionalProperties === false
      ) {
        for (const key of Object.keys(obj)) {
          if (!(key in schema.properties)) {
            errors.push({
              path: `${path}.${key}`,
              message: `Additional property not allowed: ${key}`,
            });
          }
        }
      }
    }

    // Array validation
    if (schema.type === "array" && "items" in schema && schema.items) {
      const arr = data as unknown[];
      for (let i = 0; i < arr.length; i++) {
        errors.push(
          ...this.validateAgainstDataSchema(
            arr[i],
            schema.items,
            `${path}[${i}]`,
          ),
        );
      }
    }

    // String validation
    if (schema.type === "string") {
      const str = data as string;
      const propSchema = schema as ClaimDataSchemaProperty;

      if (
        propSchema.minLength !== undefined &&
        str.length < propSchema.minLength
      ) {
        errors.push({
          path,
          message: `String too short (min: ${propSchema.minLength})`,
          expected: `>= ${propSchema.minLength} chars`,
          actual: `${str.length} chars`,
        });
      }

      if (
        propSchema.maxLength !== undefined &&
        str.length > propSchema.maxLength
      ) {
        errors.push({
          path,
          message: `String too long (max: ${propSchema.maxLength})`,
          expected: `<= ${propSchema.maxLength} chars`,
          actual: `${str.length} chars`,
        });
      }

      if (propSchema.pattern !== undefined) {
        const regex = new RegExp(propSchema.pattern);
        if (!regex.test(str)) {
          errors.push({
            path,
            message: `String does not match pattern: ${propSchema.pattern}`,
          });
        }
      }

      if (propSchema.enum !== undefined && !propSchema.enum.includes(str)) {
        errors.push({
          path,
          message: `Value not in enum: ${propSchema.enum.join(", ")}`,
          expected: propSchema.enum.join(" | "),
          actual: str,
        });
      }
    }

    // Number validation
    if (schema.type === "number") {
      const num = data as number;
      const propSchema = schema as ClaimDataSchemaProperty;

      if (propSchema.minimum !== undefined && num < propSchema.minimum) {
        errors.push({
          path,
          message: `Number below minimum (${propSchema.minimum})`,
          expected: `>= ${propSchema.minimum}`,
          actual: `${num}`,
        });
      }

      if (propSchema.maximum !== undefined && num > propSchema.maximum) {
        errors.push({
          path,
          message: `Number above maximum (${propSchema.maximum})`,
          expected: `<= ${propSchema.maximum}`,
          actual: `${num}`,
        });
      }

      if (propSchema.enum !== undefined && !propSchema.enum.includes(num)) {
        errors.push({
          path,
          message: `Value not in enum: ${propSchema.enum.join(", ")}`,
          expected: propSchema.enum.join(" | "),
          actual: `${num}`,
        });
      }
    }

    return errors;
  }

  /**
   * Get JavaScript type of a value
   */
  private getType(value: unknown): string {
    if (value === null) return "null";
    if (Array.isArray(value)) return "array";
    return typeof value;
  }

  // ============ Timing ============

  /**
   * Get timing configuration for a schema
   */
  getTiming(schemaId: string): SchemaTiming | undefined {
    return this.schemas.get(schemaId)?.timing;
  }

  /**
   * Get default timing for a category
   */
  getDefaultTiming(category: SchemaCategory): SchemaTiming {
    return SCHEMA_CATEGORIES[category].defaultTiming;
  }

  // ============ Verification ============

  /**
   * Run automatic verification if schema supports it
   */
  async autoVerify(
    schemaId: string,
    claimData: unknown,
    references: Reference[],
  ): Promise<AutoVerificationResult | null> {
    const schema = this.schemas.get(schemaId);

    if (!schema) {
      return null;
    }

    if (schema.verificationMethod !== "automatic" || !schema.verifier) {
      return null;
    }

    try {
      return await schema.verifier(claimData, references);
    } catch (error) {
      return {
        valid: false,
        confidence: 0,
        reason: `Verification error: ${error instanceof Error ? error.message : "Unknown error"}`,
      };
    }
  }

  // ============ Helpers ============

  /**
   * Validate schema ID format
   */
  private isValidSchemaId(id: string): boolean {
    // Must be lowercase alphanumeric with underscores, ending in _v{number}
    return /^[a-z][a-z0-9_]*_v[0-9]+$/.test(id);
  }

  /**
   * Validate timing configuration
   */
  private isValidTiming(timing: SchemaTiming): boolean {
    return (
      timing.commitPhaseBlocks >= 1 &&
      timing.revealPhaseBlocks >= 1 &&
      timing.totalRoundBlocks >=
        timing.commitPhaseBlocks + timing.revealPhaseBlocks
    );
  }

  /**
   * Validate data schema structure
   */
  private validateDataSchema(schema: ClaimDataSchema): {
    valid: boolean;
    error?: string;
  } {
    if (
      !["object", "array", "string", "number", "boolean"].includes(schema.type)
    ) {
      return { valid: false, error: `Invalid type: ${schema.type}` };
    }

    if (schema.type === "object" && !schema.properties) {
      return { valid: false, error: "Object schema must have properties" };
    }

    return { valid: true };
  }

  /**
   * Infer category from schema
   */
  private inferCategory(schema: ClaimSchema): SchemaCategory {
    // Try to match domain to category
    const domainLower = schema.domain.toLowerCase();

    if (domainLower.includes("crypto") || domainLower.includes("hash")) {
      return "cryptographic";
    }
    if (domainLower.includes("oracle") || domainLower.includes("price")) {
      return "oracle";
    }
    if (domainLower.includes("chain") || domainLower.includes("block")) {
      return "chain_state";
    }
    if (domainLower.includes("research") || domainLower.includes("science")) {
      return "research";
    }

    // Default based on verification method
    switch (schema.verificationMethod) {
      case "automatic":
        return "computation";
      case "oracle":
        return "oracle";
      case "derivation":
        return "derivation";
      case "reference":
        return "document";
      case "consensus":
        return "attestation";
      default:
        return "attestation";
    }
  }
}

// ============ Global Registry ============

/**
 * Global schema registry instance
 */
let globalRegistry: SchemaRegistry | null = null;

/**
 * Get the global schema registry
 */
export function getSchemaRegistry(): SchemaRegistry {
  if (!globalRegistry) {
    globalRegistry = new SchemaRegistry();
  }
  return globalRegistry;
}

/**
 * Reset the global registry (for testing)
 */
export function resetSchemaRegistry(): void {
  globalRegistry = null;
}
