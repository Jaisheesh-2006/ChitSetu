import { createClient } from "@supabase/supabase-js";

let cachedClient: ReturnType<typeof createClient> | null = null;

export function getSupabaseClient() {
  if (cachedClient) {
    return cachedClient;
  }

  const supabaseURL = process.env.NEXT_PUBLIC_SUPABASE_URL;
  const supabaseAnonKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY;
  if (!supabaseURL || !supabaseAnonKey) {
    throw new Error(
      "NEXT_PUBLIC_SUPABASE_URL and NEXT_PUBLIC_SUPABASE_ANON_KEY are required",
    );
  }

  cachedClient = createClient(supabaseURL, supabaseAnonKey);
  return cachedClient;
}
