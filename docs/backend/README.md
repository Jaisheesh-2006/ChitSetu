# Backend Documentation

## Overview
The backend is a Go + Gin API located at `services/backend`.
It orchestrates:
- Auth and session tokens
- User profile + KYC state transitions
- Fund lifecycle and member approvals
- Payments and contribution updates
- Auction engine, realtime participant tracking, and payout flow
- Chat messages

## Runtime Entry Point
- `services/backend/cmd/server/main.go`

Startup sequence (high level):
1. Load environment variables
2. Connect to MongoDB
3. Ensure indexes
4. Start cron/schedulers
5. Register API routes and WebSocket endpoint

## Main Modules
- `internal/auth`: registration, login, refresh, reset flows
- `internal/users`: profile, PAN verify, history generation, ML scoring persistence
- `internal/chitfund`: fund creation, applications, approvals, contributions view
- `internal/payments`: payment session/order/verify logic + reminder cron
- `internal/auction`: waiting room, bidding rules, finalize, payout tracking
- `internal/chat`: fund and auction chat persistence
- `internal/ws`: websocket hub/room manager

## Route Map (Grouped)
- `/auth/*`: register, login, refresh, verify, forgot/reset
- `/user/*`: profile upsert and KYC action endpoints
- `/users/*`: profile, risk score, kyc status, my funds, my contributions
- `/funds/*`: fund operations, members, applications, auction and chat routes
- `/payments/*`: payment session, order creation, payment verification
- `/ws/funds/:id`: realtime websocket endpoint

## Data Store
MongoDB is the operational source of truth.
Important collections include:
- `users`, `user_profiles`, `auth_sessions`
- `funds`, `fund_members`, `contributions`
- `payment_sessions`, `payment_orders`
- `auction_sessions`, `auction_results`, `payouts`
- `chat_messages`

Index creation is handled by `pkg/database/mongo.go` at startup.

## KYC Data Persistence
KYC-related values are stored in `users` document paths:
- `kyc.status`
- `kyc.verified_at`
- `kyc.bank_account`
- `kyc.ifsc_code`
- `credit.cibil_score`
- `credit.synthetic_history`
- `credit.score`, `credit.risk_category`, `credit.default_probability`, `credit.checked_at`

## Auction Rules (Current)
- Allowed increments: 10, 100, 200
- Same user cannot place consecutive bids
- Discount is global cumulative (auction-wide)
- Discount cap: 50 percent of monthly pool
- If cap is reached, backend auto-finalizes immediately
- Otherwise finalization happens after 20s idle window by scheduler

## Background Jobs
- Payment reminder cron (daily)
- Underfilled fund cleanup cron (daily)
- Auction scheduler tick (every second)
- Payout retry job (triggered from auction scheduler)

## External Integrations
- Supabase token verification for OAuth path
- Razorpay for order creation and payment signature verification
- Resend for reminder and auth email notifications
- ML service (FastAPI) for credit/history synthesis and risk scoring

## Local Development
```bash
cd services/backend
go mod download
go run cmd/server/main.go
```

## Test / Compile
```bash
cd services/backend
go test ./...
```
