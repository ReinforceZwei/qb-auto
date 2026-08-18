import { useMemo, useState } from 'react';
import {
  Card,
  Checkbox,
  Group,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import { useJobsRealtime } from '../lib/realtime';
import { ACTIVE_STATUSES, type Job } from '../lib/types';
import { JobTable } from '../components/JobTable';
import { RetryJobModal } from '../components/RetryJobModal';

function isToday(value: string | undefined): boolean {
  if (!value) return false;
  const d = new Date(value);
  const now = new Date();
  return (
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate()
  );
}

interface StatCardProps {
  label: string;
  value: number;
  color?: string;
}

function StatCard({ label, value, color }: StatCardProps) {
  return (
    <Card withBorder radius="md" padding="lg">
      <Text size="sm" c="dimmed">
        {label}
      </Text>
      <Text fz={34} fw={700} c={color}>
        {value}
      </Text>
    </Card>
  );
}

export function MonitorPage() {
  const { list } = useJobsRealtime();
  const [showTerminal, setShowTerminal] = useState(false);
  const [retryJob, setRetryJob] = useState<Job | null>(null);

  const stats = useMemo(() => {
    const active = list.filter((j) => ACTIVE_STATUSES.includes(j.status));
    const processing = list.filter((j) => j.status.startsWith('processing_'));
    const failed = list.filter((j) => j.status === 'error');
    const doneToday = list.filter((j) => j.status === 'done' && isToday(j.completed));
    return { active, processing, failed, doneToday };
  }, [list]);

  const visibleJobs = showTerminal ? list : stats.active;

  return (
    <Stack gap="lg">
      <div>
        <Title order={2}>Monitor</Title>
        <Text c="dimmed" size="sm">
          Live view of active jobs (updates in realtime).
        </Text>
      </div>

      <SimpleGrid cols={{ base: 2, sm: 4 }}>
        <StatCard label="Active jobs" value={stats.active.length} />
        <StatCard label="Processing" value={stats.processing.length} color="indigo" />
        <StatCard label="Failed" value={stats.failed.length} color="red" />
        <StatCard label="Done today" value={stats.doneToday.length} color="green" />
      </SimpleGrid>

      <Card withBorder radius="md" padding="lg">
        <Stack gap="sm">
          <Group justify="space-between">
            <Text fw={600}>Jobs</Text>
            <Checkbox
              label="Show done & failed"
              checked={showTerminal}
              onChange={(e) => setShowTerminal(e.currentTarget.checked)}
            />
          </Group>
          <JobTable
            jobs={visibleJobs}
            onRetry={(job) => setRetryJob(job)}
          />
        </Stack>
      </Card>

      <RetryJobModal
        job={retryJob}
        opened={retryJob !== null}
        onClose={() => setRetryJob(null)}
        onRetried={() => {
          // Realtime subscription already reflects the new status; nothing else to do.
        }}
      />
    </Stack>
  );
}
