import PocketBase from 'pocketbase';

/**
 * Single PocketBase client for the whole app.
 *
 * - Local development: `VITE_PB_URL` points at the Go server (e.g. http://127.0.0.1:8090).
 * - Production (embedded build): empty string → same origin as the server.
 */
export const pb = new PocketBase(import.meta.env.VITE_PB_URL ?? '');

export function isAuthenticated(): boolean {
  return pb.authStore.isValid;
}

export async function signIn(email: string, password: string): Promise<void> {
  await pb.collection('_superusers').authWithPassword(email, password);
}

export function signOut(): void {
  pb.authStore.clear();
}
