import PocketBase from 'pocketbase';

/**
 * Single PocketBase client for the whole app.
 *
 * - Local development: `VITE_PB_URL` points at the Go server (e.g. http://127.0.0.1:8091).
 * - Production (embedded build): unset/empty → '/' (root-absolute same-origin).
 *   NOTE: never fall back to a plain empty string — the SDK resolves relative
 *   base URLs against `window.location.pathname`, so on /login the API calls
 *   would wrongly go to /login/api/... instead of /api/....
 */
export const pb = new PocketBase(import.meta.env.VITE_PB_URL || '/');

export function isAuthenticated(): boolean {
  return pb.authStore.isValid;
}

export async function signIn(email: string, password: string): Promise<void> {
  await pb.collection('_superusers').authWithPassword(email, password);
}

export function signOut(): void {
  pb.authStore.clear();
}
