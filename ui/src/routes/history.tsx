import { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Card,
  Code,
  Drawer,
  Group,
  MultiSelect,
  Pagination,
  Select,
  Stack,
  Text,
  TextInput,
  Title,
} from '@mantine/core';
import { IconRotate } from '@tabler/icons-react';
import { pb } from '../lib/pocketbase';
import {
  ALL_STATUSES,
  STATUS_META,
  type Job,
} from '../lib/types';
import { JobTable } from '../components/JobTable';
import { RetryJobModal } from '../components/RetryJobModal';

const PER_PAGE = 20;

function escapeFilterValue(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
}

function buildFilter(
  statuses: string[],
  category: string | null,
  query: string,
): string {
  const parts: string[] = [];
  if (statuses.length > 0) {
    parts.push(
      `(${statuses.map((s) => `status = '${escapeFilterValue(s)}'`).join(' || ')})`,
    );
  }
  if (category === 'anime') {
    parts.push(`category = 'anime'`);
  } else if (category === 'other') {
    parts.push(`category != 'anime'`);
  }
  const q = query.trim();
  if (q) {
    const escaped = escapeFilterValue(q);
    parts.push(
      `(anime_title ~ '${escaped}' || torrent_hash ~ '${escaped}')`,
    );
  }
  return parts.join(' && ');
}

interface JobDetailProps {
  job: Job;
  onRetry: (job: Job) => void;
}

function JobDetail({ job, onRetry }: JobDetailProps) {
  const rows: Array<[string, string]> = [
    ['ID', job.id],
    ['Status', STATUS_META[job.status]?.label ?? job.status],
    ['Torrent hash', job.torrent_hash],
    ['Category', job.category || '—'],
    ['Anime title', job.anime_title || '—'],
    ['AnimeList ID', job.anime_list_id || '—'],
    ['TMDb ID', job.tmdb_id ? String(job.tmdb_id) : '—'],
    ['Season', job.tmdb_season ? String(job.tmdb_season) : '—'],
    ['Error', job.error || '—'],
    ['Created', job.created || '—'],
    ['Updated', job.updated || '—'],
    ['Completed', job.completed || '—'],
  ];

  return (
    <Stack gap="sm">
      {job.status === 'error' && (
        <Group justify="space-between" align="center">
          <Text c="red" size="sm" style={{ wordBreak: 'break-all' }}>
            {job.error}
          </Text>
          <Button
            size="xs"
            variant="light"
            color="indigo"
            leftSection={<IconRotate size={14} />}
            onClick={() => onRetry(job)}
          >
            Restart job
          </Button>
        </Group>
      )}
      {rows.map(([label, value]) => (
        <Group key={label} justify="space-between" gap="xl" wrap="nowrap">
          <Text size="sm" c="dimmed" w={110}>
            {label}
          </Text>
          <Text
            size="sm"
            ff={label === 'Torrent hash' || label === 'ID' ? 'monospace' : undefined}
            style={{ wordBreak: 'break-all', textAlign: 'right' }}
          >
            {value}
          </Text>
        </Group>
      ))}
      <Code block mt="xs">
        {JSON.stringify(job, null, 2)}
      </Code>
    </Stack>
  );
}

export function HistoryPage() {
  const [statuses, setStatuses] = useState<string[]>([]);
  const [category, setCategory] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailJob, setDetailJob] = useState<Job | null>(null);
  const [retryJob, setRetryJob] = useState<Job | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const filter = buildFilter(statuses, category, query);
      const result = await pb.collection('jobs').getList<Job>(page, PER_PAGE, {
        filter,
        sort: '-created',
      });
      setJobs(result.items);
      setTotalPages(result.totalPages);
      setTotalItems(result.totalItems);
    } catch (err) {
      console.error('failed to load job history', err);
      setJobs([]);
      setTotalPages(1);
      setTotalItems(0);
    } finally {
      setLoading(false);
    }
  }, [statuses, category, query, page]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleFiltersChange = (resetPage: boolean) => {
    if (resetPage) setPage(1);
  };

  return (
    <Stack gap="lg">
      <div>
        <Title order={2}>Job history</Title>
        <Text c="dimmed" size="sm">
          Browse all jobs ({totalItems} total).
        </Text>
      </div>

      <Card withBorder radius="md" padding="lg">
        <Stack gap="sm">
          <Group align="flex-end" grow>
            <MultiSelect
              label="Status"
              placeholder="All statuses"
              data={ALL_STATUSES.map((s) => ({
                value: s,
                label: STATUS_META[s].label,
              }))}
              value={statuses}
              onChange={(v) => {
                setStatuses(v);
                handleFiltersChange(true);
              }}
              clearable
            />
            <Select
              label="Category"
              placeholder="All"
              data={[
                { value: 'anime', label: 'Anime' },
                { value: 'other', label: 'Other' },
              ]}
              value={category}
              onChange={(v) => {
                setCategory(v);
                handleFiltersChange(true);
              }}
              clearable
            />
            <TextInput
              label="Search"
              placeholder="Anime title or torrent hash"
              value={query}
              onChange={(e) => {
                setQuery(e.currentTarget.value);
                handleFiltersChange(true);
              }}
            />
          </Group>

          <JobTable
            jobs={jobs}
            loading={loading}
            onRowClick={(job) => setDetailJob(job)}
            onRetry={(job) => setRetryJob(job)}
          />

          <Group justify="space-between">
            <Text size="sm" c="dimmed">
              {totalItems} jobs
            </Text>
            <Pagination
              total={totalPages}
              value={page}
              onChange={setPage}
              withEdges
            />
          </Group>
        </Stack>
      </Card>

      <Drawer
        opened={detailJob !== null}
        onClose={() => setDetailJob(null)}
        title={detailJob ? 'Job details' : ''}
        position="right"
        size="md"
      >
        {detailJob && (
          <JobDetail
            job={detailJob}
            onRetry={(job) => {
              setDetailJob(null);
              setRetryJob(job);
            }}
          />
        )}
      </Drawer>

      <RetryJobModal
        job={retryJob}
        opened={retryJob !== null}
        onClose={() => setRetryJob(null)}
        onRetried={() => {
          void load();
        }}
      />
    </Stack>
  );
}
