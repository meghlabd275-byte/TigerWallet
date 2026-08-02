# TigerWallet Bug Bounty Program

## Overview

TigerWallet invites security researchers and ethical hackers to help keep our users safe by reporting vulnerabilities in our systems. We welcome responsible disclosure and offer rewards for qualifying findings.

## Program Scope

### In-Scope Assets

| Category | Targets | Severity |
|----------|---------|----------|
| Smart Contracts | Token contracts, DeFi integrations | All |
| Web Applications | tigerwallet.com, app.tigerwallet.com | All |
| Mobile Apps | iOS, Android apps | All |
| APIs | api.tigerwallet.com | All |
| Browser Extensions | Chrome, Firefox, Edge | All |
| Backend Services | All production services | All |

### Out of Scope

- Social engineering
- Physical security
- DDoS attacks
- Third-party services
- Previously reported issues
- Low-severity UI bugs

## Rewards

### Vulnerability Severity

| Severity | Bounty Range | Examples |
|----------|--------------|----------|
| **Critical** | $10,000 - $50,000 | Key theft, smart contract exploit, data breach |
| **High** | $2,500 - $10,000 | Account takeover, XSS with impact, injection |
| **Medium** | $500 - $2,500 | Stored XSS, CSRF, information disclosure |
| **Low** | $100 - $500 | Minor bugs, best practice violations |
| **Info** | Thanks only | Spelling errors, minor suggestions |

### Bonus Factors

- **Working Exploit**: +50%
- **Full Chain Attack**: +100%
- **Novel Vulnerability**: +25%
- **Quality Report**: +25%
- **Quick Fix Turnaround**: +10%

## Reporting Process

### 1. Submit Report

Submit via: https://hackerone.com/tigerwallet

Required information:
```
Title: [Brief description]
Description: [Detailed explanation]
Steps to Reproduce: [Clear reproduction steps]
Impact: [Security impact assessment]
Evidence: [Screenshots, logs, PoC code]
```

### 2. Acknowledgment

We will acknowledge within **24 hours** and provide a timeline for response.

### 3. Investigation

Our security team will:
- Reproduce the issue
- Assess severity
- Determine root cause
- Plan remediation

### 4. Resolution

- Critical: Fix within **24 hours**
- High: Fix within **7 days**
- Medium: Fix within **30 days**
- Low: Fix within **90 days**

### 5. Disclosure

- Private disclosure until fix
- Public disclosure after patch
- Credit in security hall of fame

## Rules

### Do

- [ ] Report responsibly
- [ ] Provide detailed reproduction steps
- [ ] Keep communication private
- [ ] Delete test data after
- [ ] Give reasonable time to fix
- [ ] Accept our severity decision

### Don't

- [ ] Exploit for financial gain
- [ ] Access other user data
- [ ] Modify or delete data
- [ ] Attack our infrastructure
- [ ] Publicize before fix
- [ ] Use automated scanners excessively

## Safe Harbor

We promise:
- **No Legal Action**: No lawsuit for good-faith research
- **Bug Hall of Fame**: Public recognition
- **Swift Response**: Quick triage and fix
- **Fair Rewards**: Based on impact, not effort

## Recognition

### Hall of Fame

| Researcher | Vulnerabilities | Date |
|------------|----------------|------|
| [Name] | Critical x2 | 2024-XX |
| [Name] | High x3 | 2024-XX |
| [Name] | Medium x5 | 2024-XX |

### Tier System

| Tier | Requirements | Benefits |
|------|--------------|----------|
| Bronze | 1 Low+ | Name in Hall of Fame |
| Silver | 3 Medium+ | Early access, swag |
| Gold | 1 High+ | Private program access |
| Platinum | 1 Critical+ | Dedicated thanks, conference sponsorship |

## Contact

- **HackerOne**: https://hackerone.com/tigerwallet
- **Email**: security@tigerwallet.com
- **PGP**: [security-pgp-key.asc](./security-pgp-key.asc)

**Emergency**: For critical vulnerabilities requiring immediate attention, contact security@tigerwallet.com with "URGENT" in subject line.

---

*Effective Date: 2026-08-02*
*Version: 1.0*
