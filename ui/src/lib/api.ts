import { pb } from './pocketbase';
import type { Job } from './types';

/** Payload for POST /api/jobs/{id}/retry. */
export interface RetryRequest {
  /** "full" re-runs the normal pipeline; "rsync" skips straight to rsync. */
  mode: 'full' | 'rsync';
  anime_title?: string;
  anime_list_id?: string;
  tmdb_id?: number;
  tmdb_season?: number;
}

/** Restart a failed job via the custom backend route. */
export async function retryJob(id: string, req: RetryRequest): Promise<Job> {
  return pb.send(`/api/jobs/${id}/retry`, {
    method: 'POST',
    body: req,
  });
}

export interface PlaygroundResult {
  ok: boolean;
  status: number;
  data: unknown;
  durationMs: number;
}

/** Raw request used by the API playground page (attaches the PB auth token). */
export async function playgroundRequest(
  method: string,
  path: string,
  body?: string,
): Promise<PlaygroundResult> {
  const base = import.meta.env.VITE_PB_URL ?? '';
  const headers: Record<string, string> = {};
  if (pb.authStore.token) {
    headers['Authorization'] = pb.authStore.token;
  }
  if (body) {
    headers['Content-Type'] = 'application/json';
  }

  const started = performance.now();
  const res = await fetch(base + path, { method, headers, body });
  const text = await res.text();
  const durationMs = Math.round(performance.now() - started);

  let data: unknown = text;
  try {
    data = JSON.parse(text);
  } catch {
    // keep raw text when the response is not JSON
  }

  return { ok: res.ok, status: res.status, data, durationMs };
}
