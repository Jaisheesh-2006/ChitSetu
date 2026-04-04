---
description: "Database rules for financial backend (MongoDB + Go)"
applyTo: "**/*.go"
---

# Database Instructions (Critical - Financial System)

This project handles financial flows (contributions, auctions, payouts) and 
identity (KYC). Database correctness is more important than code elegance.

Follow these rules strictly.

---

## Core Principles

- MongoDB is the source of truth
- Never trust application state over database state
- All financial operations must be atomic wherever MongoDB allows
- Prevent inconsistent state at all costs
- Prefer correctness over performance

---

## Transactions (MANDATORY)

MongoDB multi-document transactions are required for any operation involving:
- contributions
- fund membership changes
- auction finalization
- payouts

Pattern:
- Start session via client.StartSession()
- Call session.WithTransaction()
- Validate state inside the transaction
- Perform updates inside the transaction
- Return error to trigger automatic abort if any step fails

Never split financial operations across multiple collection writes 
without a session transaction.

Single-document atomic operations (FindOneAndUpdate) are preferred when 
the entire operation touches only one document — no session needed for those.

---

## Idempotency (CRITICAL)

All payment-related operations must be idempotent.

- Use a payment_ref field as an idempotency key on contribution documents
- Enforce uniqueness via a unique index on payment_ref
- On duplicate payment_ref → return the existing document, do not insert again
- Handle mongo.IsDuplicateKeyError explicitly — never silently ignore it

---

## Atomic Updates (Row-Level Locking Equivalent)

MongoDB has no SELECT FOR UPDATE. Use these patterns instead:

- Use FindOneAndUpdate with a filter that includes the expected current state
  Example: filter on { _id, status: "pending" } when marking as paid
  If 0 documents matched → the state changed under you → treat as failure
- For auction and payout operations, use optimistic concurrency:
  include a version field on documents, increment on every write,
  filter on the expected version — retry on mismatch
- Never fetch a document and update it in two separate operations 
  for financial writes — always use FindOneAndUpdate in a single call

---

## State Machines (STRICT)

Never allow arbitrary status updates.
Validate the transition in the filter itself, not just in application code.

Allowed states and transitions:

### KYC (field on users document)
- pending → pan_verified
- pan_verified → credit_fetched
- credit_fetched → ml_ready

### Contribution
- pending → paid
- pending → failed

### Fund
- open → active
- active → closed

Rules:
- Filter on the current expected status when performing any status update
- If the update matches 0 documents → reject with a domain error
- Never set status directly without encoding the valid transition in the filter

---

## Data Integrity Rules

Enforce these via unique indexes — not just application logic:

- PAN must be unique across all users
  → unique index on users.profile.pan (sparse, since not all users have it yet)
- A user cannot join the same fund twice
  → unique compound index on fund_members (fund_id, user_id)
- A user cannot pay twice for the same cycle
  → unique compound index on contributions (fund_id, user_id, cycle_number)
- Only one auction result per fund per cycle
  → unique compound index on auction_results (fund_id, cycle_number)
- A user can win only once per fund
  → unique compound index on auction_results (fund_id, winner_id)

Handle mongo.IsDuplicateKeyError for every insert that touches these 
collections — return a clear domain error, never a raw MongoDB error.

---

## Monetary Values

- Always store monetary values as int64 (paise/cents) — never float64
- Never perform money calculations using float64
- Convert to decimal for display only, at the response layer
- Always validate amounts > 0 before any DB write

---

## Time Handling

- Always store timestamps as time.Time (which BSON maps to UTC)
- Never trust client-provided timestamps for financial records
- Set created_at, paid_at, joined_at server-side using time.Now().UTC()
- Never use string representations of time in financial documents

---

## Query Patterns

- Always check ModifiedCount / MatchedCount after every UpdateOne or UpdateMany
- If ModifiedCount == 0 → treat as a failure, return a domain error
- Never assume an update succeeded without checking the result
- Use FindOneAndUpdate with ReturnDocument: options.After when you need 
  the updated document — never fetch separately after updating

---

## Error Handling

- Never ignore errors from any MongoDB operation
- Wrap all errors with context: fmt.Errorf("operation description: %w", err)
- Check mongo.IsDuplicateKeyError for all inserts on constrained collections
- Check for mongo.ErrNoDocuments explicitly — return 404, not a 500
- Never return raw MongoDB error messages to API clients

---

## Indexes (Define at Startup)

Create all indexes in code at application startup, not manually.
Use CreateOne with options.Index() and SetUnique / SetSparse as needed.

Required indexes:
- users: unique on email, sparse unique on profile.pan
- funds: on status, on creator_id
- fund_members: unique compound on (fund_id, user_id), on user_id, on fund_id
- contributions: unique compound on (fund_id, user_id, cycle_number), 
  on (user_id, status), on fund_id
- auction_results: unique compound on (fund_id, cycle_number), 
  unique compound on (fund_id, winner_id)

---

## Document Design Rules

- All _id fields are UUID strings — never use MongoDB ObjectIDs
- Embed subdocuments (kyc, credit, profile) only when they are always 
  fetched together with the parent and never queried independently
- Use separate collections (fund_members, contributions) for data that 
  grows unboundedly or needs independent querying
- Never use arrays inside documents for financial records that grow over time
  (e.g. don't store contributions as an array inside the fund document)

---

## Anti-Patterns (DO NOT DO)

- ❌ Multi-collection financial writes without a session transaction
- ❌ Fetch then update in two separate calls for financial operations
- ❌ Using float64 for any monetary value
- ❌ Trusting client-provided timestamps or amounts blindly
- ❌ Updating status without encoding the valid transition in the filter
- ❌ Silently ignoring IsDuplicateKeyError
- ❌ Not checking ModifiedCount after financial updates
- ❌ Storing contribution or membership records as arrays inside parent documents
- ❌ Using MongoDB ObjectIDs — always UUID strings
- ❌ Returning raw MongoDB errors to API clients

---

## Expected Code Behavior

Generated code must:
- Use session transactions for any write touching more than one collection
- Use FindOneAndUpdate with state-based filters for single-document financial writes
- Validate state transitions via the query filter, not just if-statements
- Handle IsDuplicateKeyError and ErrNoDocuments explicitly on every operation
- Check ModifiedCount after every update
- Be safe under concurrent requests — two goroutines doing the same operation 
  simultaneously must not corrupt data

---

## Mental Model

Before every database write, answer:
- Can this be run twice safely? (idempotency)
- Can two users do this at the same time safely? (concurrency)
- Can this leave the system in an inconsistent state? (atomicity)

If any answer is unclear → the implementation is wrong.