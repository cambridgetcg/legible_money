/**
 * CLI Output Formatting
 *
 * Agent-first output: JSON by default, human-readable optional.
 * No colors in JSON mode for clean parsing.
 */

import type {
  CLIResponse,
  CLISuccess,
  CLIError,
  CLIMeta,
  OutputMode,
} from "../types.js";

// ============ Output Functions ============

/**
 * Output a successful response
 */
export function outputSuccess<T>(
  command: string,
  data: T,
  meta: Partial<CLIMeta>,
  mode: OutputMode,
): void {
  const response: CLISuccess<T> = {
    success: true,
    command,
    data,
    meta: {
      timestamp: new Date().toISOString(),
      latencyMs: meta.latencyMs ?? 0,
      blockNumber: meta.blockNumber,
      endpoint: meta.endpoint,
    },
  };

  output(response, mode);
}

/**
 * Output an error response
 */
export function outputError(error: CLIError, mode: OutputMode): void {
  output(error, mode);
}

/**
 * Output any response based on mode
 */
export function output(response: CLIResponse, mode: OutputMode): void {
  switch (mode) {
    case "json":
      outputJSON(response);
      break;
    case "human":
      outputHuman(response);
      break;
    case "quiet":
      outputQuiet(response);
      break;
  }
}

// ============ JSON Output ============

/**
 * Output as JSON (default, agent-optimized)
 */
function outputJSON(response: CLIResponse): void {
  // Pretty-print for readability but still valid JSON
  console.log(JSON.stringify(response, null, 2));
}

// ============ Human Output ============

/**
 * Output in human-readable format
 */
function outputHuman(response: CLIResponse): void {
  if (response.success) {
    outputHumanSuccess(response as CLISuccess);
  } else {
    outputHumanError(response as CLIError);
  }
}

function outputHumanSuccess(response: CLISuccess): void {
  const { command, data, meta } = response;

  // Header
  console.log(`\n${command}`);
  console.log("─".repeat(40));

  // Data (format based on type)
  if (typeof data === "object" && data !== null) {
    formatObject(data as Record<string, unknown>);
  } else {
    console.log(String(data));
  }

  // Footer with metadata
  console.log("─".repeat(40));
  console.log(
    `Block: ${meta.blockNumber ?? "N/A"} | Latency: ${meta.latencyMs}ms`,
  );
  console.log();
}

function outputHumanError(response: CLIError): void {
  const { command, error } = response;

  console.error(`\n[ERROR] ${command}`);
  console.error("─".repeat(40));
  console.error(`Code:    ${error.code} (${error.name})`);
  console.error(`Message: ${error.message}`);
  if (error.suggestion) {
    console.error(`Hint:    ${error.suggestion}`);
  }
  if (error.retryable) {
    console.error(`Status:  Retryable`);
  }
  console.error();
}

/**
 * Format object for human output
 */
function formatObject(obj: Record<string, unknown>, indent = 0): void {
  const prefix = "  ".repeat(indent);

  for (const [key, value] of Object.entries(obj)) {
    if (value === null || value === undefined) {
      continue;
    }

    const label = formatLabel(key);

    if (typeof value === "object" && !Array.isArray(value)) {
      console.log(`${prefix}${label}:`);
      formatObject(value as Record<string, unknown>, indent + 1);
    } else if (Array.isArray(value)) {
      console.log(`${prefix}${label}: [${value.length} items]`);
      if (value.length <= 5) {
        value.forEach((item, i) => {
          if (typeof item === "object") {
            console.log(`${prefix}  [${i}]:`);
            formatObject(item as Record<string, unknown>, indent + 2);
          } else {
            console.log(`${prefix}  - ${item}`);
          }
        });
      }
    } else if (typeof value === "string" && value.length > 50) {
      // Truncate long strings
      console.log(`${prefix}${label}: ${value.slice(0, 47)}...`);
    } else {
      console.log(`${prefix}${label}: ${formatValue(value)}`);
    }
  }
}

/**
 * Format camelCase to Title Case
 */
function formatLabel(key: string): string {
  return key
    .replace(/([A-Z])/g, " $1")
    .replace(/^./, (s) => s.toUpperCase())
    .trim();
}

/**
 * Format values for display
 */
function formatValue(value: unknown): string {
  if (typeof value === "bigint") {
    return value.toString();
  }
  if (typeof value === "number") {
    // Format large numbers with commas
    if (value > 1000000) {
      return value.toLocaleString();
    }
    // Format decimals
    if (!Number.isInteger(value)) {
      return value.toFixed(4);
    }
  }
  return String(value);
}

// ============ Quiet Output ============

/**
 * Minimal output for scripting
 */
function outputQuiet(response: CLIResponse): void {
  if (response.success) {
    const data = response.data;

    // For simple values, just output the value
    if (
      typeof data === "string" ||
      typeof data === "number" ||
      typeof data === "boolean"
    ) {
      console.log(data);
      return;
    }

    // For objects, output just the primary field
    if (typeof data === "object" && data !== null) {
      const obj = data as Record<string, unknown>;

      // Common primary fields
      const primaryFields = [
        "balance",
        "did",
        "txHash",
        "factId",
        "claimId",
        "status",
      ];
      for (const field of primaryFields) {
        if (field in obj) {
          console.log(obj[field]);
          return;
        }
      }

      // Fallback: output first value
      const firstValue = Object.values(obj)[0];
      if (firstValue !== undefined) {
        console.log(firstValue);
      }
    }
  } else {
    // Errors still output message
    const errorResponse = response as CLIError;
    console.error(errorResponse.error.message);
  }
}

// ============ Table Formatting ============

/**
 * Format data as ASCII table (for lists)
 */
export function formatTable(
  headers: string[],
  rows: string[][],
  mode: OutputMode,
): void {
  if (mode === "json") {
    // In JSON mode, this shouldn't be called directly
    return;
  }

  if (mode === "quiet") {
    // Just output first column values
    rows.forEach((row) => console.log(row[0]));
    return;
  }

  // Calculate column widths
  const widths = headers.map((h, i) => {
    const dataMax = Math.max(...rows.map((r) => (r[i] || "").length));
    return Math.max(h.length, dataMax);
  });

  // Header
  const headerRow = headers.map((h, i) => h.padEnd(widths[i])).join(" | ");
  const separator = widths.map((w) => "─".repeat(w)).join("─┼─");

  console.log(headerRow);
  console.log(separator);

  // Data rows
  rows.forEach((row) => {
    const dataRow = row
      .map((cell, i) => (cell || "").padEnd(widths[i]))
      .join(" | ");
    console.log(dataRow);
  });
}

// ============ Progress Output ============

/**
 * Output progress indicator (human mode only)
 */
export function outputProgress(message: string, mode: OutputMode): void {
  if (mode === "human") {
    process.stderr.write(`${message}...\r`);
  }
}

/**
 * Clear progress indicator
 */
export function clearProgress(mode: OutputMode): void {
  if (mode === "human") {
    process.stderr.write("\r" + " ".repeat(60) + "\r");
  }
}
