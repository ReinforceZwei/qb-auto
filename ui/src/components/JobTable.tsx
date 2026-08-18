import {
  ActionIcon,
  Group,
  Table,
  Text,
  Tooltip,
} from '@mantine/core';
import { IconRefresh, IconSearch } from '@tabler/icons-react';
import type { Job } from '../lib/types';
import { JobStatusBadge } from './JobStatusBadge';

interface JobTableProps {
  jobs: Job[];
  onRowClick?: (job: Job) => void;
  onRetry?: (job: Job) => void;
  loading?: boolean;
}

function formatDate(value: string | undefined): string {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString();
}

/** Reusable jobs table used by both the monitor and history pages. */
export function JobTable({ jobs, onRowClick, onRetry, loading }: JobTableProps) {
  return (
    <Table highlightOnHover striped verticalSpacing="sm">
      <Table.Thead>
        <Table.Tr>
          <Table.Th>Status</Table.Th>
          <Table.Th>Title</Table.Th>
          <Table.Th>Category</Table.Th>
          <Table.Th>Torrent hash</Table.Th>
          <Table.Th>Created</Table.Th>
          <Table.Th>Error</Table.Th>
          <Table.Th w={80}>Actions</Table.Th>
        </Table.Tr>
      </Table.Thead>
      <Table.Tbody>
        {jobs.length === 0 && !loading && (
          <Table.Tr>
            <Table.Td colSpan={7}>
              <Text c="dimmed" ta="center" py="lg">
                No jobs
              </Text>
            </Table.Td>
          </Table.Tr>
        )}
        {jobs.map((job) => (
          <Table.Tr
            key={job.id}
            style={onRowClick ? { cursor: 'pointer' } : undefined}
            onClick={onRowClick ? () => onRowClick(job) : undefined}
          >
            <Table.Td>
              <JobStatusBadge status={job.status} />
            </Table.Td>
            <Table.Td>
              <Text size="sm" fw={500} lineClamp={1} maw={320}>
                {job.anime_title || job.torrent_hash}
              </Text>
            </Table.Td>
            <Table.Td>
              <Text size="sm" c="dimmed">
                {job.category || '—'}
              </Text>
            </Table.Td>
            <Table.Td>
              <Tooltip label={job.torrent_hash}>
                <Text
                  size="xs"
                  c="dimmed"
                  ff="monospace"
                  style={{ maxWidth: 140 }}
                  truncate
                >
                  {job.torrent_hash}
                </Text>
              </Tooltip>
            </Table.Td>
            <Table.Td>
              <Text size="xs" c="dimmed">
                {formatDate(job.created)}
              </Text>
            </Table.Td>
            <Table.Td>
              {job.error ? (
                <Tooltip label={job.error} multiline maw={400}>
                  <Text size="xs" c="red" lineClamp={1} maw={240}>
                    {job.error}
                  </Text>
                </Tooltip>
              ) : (
                <Text size="xs" c="dimmed">
                  —
                </Text>
              )}
            </Table.Td>
            <Table.Td>
              <Group gap={4} wrap="nowrap" onClick={(e) => e.stopPropagation()}>
                {onRetry && job.status === 'error' && (
                  <Tooltip label="Restart job">
                    <ActionIcon
                      variant="subtle"
                      color="indigo"
                      size="sm"
                      onClick={() => onRetry(job)}
                    >
                      <IconRefresh size={16} />
                    </ActionIcon>
                  </Tooltip>
                )}
                {onRowClick && (
                  <Tooltip label="View details">
                    <ActionIcon variant="subtle" color="gray" size="sm">
                      <IconSearch size={16} />
                    </ActionIcon>
                  </Tooltip>
                )}
              </Group>
            </Table.Td>
          </Table.Tr>
        ))}
      </Table.Tbody>
    </Table>
  );
}
