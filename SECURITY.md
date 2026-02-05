# Security Policy

## Reporting a Vulnerability

The Legible Money team takes security vulnerabilities seriously. We appreciate your efforts to responsibly disclose your findings.

### How to Report

**DO NOT** create a public GitHub issue for security vulnerabilities.

Instead, please report security vulnerabilities through one of these channels:

1. **GitHub Security Advisories** (Preferred)
   - Go to the [Security Advisories](https://github.com/legible-money/legible-money/security/advisories) page
   - Click "Report a vulnerability"
   - Provide details following the template below

2. **Email**
   - Send to: security@legible.money
   - Use PGP encryption if possible (key available on request)

### What to Include

Please include as much of the following information as possible:

- Type of vulnerability (e.g., buffer overflow, SQL injection, XSS)
- Full paths of affected source files
- Location of the affected code (tag/branch/commit or direct URL)
- Step-by-step instructions to reproduce
- Proof-of-concept or exploit code (if possible)
- Impact assessment and potential attack scenarios

### Response Timeline

| Severity | Initial Response | Resolution Target |
|----------|------------------|-------------------|
| Critical | 4 hours | 24-48 hours |
| High | 24 hours | 1 week |
| Medium | 48 hours | 2 weeks |
| Low | 72 hours | Next release |

### What to Expect

1. **Acknowledgment**: We will acknowledge receipt within the response time above
2. **Assessment**: We will assess the vulnerability and determine severity
3. **Updates**: We will keep you informed of our progress
4. **Credit**: We will credit you in the security advisory (unless you prefer anonymity)
5. **Disclosure**: We follow coordinated disclosure practices

### Severity Classification

**Critical**: Remote code execution, consensus manipulation, key extraction, fund theft
**High**: Denial of service, significant information disclosure, authentication bypass
**Medium**: Limited information disclosure, non-critical logic errors
**Low**: Minor issues, hardening improvements

### Security-Critical Code Paths

The following code paths require additional scrutiny:

- `core/crypto/` - Cryptographic operations
- `consensus/pot/` - Consensus engine
- `state/transitions.ts` - State transitions
- `agents/identity/` - Key management
- `protocol/rpc/` - Network-exposed interfaces

### Safe Harbor

We consider security research conducted in accordance with this policy to be:

- Authorized in accordance with the Computer Fraud and Abuse Act (CFAA)
- Exempt from DMCA restrictions on circumvention
- Lawful and beneficial to the security of the protocol

We will not initiate legal action against researchers who:

- Act in good faith to avoid harm to users
- Report vulnerabilities promptly
- Allow reasonable time for remediation before disclosure
- Do not exploit vulnerabilities beyond what is necessary to demonstrate them

### Bug Bounty

We are working on establishing a formal bug bounty program. Until then, significant findings may be eligible for recognition or rewards at our discretion.

## Supported Versions

| Version | Supported |
|---------|-----------|
| main branch | Yes |
| release/* branches | Yes |
| Older releases | Case-by-case |

## Security Best Practices

When contributing to Legible Money, please follow these security guidelines:

1. **Never commit secrets** - No private keys, API tokens, or credentials
2. **Use domain separation** - All hashing must use domain-separated functions
3. **Avoid non-determinism** - No `Date.now()`, `Math.random()`, or unordered iterations in consensus
4. **Validate inputs** - All external inputs must be validated
5. **Handle errors safely** - Errors should not leak internal state
6. **Sign commits** - All commits to protected branches must be GPG-signed

## Previous Security Advisories

No security advisories have been published yet.

---

Thank you for helping keep Legible Money secure.
