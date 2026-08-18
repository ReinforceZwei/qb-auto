import { useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Group,
  Modal,
  NumberInput,
  SegmentedControl,
  Stack,
  Text,
  TextInput,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconAlertCircle, IconRotate } from '@tabler/icons-react';
import { retryJob } from '../lib/api';
import type { Job } from '../lib/types';

interface RetryJobModalProps {
  /** The job to restart; when null the modal is closed. */
  job: Job | null;
  opened: boolean;
  onClose: () => void;
  /** Called after a successful retry so callers can refresh. */
  onRetried: () => void;
}

type RetryMode = 'full' | 'rsync';

/** Manual restart of a failed job, with optional "skip to rsync" mode. */
export function RetryJobModal({
  job,
  opened,
  onClose,
  onRetried,
}: RetryJobModalProps) {
  const [mode, setMode] = useState<RetryMode>('full');
  const [animeTitle, setAnimeTitle] = useState('');
  const [animeListId, setAnimeListId] = useState('');
  const [tmdbId, setTmdbId] = useState<number | ''>('');
  const [tmdbSeason, setTmdbSeason] = useState<number | ''>('');
  const [submitting, setSubmitting] = useState(false);

  // Reset the form every time the modal opens with a (new) job.
  useEffect(() => {
    if (job) {
      setMode('full');
      setAnimeTitle(job.anime_title || '');
      setAnimeListId(job.anime_list_id || '');
      setTmdbId(job.tmdb_id || '');
      setTmdbSeason(job.tmdb_season || '');
    }
  }, [job, opened]);

  if (!job) return null;

  const isAnime = job.category === 'anime';
  const canSkipToRsync = isAnime;

  const handleSubmit = async () => {
    if (!job) return;
    if (mode === 'rsync' && !animeTitle.trim()) {
      notifications.show({
        color: 'red',
        title: 'Missing anime title',
        message: 'An anime title is required to skip directly to rsync.',
      });
      return;
    }

    setSubmitting(true);
    try {
      await retryJob(job.id, {
        mode,
        anime_title: animeTitle.trim(),
        anime_list_id: animeListId.trim() || undefined,
        tmdb_id: tmdbId === '' ? undefined : tmdbId,
        tmdb_season: tmdbSeason === '' ? undefined : tmdbSeason,
      });
      notifications.show({
        color: 'green',
        title: 'Job restarted',
        message:
          mode === 'rsync'
            ? 'Job queued for rsync directly.'
            : 'Job re-queued into the normal pipeline.',
      });
      onRetried();
      onClose();
    } catch (err) {
      notifications.show({
        color: 'red',
        title: 'Restart failed',
        message: err instanceof Error ? err.message : 'Unknown error',
      });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title="Restart failed job"
      size="lg"
    >
      <Stack gap="md">
        <Alert
          icon={<IconAlertCircle size={16} />}
          color="red"
          variant="light"
        >
          <Text size="sm" fw={500} style={{ wordBreak: 'break-all' }}>
            {job.torrent_hash}
          </Text>
          <Text size="xs" c="dimmed">
            {job.error}
          </Text>
        </Alert>

        <SegmentedControl
          value={mode}
          onChange={(v) => setMode(v as RetryMode)}
          data={[
            { label: 'Full retry', value: 'full' },
            {
              label: 'Skip to rsync',
              value: 'rsync',
              disabled: !canSkipToRsync,
            },
          ]}
          fullWidth
        />

        {mode === 'full' && (
          <Text size="sm" c="dimmed">
            Re-runs the job through the normal pipeline. For anime jobs with an
            already-resolved title, this continues from the rsync stage.
          </Text>
        )}

        {mode === 'rsync' && (
          <Stack gap="sm">
            <Text size="sm" c="dimmed">
              Manually provide the resolved title and skip straight to the
              rsync transfer stage.
            </Text>
            <TextInput
              label="Anime title"
              placeholder="e.g. My Hero Academia S3"
              required
              value={animeTitle}
              onChange={(e) => setAnimeTitle(e.currentTarget.value)}
            />
            <TextInput
              label="AnimeList ID (optional)"
              description="Leave empty to skip marking the anime as downloaded."
              value={animeListId}
              onChange={(e) => setAnimeListId(e.currentTarget.value)}
            />
            <Group grow>
              <NumberInput
                label="TMDb ID"
                value={tmdbId}
                onChange={(value) =>
                  setTmdbId(typeof value === 'number' ? value : '')
                }
                allowDecimal={false}
              />
              <NumberInput
                label="Season number"
                value={tmdbSeason}
                onChange={(value) =>
                  setTmdbSeason(typeof value === 'number' ? value : '')
                }
                allowDecimal={false}
              />
            </Group>
          </Stack>
        )}

        <Group justify="flex-end">
          <Button variant="default" onClick={onClose} disabled={submitting}>
            Cancel
          </Button>
          <Button
            leftSection={<IconRotate size={16} />}
            onClick={handleSubmit}
            loading={submitting}
          >
            Restart job
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
