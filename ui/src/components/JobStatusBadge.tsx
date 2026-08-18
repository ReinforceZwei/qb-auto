import { Badge } from '@mantine/core';
import { STATUS_META, type JobStatus } from '../lib/types';

export function JobStatusBadge({ status }: { status: JobStatus }) {
  const meta = STATUS_META[status] ?? { label: status, color: 'gray' };
  return (
    <Badge color={meta.color} variant="light" size="sm">
      {meta.label}
    </Badge>
  );
}
