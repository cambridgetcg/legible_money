/**
 * Automated Security Scanner
 *
 * Performs automated security analysis on code changes. Detects
 * common vulnerabilities, backdoor patterns, and obfuscation.
 *
 * PP HIERARCHY:
 *   TRUTH: Pattern detection based on known vulnerability signatures
 *   UNDERSTANDING: Clear classification of security issues
 *   BEAUTY: Composable scanner architecture
 *   JUSTICE: Appropriate severity levels
 *   CREATIVITY: AI-assisted pattern detection for novel threats
 */

import type { DID } from "../../core/agents/identity/did.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";

// ============ Scanner Configuration ============

/**
 * Scan levels with increasing depth and cost.
 */
export type ScanLevel =
  | "basic"
  | "full"
  | "full_formal"
  | "full_formal_external";

/**
 * Scanner configuration.
 */
export interface ScannerConfig {
  level: ScanLevel;
  maxFileSize: number; // bytes
  timeout: number; // ms
  parallelScans: number;
}

export const DEFAULT_SCANNER_CONFIG: ScannerConfig = {
  level: "full",
  maxFileSize: 1024 * 1024, // 1MB
  timeout: 60000, // 60 seconds
  parallelScans: 4,
};

// ============ Pattern Definitions ============

/**
 * Vulnerability pattern to detect.
 */
export interface VulnerabilityPattern {
  id: string;
  name: string;
  description: string;
  severity: "critical" | "high" | "medium" | "low";
  pattern: RegExp;
  fileTypes: string[]; // File extensions to check
  falsePositiveHints: string[]; // Patterns that indicate false positive
}

/**
 * Known vulnerability patterns.
 */
