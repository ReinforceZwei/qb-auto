import { useEffect, useState } from 'react';
import { pb } from './pocketbase';
import type { Job } from './types';

export interface RealtimeJobs {
  /** All jobs currently seen, keyed by id. */
  byId: Record<string, Job>;
  /** Jobs ordered by creation time (newest last). */
  list: Job[];
}

/**
 * Subscribes to realtime changes on the `jobs` collection and returns the
 * current set of jobs, upserted on every create/update and removed on delete.
 *
 * On mount it also loads the currently active (non-terminal) jobs, because
 * realtime events only cover changes that happen after subscribing.
 */
export function useJobsRealtime(): RealtimeJobs {
  const [byId, setById] = useState<Record<string, Job>>({});

  useEffect(() => {
    let active = true;
    let unsubscribe: (() => Promise<void>) | undefined;

    // Initial load of jobs already in the pipeline.
    pb.collection('jobs')
      .getList<Job>(1, 200, {
        filter: "status != 'done' && status != 'error'",
        sort: 'created',
      })
      .then((result) => {
        if (!active) return;
        setById(Object.fromEntries(result.items.map((j) => [j.id, j])));
      })
      .catch((err) => {
        console.error('failed to load active jobs', err);
      });

    pb.collection('jobs')
      .subscribe('*', (event) => {
        const record = event.record as Job;
        setById((prev) => {
          const next = { ...prev };
          if (event.action === 'delete') {
            delete next[record.id];
            return next;
          }
          next[record.id] = record;
          return next;
        });
      })
      .then((fn) => {
        if (!active) {
          void fn();
        } else {
          unsubscribe = fn;
        }
      })
      .catch((err) => {
        console.error('jobs realtime subscription failed', err);
      });

    return () => {
      active = false;
      unsubscribe?.();
    };
  }, []);

  return {
    byId,
    list: Object.values(byId).sort((a, b) =>
      a.created.localeCompare(b.created),
    ),
  };
}
