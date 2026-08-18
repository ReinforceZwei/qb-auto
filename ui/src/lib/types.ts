/** Job lifecycle statuses, mirroring models/job.go. */
export type JobStatus =
  | 'pending'
  | 'processing_title'
  | 'pending_rsync'
  | 'processing_rsync'
  | 'pending_notify'
  | 'processing_notify'
  | 'done'
  | 'error';

/** A `jobs` record as returned by the PocketBase API. */
export interface Job {
  id: string;
  status: JobStatus;
  torrent_hash: string;
  category: string;
  anime_title: string;
  anime_list_id: string;
  tmdb_id: number;
  tmdb_season: number;
  error: string;
  completed: string;
  created: string;
  updated: string;
  collectionId: string;
  collectionName: string;
}

/** Presentational metadata for each status (drives badges, filters, legends). */
export const STATUS_META: Record<JobStatus, { label: string; color: string }> = {
  pending: { label: 'Pending', color: 'blue' },
  processing_title: { label: 'Processing title', color: 'indigo' },
  pending_rsync: { label: 'Pending rsync', color: 'cyan' },
  processing_rsync: { label: 'Processing rsync', color: 'teal' },
  pending_notify: { label: 'Pending notify', color: 'violet' },
  processing_notify: { label: 'Processing notify', color: 'grape' },
  done: { label: 'Done', color: 'green' },
  error: { label: 'Error', color: 'red' },
};

export const ALL_STATUSES = Object.keys(STATUS_META) as JobStatus[];

/** Statuses shown on the monitor (jobs still in the pipeline). */
export const ACTIVE_STATUSES: JobStatus[] = ALL_STATUSES.filter(
  (s) => s !== 'done' && s !== 'error',
);

/** Human-readable title for a job (falls back to the torrent hash). */
export function jobDisplayTitle(job: Job): string {
  return job.anime_title || job.torrent_hash;
}