export const VULNERABILITY_PATTERNS: VulnerabilityPattern[] = [
  // Critical patterns
  {
    id: "CRIT-001",
    name: "Hardcoded Private Key",
    description: "Private key hardcoded in source code",
    severity: "critical",
    pattern:
      /(?:private_?key|secret_?key|priv_?key)\s*[:=]\s*['"`][a-fA-F0-9]{64,}['"`]/gi,
    fileTypes: [".ts", ".js", ".json", ".env"],
    falsePositiveHints: ["test", "mock", "example", "sample"],
  },
  {
    id: "CRIT-002",
    name: "Eval with User Input",
    description: "Dynamic code execution with potentially untrusted input",
    severity: "critical",
    pattern: /eval\s*\(\s*(?:[^)]*(?:input|param|arg|req|body|query)[^)]*)\)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: [],
  },
  {
    id: "CRIT-003",
    name: "Command Injection",
    description: "Shell command execution with user input",
    severity: "critical",
    pattern: /(?:exec|spawn|execSync|spawnSync)\s*\([^)]*\+[^)]*\)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["test"],
  },

  // High severity
  {
    id: "HIGH-001",
    name: "Prototype Pollution",
    description: "Potential prototype pollution vulnerability",
    severity: "high",
    pattern:
      /\[['"`]__proto__['"`]\]|\[['"`]constructor['"`]\]|\[['"`]prototype['"`]\]/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["test", "check", "validate"],
  },
  {
    id: "HIGH-002",
    name: "SQL Injection",
    description: "SQL query built with string concatenation",
    severity: "high",
    pattern:
      /(?:SELECT|INSERT|UPDATE|DELETE|DROP)\s+.*\+\s*(?:req|input|param|arg)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["test", "mock"],
  },
  {
    id: "HIGH-003",
    name: "Weak Cryptography",
    description: "Use of weak cryptographic algorithms",
    severity: "high",
    pattern:
      /(?:createHash|createCipher)\s*\(\s*['"`](?:md5|sha1|des|rc4)['"`]\s*\)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["legacy", "compat", "checksum"],
  },
  {
    id: "HIGH-004",
    name: "Unvalidated Redirect",
    description: "Redirect to user-controlled URL",
    severity: "high",
    pattern: /redirect\s*\(\s*(?:req|input|param|arg)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: [],
  },

  // Medium severity
  {
    id: "MED-001",
    name: "Missing Error Handling",
    description: "Async operation without error handling",
    severity: "medium",
    pattern: /\.then\s*\([^)]+\)\s*(?!\.catch)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: [],
  },
  {
    id: "MED-002",
    name: "Sensitive Data Logging",
    description: "Potentially sensitive data in logs",
    severity: "medium",
    pattern:
      /console\.(?:log|info|debug)\s*\([^)]*(?:password|token|secret|key|credential)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["redact", "mask", "hide"],
  },
  {
    id: "MED-003",
    name: "Integer Overflow",
    description: "Potential integer overflow in arithmetic",
    severity: "medium",
    pattern:
      /(?:balance|amount|stake|value)\s*[+\-*]\s*(?:balance|amount|stake|value)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["bigint", "BigInt", "safe"],
  },

  // Low severity
  {
    id: "LOW-001",
    name: "TODO/FIXME Comment",
    description: "Unresolved TODO or FIXME",
    severity: "low",
    pattern: /\/\/\s*(?:TODO|FIXME|HACK|XXX):/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: [],
  },
  {
    id: "LOW-002",
    name: "Debug Code",
    description: "Debug code left in source",
    severity: "low",
    pattern: /console\.(?:log|debug)\s*\(\s*['"`]debug/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: [],
  },
];

/**
 * Backdoor patterns to detect.
 */
export const BACKDOOR_PATTERNS: VulnerabilityPattern[] = [
  {
    id: "BACK-001",
    name: "Hidden Admin Access",
    description: "Hardcoded admin credentials or bypass",
    severity: "critical",
    pattern: /(?:admin|root|superuser)\s*[:=]\s*['"`][^'"]+['"`]/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["test", "example", "env"],
  },
  {
    id: "BACK-002",
    name: "Conditional Bypass",
    description: "Suspicious conditional that could bypass security",
    severity: "critical",
    pattern:
      /if\s*\(\s*(?:process\.env\.DEBUG|isDebug|debugMode)\s*\)\s*(?:return|throw)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["test"],
  },
  {
    id: "BACK-003",
    name: "Obfuscated Code",
    description: "Potentially obfuscated or encoded code",
    severity: "high",
    pattern: /atob\s*\(\s*['"`][A-Za-z0-9+/=]{50,}['"`]\s*\)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: [],
  },
  {
    id: "BACK-004",
    name: "External Data Exfiltration",
    description: "Suspicious external network call",
    severity: "high",
    pattern:
      /fetch\s*\(\s*['"`](?:http|https):\/\/(?!localhost|127\.0\.0\.1)/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["api", "rpc", "test"],
  },
];

/**
 * Obfuscation patterns.
 */
export const OBFUSCATION_PATTERNS: VulnerabilityPattern[] = [
  {
    id: "OBF-001",
    name: "Heavy String Encoding",
    description: "Excessive string encoding/decoding",
    severity: "medium",
    pattern: /(?:btoa|atob|encodeURI|decodeURI){2,}/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: ["util", "helper"],
  },
  {
    id: "OBF-002",
    name: "Dynamic Property Access",
    description: "Computed property names that could hide behavior",
    severity: "low",
    pattern: /\[\s*['"`][a-z]+['"`]\s*\+\s*['"`][a-z]+['"`]\s*\]/gi,
    fileTypes: [".ts", ".js"],
    falsePositiveHints: [],
  },
];

// ============ Scanner Results ============

/**
 * Individual finding from a scan.
 */
export interface ScanFinding {
  patternId: string;
  patternName: string;
  severity: "critical" | "high" | "medium" | "low";
  description: string;
  file: string;
  line: number;
  column: number;
  matchedText: string;
  isPotentialFalsePositive: boolean;
  falsePositiveReason?: string;
}

/**
 * Complexity metrics for a file.
 */
export interface FileComplexity {
  file: string;
  lines: number;
  functions: number;
  cyclomaticComplexity: number;
  dependencies: string[];
}

/**
 * Complete scan result.
 */
export interface SecurityScanResult {
  scanId: string;
  scanLevel: ScanLevel;
  startedAt: number;
  completedAt: number;

  // Findings
  findings: ScanFinding[];
  findingsBySeverity: {
    critical: number;
    high: number;
    medium: number;
    low: number;
  };

  // Complexity
  complexity: FileComplexity[];
  averageComplexity: number;

  // Dependencies
  dependencyIssues: DependencyIssue[];

  // Overall verdict
  riskLevel: "critical" | "high" | "medium" | "low" | "clean";
  humanReviewRequired: boolean;
  blockingIssues: ScanFinding[];
  summary: string;
}

export interface DependencyIssue {
  package: string;
  version: string;
  vulnerability: string;
  severity: "critical" | "high" | "medium" | "low";
}

// ============ Scanner Implementation ============

/**
 * Security scanner.
 */
export class SecurityScanner {
  private config: ScannerConfig;

  constructor(config: ScannerConfig = DEFAULT_SCANNER_CONFIG) {
    this.config = config;
  }

  /**
   * Scan files for security issues.
   */
  async scan(
    files: Array<{ path: string; content: string }>,
  ): Promise<SecurityScanResult> {
    const scanId = `scan-${Date.now()}-${sha512_256Hex(JSON.stringify(files.map((f) => f.path))).slice(0, 8)}`;
    const startedAt = Date.now();

    const findings: ScanFinding[] = [];
    const complexity: FileComplexity[] = [];

    // Select patterns based on scan level
    const patterns = this.getPatternsForLevel(this.config.level);

    // Scan each file
    for (const file of files) {
      // Check file size
      if (file.content.length > this.config.maxFileSize) {
        continue; // Skip oversized files
      }

      // Check file type
      const ext = this.getFileExtension(file.path);

      // Run pattern matching
      const fileFindings = this.scanFile(
        file.path,
        file.content,
        ext,
        patterns,
      );
      findings.push(...fileFindings);

      // Calculate complexity
      const fileComplexity = this.calculateComplexity(file.path, file.content);
      complexity.push(fileComplexity);
    }

    // Count by severity
    const findingsBySeverity = {
      critical: findings.filter((f) => f.severity === "critical").length,
      high: findings.filter((f) => f.severity === "high").length,
      medium: findings.filter((f) => f.severity === "medium").length,
      low: findings.filter((f) => f.severity === "low").length,
    };

    // Calculate average complexity
    const averageComplexity =
      complexity.length > 0
        ? complexity.reduce((sum, c) => sum + c.cyclomaticComplexity, 0) /
          complexity.length
        : 0;

    // Determine risk level
    let riskLevel: "critical" | "high" | "medium" | "low" | "clean";
    if (findingsBySeverity.critical > 0) {
      riskLevel = "critical";
    } else if (findingsBySeverity.high > 0) {
      riskLevel = "high";
    } else if (findingsBySeverity.medium > 0) {
      riskLevel = "medium";
    } else if (findingsBySeverity.low > 0) {
      riskLevel = "low";
    } else {
      riskLevel = "clean";
    }

    // Identify blocking issues (critical always blocks; high blocks unless false positive)
    const blockingIssues = findings.filter(
      (f) =>
        f.severity === "critical" ||
        (f.severity === "high" && !f.isPotentialFalsePositive),
    );

    // Determine if human review is required
    const humanReviewRequired =
      riskLevel === "critical" ||
      riskLevel === "high" ||
      blockingIssues.length > 0;

    return {
      scanId,
      scanLevel: this.config.level,
      startedAt,
      completedAt: Date.now(),
      findings,
      findingsBySeverity,
      complexity,
      averageComplexity,
      dependencyIssues: [], // TODO: Implement dependency scanning
      riskLevel,
      humanReviewRequired,
      blockingIssues,
      summary: this.generateSummary(
        findingsBySeverity,
        riskLevel,
        blockingIssues.length,
      ),
    };
  }

  /**
   * Scan a single file.
   */
  private scanFile(
    filePath: string,
    content: string,
    fileExt: string,
    patterns: VulnerabilityPattern[],
  ): ScanFinding[] {
    const findings: ScanFinding[] = [];
    const lines = content.split("\n");

    for (const pattern of patterns) {
      // Check if pattern applies to this file type
      if (!pattern.fileTypes.includes(fileExt)) {
        continue;
      }

      // Reset regex state
      pattern.pattern.lastIndex = 0;

      // Search for matches
      let match;
      while ((match = pattern.pattern.exec(content)) !== null) {
        // Find line number
        const beforeMatch = content.slice(0, match.index);
        const lineNumber = beforeMatch.split("\n").length;
        const lineStart = beforeMatch.lastIndexOf("\n") + 1;
        const column = match.index - lineStart;

        // Check for false positives
        const line = lines[lineNumber - 1] || "";
        const { isFalsePositive, reason } = this.checkFalsePositive(
          line,
          pattern.falsePositiveHints,
        );

        findings.push({
          patternId: pattern.id,
          patternName: pattern.name,
          severity: pattern.severity,
          description: pattern.description,
          file: filePath,
          line: lineNumber,
          column,
          matchedText: match[0].slice(0, 100), // Truncate long matches
          isPotentialFalsePositive: isFalsePositive,
          falsePositiveReason: reason,
        });
      }
    }

    return findings;
  }

  /**
   * Check if a finding is likely a false positive.
   */
  private checkFalsePositive(
    line: string,
    hints: string[],
  ): { isFalsePositive: boolean; reason?: string } {
    const lowerLine = line.toLowerCase();

    for (const hint of hints) {
      if (lowerLine.includes(hint.toLowerCase())) {
        return { isFalsePositive: true, reason: `Contains '${hint}'` };
      }
    }

    return { isFalsePositive: false };
  }

  /**
   * Calculate complexity metrics for a file.
   */
  private calculateComplexity(
    filePath: string,
    content: string,
  ): FileComplexity {
    const lines = content.split("\n").length;

    // Count functions (simple heuristic)
    const functionMatches = content.match(/(?:function|=>|async\s+function)/g);
    const functions = functionMatches ? functionMatches.length : 0;

    // Cyclomatic complexity (simplified)
    const branchKeywords = content.match(
      /\b(?:if|else|for|while|switch|case|catch|&&|\|\||\?)/g,
    );
    const cyclomaticComplexity =
      (branchKeywords ? branchKeywords.length : 0) + 1;

    // Extract dependencies (imports)
    const importMatches = content.match(
      /(?:import|require)\s*\(?['"`]([^'"]+)['"`]\)?/g,
    );
    const dependencies = importMatches
      ? importMatches
          .map((m) => {
            const match = m.match(/['"`]([^'"]+)['"`]/);
            return match ? match[1] : "";
          })
          .filter(Boolean)
      : [];

    return {
      file: filePath,
      lines,
      functions,
      cyclomaticComplexity,
      dependencies,
    };
  }

  /**
   * Get patterns for scan level.
   */
  private getPatternsForLevel(level: ScanLevel): VulnerabilityPattern[] {
    switch (level) {
      case "basic":
        // Only critical patterns
        return [
          ...VULNERABILITY_PATTERNS.filter((p) => p.severity === "critical"),
          ...BACKDOOR_PATTERNS.filter((p) => p.severity === "critical"),
        ];

      case "full":
        // All standard patterns
        return [...VULNERABILITY_PATTERNS, ...BACKDOOR_PATTERNS];

      case "full_formal":
      case "full_formal_external":
        // All patterns including obfuscation
        return [
          ...VULNERABILITY_PATTERNS,
          ...BACKDOOR_PATTERNS,
          ...OBFUSCATION_PATTERNS,
        ];
    }
  }

  /**
   * Get file extension.
   */
  private getFileExtension(filePath: string): string {
    const match = filePath.match(/\.[^.]+$/);
    return match ? match[0] : "";
  }

  /**
   * Generate human-readable summary.
   */
  private generateSummary(
    counts: { critical: number; high: number; medium: number; low: number },
    riskLevel: string,
    blockingCount: number,
  ): string {
    const total = counts.critical + counts.high + counts.medium + counts.low;

    if (total === 0) {
      return "No security issues detected";
    }

    const parts: string[] = [];
    if (counts.critical > 0) parts.push(`${counts.critical} critical`);
    if (counts.high > 0) parts.push(`${counts.high} high`);
    if (counts.medium > 0) parts.push(`${counts.medium} medium`);
    if (counts.low > 0) parts.push(`${counts.low} low`);

    let summary = `Found ${total} issue(s): ${parts.join(", ")}. `;
    summary += `Overall risk: ${riskLevel}. `;

    if (blockingCount > 0) {
      summary += `${blockingCount} blocking issue(s) require attention.`;
    }

    return summary;
  }
}

/**
 * Create a scanner with default config.
 */
export function createScanner(level: ScanLevel = "full"): SecurityScanner {
  return new SecurityScanner({ ...DEFAULT_SCANNER_CONFIG, level });
}

/**
 * Quick scan for critical issues only.
 */
export async function quickScan(
  files: Array<{ path: string; content: string }>,
): Promise<SecurityScanResult> {
  const scanner = createScanner("basic");
  return scanner.scan(files);
}

/**
 * Full scan with all patterns.
 */
export async function fullScan(
  files: Array<{ path: string; content: string }>,
): Promise<SecurityScanResult> {
  const scanner = createScanner("full");
  return scanner.scan(files);
}
