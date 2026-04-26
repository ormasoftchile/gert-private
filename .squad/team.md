# Squad Team

> Runbook engine and tooling for safe, auditable operations.

## Coordinator

| Name | Role | Notes |
|------|------|-------|
| Squad | Coordinator | Routes work, enforces handoffs and reviewer gates. |

## Members

| Name | Role | Charter | Status |
|------|------|---------|--------|
| Leslie | LaTeX Specialist | [charter](.squad/agents/leslie/charter.md) | 📝 Active |
| Dennis | CS Researcher | [charter](.squad/agents/dennis/charter.md) | 🔬 Active |
| Ken | Software Architect | [charter](.squad/agents/ken/charter.md) | 🏗️ Active |
| Barbara | Integrations Specialist | [charter](.squad/agents/barbara/charter.md) | 🔌 Active |
| Brian | Go Programmer | [charter](.squad/agents/brian/charter.md) | 🔧 Active |
| John | YAML/Schema Specialist | [charter](.squad/agents/john/charter.md) | 📋 Active |
| Ada | iOS Engineer | [charter](.squad/agents/ada/charter.md) | 🍎 Active |
| James | Android Engineer | [charter](.squad/agents/james/charter.md) | 🤖 Active |
| Scribe | Session Logger | [charter](.squad/agents/scribe/charter.md) | 📋 Silent |
| Ralph | Work Monitor | — | 🔄 Monitor |

## Coding Agent

<!-- copilot-auto-assign: false -->

| Name | Role | Charter | Status |
|------|------|---------|--------|
| @copilot | Coding Agent | — | 🤖 Coding Agent |

### Capabilities

**🟢 Good fit — auto-route when enabled:**
- Bug fixes with clear reproduction steps
- Test coverage (adding missing tests, fixing flaky tests)
- Lint/format fixes and code style cleanup
- Dependency updates and version bumps
- Small isolated features with clear specs
- Boilerplate/scaffolding generation
- Documentation fixes and README updates

**🟡 Needs review — route to @copilot but flag for squad member PR review:**
- Medium features with clear specs and acceptance criteria
- Refactoring with existing test coverage
- API endpoint additions following established patterns
- Migration scripts with well-defined schemas

**🔴 Not suitable — route to squad member instead:**
- Architecture decisions and system design
- Multi-system integration requiring coordination
- Ambiguous requirements needing clarification
- Security-critical changes (auth, encryption, access control)
- Performance-critical paths requiring benchmarking
- Changes requiring cross-team discussion

## Project Context

- **Owner:** ormasoftchile
- **Stack:** Go, TypeScript, C#, YAML, Azure
- **Description:** YAML-driven runbook orchestration with extensible tools/providers and governance.
- **Created:** 2026-03-17
