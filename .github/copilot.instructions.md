---
description: "Repository-wide coding instructions for the ChitSetu monorepo"
applyTo: "**"
---

# ChitSetu Repository Instructions

These instructions apply across the full repository.
Use them together with specialized instruction files in [.github/instructions/go.instructions.md](.github/instructions/go.instructions.md) and [.github/instructions/database.instructions.md](.github/instructions/database.instructions.md).

## Project Scope

- Monorepo for a decentralized chit fund platform
- Services in this repo:
  - Go backend API
  - Next.js frontend
  - Solidity contracts with Hardhat
  - SpacetimeDB auction service
  - Python ML service
- Domain is fintech + KYC, so correctness and safety take priority over novelty

## Global Engineering Principles

- Keep solutions simple, modular, and hackathon-executable
- Preserve existing architecture and naming conventions
- Prefer explicitness over hidden behavior
- Fail clearly on errors; do not swallow errors
- Avoid broad refactors when implementing focused tasks
- Make the smallest safe change that satisfies requirements

## Monorepo Boundaries

- Keep business logic inside the service that owns it
- Do not duplicate domain logic across services
- Use clear contracts between services (HTTP APIs, payload schemas, event/result messages)
- Keep integration points explicit and documented

## API and Contract Rules

- Use consistent JSON request and response shapes
- Validate external input at service boundaries
- Return stable status codes and meaningful error messages
- Do not leak internal stack traces, SQL details, or secrets in responses
- Keep request and response models backward compatible unless breaking change is requested

## Data and Financial Safety

- Treat persistent storage as source of truth
- Protect multi-step financial flows with transaction-safe logic in the owning service
- Enforce invariants through schema constraints where possible
- Use idempotency keys for payment and webhook operations
- Keep status transitions explicit and validated

## Security and Privacy

- Never commit secrets, keys, or credentials
- Load secrets from environment variables
- Protect authentication state with short-lived access tokens and revocable refresh sessions
- Treat PAN and identity data as sensitive
- Apply least-privilege design for service integrations

## Service-Specific Guidance

### Go Backend

- Follow [.github/instructions/go.instructions.md](.github/instructions/go.instructions.md)
- Follow [.github/instructions/database.instructions.md](.github/instructions/database.instructions.md) for SQL and financial correctness
- Keep handlers thin and push logic into service or repository layers

### Frontend (Next.js)

- Keep UI logic separate from transport logic
- Use typed API clients and explicit loading or error states
- Avoid coupling UI components to backend internals

### Smart Contracts (Solidity)

- Keep only financial escrow and payout enforcement on-chain
- Minimize contract complexity and surface area
- Treat contract state transitions as irreversible and safety-critical

### SpacetimeDB Auction Service

- Isolate real-time auction state and bid lifecycle logic here
- Keep auction winner output deterministic and auditable
- Send finalized outcomes to backend through a clear integration path

### ML Service (Python)

- Keep ML service focused on risk scoring only
- Keep model I/O schemas stable and documented
- Return deterministic, bounded outputs for backend consumption

## Testing and Validation

- Run targeted build checks for changed services
- Add or update tests for non-trivial logic
- Validate integration points with realistic payloads
- Prefer deterministic tests over timing-sensitive behavior

## Configuration and Operations

- Keep developer defaults in env example files
- Keep local artifacts and generated files out of version control
- Use docker compose for local multi-service setup when practical
- Keep scripts idempotent and safe to rerun

## AI Edit Expectations

- Explain what changed and why
- Keep diffs focused and avoid unrelated edits
- Note trade-offs and assumptions when relevant
- Preserve existing user changes in dirty working trees
