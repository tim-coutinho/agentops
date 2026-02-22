# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 2.x     | ✅ Active  |
| 1.x     | ⚠️ Critical fixes only |
| < 1.0   | ❌ End of life |

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

Email **fullerbt@users.noreply.github.com** with:

1. Description of the vulnerability
2. Steps to reproduce
3. Affected version(s)
4. Potential impact assessment

### What to Expect

| Stage | Timeline |
|-------|----------|
| Acknowledgment | Within 48 hours |
| Initial assessment | Within 5 business days |
| Fix for Critical/High | Within 7 days |
| Fix for Medium/Low | Within 30 days |
| Public disclosure | After fix is released |

### Severity Classification

| Severity | Description | Examples |
|----------|-------------|----------|
| **Critical** | Remote code execution, credential theft | Prompt injection that exfiltrates secrets |
| **High** | Privilege escalation, data exposure | Hook script that bypasses safety guards |
| **Medium** | Information disclosure, denial of service | Unsafe command patterns in skill templates |
| **Low** | Minor information leak, best practice violation | Verbose error messages revealing paths |

## Scope

This repository contains Claude Code plugins — text-based configuration files (skills, agents, hooks) that instruct Claude how to behave, plus a Go CLI (`ao`).

### In Scope

- Prompt injection vulnerabilities in skill/agent definitions
- Unsafe bash commands in hook scripts
- Credential exposure in examples or templates
- Command injection in the `ao` CLI
- Dependency vulnerabilities in Go modules
- Unsafe file operations in scripts

### Out of Scope

- Claude Code CLI vulnerabilities → report to [Anthropic](https://www.anthropic.com/responsible-disclosure)
- General Claude model behavior → report to [Anthropic](https://www.anthropic.com/responsible-disclosure)
- Social engineering attacks
- Vulnerabilities requiring physical access

## Safe Harbor

We consider security research conducted in good faith to be authorized. We will not pursue legal action against researchers who:

- Make a good faith effort to avoid privacy violations, data destruction, and service disruption
- Report vulnerabilities promptly and provide sufficient detail to reproduce
- Do not exploit vulnerabilities beyond what is necessary to demonstrate the issue
- Do not publicly disclose vulnerabilities before a fix is available

## Disclosure Process

1. **Reporter** submits vulnerability via email
2. **Maintainer** acknowledges receipt within 48 hours
3. **Maintainer** assesses severity and confirms timeline
4. **Maintainer** develops and tests fix
5. **Maintainer** releases fix and publishes advisory
6. **Reporter** credited in release notes (unless anonymity requested)

## Acknowledgments

We gratefully acknowledge security researchers who help keep AgentOps safe. Contributors will be credited in release notes unless they prefer to remain anonymous.
