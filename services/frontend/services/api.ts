const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

/* ── Token helpers ── */

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("chitsetu-access-token");
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("chitsetu-refresh-token");
}

export function setTokens(access: string, refresh: string) {
  localStorage.setItem("chitsetu-access-token", access);
  localStorage.setItem("chitsetu-refresh-token", refresh);
}

export function clearTokens() {
  localStorage.removeItem("chitsetu-access-token");
  localStorage.removeItem("chitsetu-refresh-token");
}

export function getCurrentUserId(): string | null {
  const token = getAccessToken();
  if (!token) return null;
  try {
    const payload = JSON.parse(atob(token.split(".")[1]));
    return payload.user_id || payload.sub || null;
  } catch {
    return null;
  }
}

/* ── Core fetch wrapper ── */

interface FetchOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  auth?: boolean;
}

async function fetchAPI<T = unknown>(
  path: string,
  options: FetchOptions = {},
): Promise<T> {
  const { body, auth = true, ...rest } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((rest.headers as Record<string, string>) || {}),
  };

  if (auth) {
    const token = getAccessToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...rest,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  // Attempt token refresh on 401
  if (res.status === 401 && auth) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      headers["Authorization"] = `Bearer ${getAccessToken()}`;
      const retryRes = await fetch(`${API_BASE}${path}`, {
        ...rest,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });
      if (!retryRes.ok) {
        const err = await retryRes
          .json()
          .catch(() => ({ error: "Request failed" }));
        throw new Error(
          err.error || `Request failed with status ${retryRes.status}`,
        );
      }
      return retryRes.json();
    }
    clearTokens();
    if (typeof window !== "undefined") {
      window.location.href = "/";
    }
    throw new Error("Session expired");
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "Request failed" }));
    throw new Error(err.error || `Request failed with status ${res.status}`);
  }

  return res.json();
}

