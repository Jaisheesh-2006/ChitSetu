# Frontend Documentation

## Overview
The frontend is a Next.js App Router project located at `services/frontend`.
It handles:
- Authentication UI (email/password and Google OAuth)
- Dashboard, profile/KYC workflows, and fund lifecycle screens
- Auction room realtime UI
- Payment checkout and verification UX

## Tech Stack
- Next.js 16 (App Router)
- React 19 + TypeScript
- MUI + Emotion
- Framer Motion
- Supabase JS (OAuth session bootstrap)

## Important Folders
- `services/frontend/app`: Route pages (`/login`, `/dashboard`, `/funds`, `/fund/[id]`, `/auction/[id]`, `/pay`)
- `services/frontend/components`: Shared UI and feature components
- `services/frontend/hooks/useAuth.tsx`: Auth context and token session state
- `services/frontend/hooks/useAuctionSocket.tsx`: Auction WebSocket client
- `services/frontend/services/api.ts`: Typed REST client wrapper and token refresh flow
- `services/frontend/lib/supabase.ts`: Supabase client bootstrap
- `services/frontend/styles`: Global styling

## Authentication Flows
### Email/Password
1. User submits credentials from UI.
2. Frontend calls backend `/auth/register` or `/auth/login`.
3. Backend returns app JWT token pair.
4. Frontend stores tokens in localStorage.

### Google OAuth
1. User clicks Google OAuth button.
2. Supabase completes OAuth redirect to `/auth/callback`.
3. Callback page calls backend `/auth/verify` with Supabase access token.
4. Backend validates token and issues app JWT token pair.
5. Frontend stores app tokens and routes user to dashboard.

## API and Token Handling
`services/api.ts` provides a central fetch wrapper:
- Sends `Authorization: Bearer <access-token>` automatically.
- On `401`, attempts refresh via `/auth/refresh`.
- If refresh fails, clears tokens and redirects to `/`.

## Realtime Auction Flow
`useAuctionSocket` connects to:
`<NEXT_PUBLIC_WS_URL>/ws/funds/:fundId?token=<access-token>`

Messages used in UI include:
- `auction_started`
- `bidding_started`
- `new_bid`
- `participants`
- `auction_ended`

The client explicitly sends `auction_room_join` when entering the room.

## Environment Variables
Required frontend env values:
- `NEXT_PUBLIC_API_URL`
- `NEXT_PUBLIC_WS_URL`
- `NEXT_PUBLIC_SUPABASE_URL`
- `NEXT_PUBLIC_SUPABASE_ANON_KEY`
- `NEXT_PUBLIC_AUTH_CALLBACK_URL`

## Local Development
```bash
cd services/frontend
npm install
npm run dev
```

## Build
```bash
cd services/frontend
npm run build
npm start
```

If build workers hit memory limits on Windows, run with:
```bash
$env:NODE_OPTIONS='--max-old-space-size=4096'
npm run build
```