async function tryRefresh(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${API_BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) return false;

    const data = await res.json();
    setTokens(data.access_token, data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

/* ── Auth API ── */

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in_sec: number;
}

export async function authRegister(
  email: string,
  password: string,
): Promise<TokenPair> {
  return fetchAPI<TokenPair>("/auth/register", {
    method: "POST",
    body: { email, password },
    auth: false,
  });
}

export async function authLogin(
  email: string,
  password: string,
): Promise<TokenPair> {
  return fetchAPI<TokenPair>("/auth/login", {
    method: "POST",
    body: { email, password },
    auth: false,
  });
}

export async function forgotPassword(
  email: string,
): Promise<{ token: string; message: string }> {
  return fetchAPI<{ token: string; message: string }>("/auth/forgot-password", {
    method: "POST",
    body: { email },
    auth: false,
  });
}

export async function resetPassword(
  token: string,
  newPassword: string,
): Promise<{ message: string }> {
  return fetchAPI<{ message: string }>("/auth/reset-password", {
    method: "POST",
    body: { token, new_password: newPassword },
    auth: false,
  });
}

/* ── User Profile API ── */

export interface ProfileInput {
  full_name: string;
  age: number;
  phone_number: string;
  pan_number: string;
  monthly_income: number;
  employment_years: number;
}

export interface ProfileData extends ProfileInput {
  id?: string;
  created_at?: string;
}

export async function upsertProfile(profile: ProfileInput) {
  return fetchAPI("/user/profile", { method: "POST", body: profile });
}

export async function getProfile(): Promise<ProfileData> {
  return fetchAPI<ProfileData>("/users/profile");
}

export interface RiskScoreData {
  score: number;
  risk_category: string;
  checked_at: string;
}

export async function getRiskScore(): Promise<RiskScoreData> {
  return fetchAPI<RiskScoreData>("/users/risk-score");
}

export interface ActiveFund {
  _id: string;
  name: string;
  total_amount: number;
  monthly_contribution: number;
  duration_months: number;
  status: string;
  start_date: string;
  creator_id: string;
  current_member_count: number;
}

export interface Contribution {
  fund_id: string;
  fund_name: string;
  amount_due: number;
  due_date: string;
  cycle_number: number;
  status: string;
}

export async function getMyFunds(): Promise<ActiveFund[]> {
  return fetchAPI<ActiveFund[]>("/users/me/funds");
}

export async function getMyContributions(): Promise<Contribution[]> {
  return fetchAPI<Contribution[]>("/users/me/contributions");
}

/* ── ChitFund API ── */

export interface CreateFundInput {
  name: string;
  description: string;
  total_amount: number;
  monthly_contribution: number;
  max_members: number;
  start_date: string;
}

export interface FundDetails {
  _id: string;
  name: string;
  description: string;
  total_amount: number;
  monthly_contribution: number;
  duration_months: number;
  max_members: number;
  status: string;
  start_date: string;
  creator_id: string;
  current_member_count: number;
}

export async function createFund(input: CreateFundInput): Promise<FundDetails> {
  return fetchAPI<FundDetails>("/funds", { method: "POST", body: input });
}

export async function listFunds(): Promise<FundDetails[]> {
  return fetchAPI<FundDetails[]>("/funds");
}

export async function getFundDetails(fundId: string): Promise<FundDetails> {
  return fetchAPI<FundDetails>(`/funds/${fundId}`);
}

export async function getFundMembers(fundId: string): Promise<any[]> {
  return fetchAPI<any[]>(`/funds/${fundId}/members`);
}

export interface ApplyResult {
  message: string;
  status: string;
}

export interface FundApplicationStatus {
  status: "none" | "pending" | "active" | "rejected";
}

export async function applyToFund(fundId: string): Promise<ApplyResult> {
  return fetchAPI<ApplyResult>(`/funds/${fundId}/apply`, { method: "POST" });
}

export async function approveFundMember(
  fundId: string,
  memberId: string,
): Promise<void> {
  return fetchAPI(`/funds/${fundId}/approve`, {
    method: "POST",
    body: { user_id: memberId },
  });
}

export async function rejectFundMember(
  fundId: string,
  memberId: string,
): Promise<void> {
  return fetchAPI(`/funds/${fundId}/reject`, {
    method: "POST",
    body: { user_id: memberId },
  });
}

export async function getFundApplicationStatus(
  fundId: string,
): Promise<FundApplicationStatus> {
  return fetchAPI<FundApplicationStatus>(`/funds/${fundId}/application-status`);
}

/* ── Payments API ── */

export interface SessionDetails {
  amount: number;
  fund_id: string;
  cycle_no: number;
  due_date: string;
}

export async function getPaymentSession(
  sessionId: string,
): Promise<SessionDetails> {
  return fetchAPI<SessionDetails>(`/payments/session/${sessionId}`);
}

export interface CreateOrderResult {
  order_id: string;
  amount: number;
  key_id: string;
}

export async function createPaymentOrder(
  sessionId: string,
): Promise<CreateOrderResult> {
  return fetchAPI<CreateOrderResult>("/payments/create-order", {
    method: "POST",
    body: { session_id: sessionId },
  });
}

export async function verifyPayment(input: {
  session_id: string;
  razorpay_order_id: string;
  razorpay_payment_id: string;
  razorpay_signature: string;
}) {
  return fetchAPI("/payments/verify", { method: "POST", body: input });
}

/* ── KYC / Trust Score API ── */

export type KYCStatus =
  | "none"
  | "pending"
  | "pan_verified"
  | "credit_fetched"
  | "ml_ready" // legacy — backend normalises to 'verified'
  | "verified";

export interface KYCStatusData {
  kyc_status: KYCStatus;
  has_cibil: boolean;
  cibil_score: number | null;
  trust_score: number;
  risk_band: string;
  default_probability: number;
  checked_at?: string;
}

export interface PANVerifyResult {
  pan_verified: boolean;
  has_cibil: boolean;
  cibil_score: number | null;
  kyc_status: KYCStatus;
  skipped?: boolean;
  error?: string;
}

export interface FetchHistoryResult {
  success: boolean;
  kyc_status: KYCStatus;
  skipped?: boolean;
  history?: Record<string, unknown>;
}

export interface RunMLResult {
  score: number;
  risk_band: string;
  default_probability: number;
  kyc_status: KYCStatus;
  skipped?: boolean;
}

export async function getKYCStatus(): Promise<KYCStatusData> {
  return fetchAPI<KYCStatusData>("/users/kyc/status");
}

export async function verifyPAN(): Promise<PANVerifyResult> {
  return fetchAPI<PANVerifyResult>("/user/kyc/verify-pan", {
    method: "POST",
    body: {},
  });
}

export async function fetchKYCHistory(
  bankAccount: string,
  ifscCode: string,
): Promise<FetchHistoryResult> {
  return fetchAPI<FetchHistoryResult>("/user/kyc/fetch-history", {
    method: "POST",
    body: { bank_account: bankAccount, ifsc_code: ifscCode },
  });
}

export async function runMLPrediction(): Promise<RunMLResult> {
  return fetchAPI<RunMLResult>("/user/kyc/run-ml", {
    method: "POST",
    body: {},
  });
}

/* ── Auction API ── */

export interface AuctionSession {
  _id: string;
  fund_id: string;
  cycle_number: number;
  status: "waiting" | "live" | "ended";
  current_price: number;
  last_bid_user_id?: string;
  last_bid_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AuctionBid {
  _id: string;
  fund_id: string;
  cycle_number: number;
  user_id: string;
  increment: number;
  resulting_price: number;
  created_at: string;
}

export interface AuctionResult {
  _id: string;
  fund_id: string;
  cycle_number: number;
  winner_user_id: string;
  winning_price: number;
  payout_amount: number;
  created_at: string;
}

export interface MemberProfileInfo {
  user_id: string;
  full_name: string;
  wallet_address: string;
  is_winner: boolean;
  dividend?: number;
}

export interface AuctionSnapshot {
  session?: AuctionSession;
  result?: AuctionResult;
  bids: AuctionBid[];
  live_countdown_seconds?: number;
  members_info?: MemberProfileInfo[];
}

export async function startAuction(fundId: string): Promise<AuctionSession> {
  return fetchAPI<AuctionSession>(`/funds/${fundId}/auction/start`, {
    method: "POST",
  });
}

export async function activateAuction(
  fundId: string,
): Promise<{ message: string }> {
  return fetchAPI<{ message: string }>(`/funds/${fundId}/auction/activate`, {
    method: "POST",
  });
}

export async function placeBid(
  fundId: string,
  increment: number,
): Promise<{ bid: AuctionBid; auction: AuctionSession }> {
  return fetchAPI<{ bid: AuctionBid; auction: AuctionSession }>(
    `/funds/${fundId}/auction/bid`,
    {
      method: "POST",
      body: { increment },
    },
  );
}

export async function getAuction(fundId: string): Promise<AuctionSnapshot> {
  return fetchAPI<AuctionSnapshot>(`/funds/${fundId}/auction`);
}

/* ── Fund Contributions API ── */

export interface FundContributionEntry {
  fund_id: string;
  user_id: string;
  cycle_number: number;
  amount_due: number;
  due_date: string;
  status: string;
  blockchain_status?: string;
}

export interface CurrentCycleContributions {
  fund_id: string;
  cycle_number: number;
  contributions: FundContributionEntry[];
  total_due_amount: number;
  current_member_count: number;
}

export async function getFundContributions(
  fundId: string,
): Promise<CurrentCycleContributions> {
  return fetchAPI<CurrentCycleContributions>(
    `/funds/${fundId}/contributions/current`,
  );
}

/* ── Web3 Wallet API ── */

export interface WalletInfo {
  address: string;
  balance: number;
}

export async function getWalletInfo(): Promise<WalletInfo> {
  return fetchAPI<WalletInfo>("/web3/wallet/info");
}

export interface WalletHistoryEntry {
  tx_hash: string;
  from: string;
  to: string;
  value: number;
  type: "credit" | "debit";
  timestamp: string;
}

export async function getWalletHistory(address: string): Promise<WalletHistoryEntry[]> {
  const data = await fetchAPI<{ history: WalletHistoryEntry[] }>(`/web3/wallet/${address}/history`);
  return data.history || [];
}

/* ── Chat API ── */

export interface ChatMessage {
  _id: string;
  fund_id: string;
  user_id: string;
  full_name: string;
  message: string;
  chat_type: string;
  cycle_number?: number;
  created_at: string;
}

export async function getChatMessages(
  fundId: string,
  chatType: "fund" | "auction",
  cycleNumber?: number,
  limit = 50,
): Promise<ChatMessage[]> {
  let url = `/funds/${fundId}/chat?type=${chatType}&limit=${limit}`;
  if (cycleNumber !== undefined) url += `&cycle=${cycleNumber}`;
  return fetchAPI<ChatMessage[]>(url);
}

export async function sendChatMessage(
  fundId: string,
  chatType: "fund" | "auction",
  message: string,
  cycleNumber?: number,
): Promise<ChatMessage> {
  return fetchAPI<ChatMessage>(`/funds/${fundId}/chat`, {
    method: "POST",
    body: { chat_type: chatType, message, cycle_number: cycleNumber },
  });
}
