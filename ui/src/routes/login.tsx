import { useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Center,
  PasswordInput,
  Stack,
  Text,
  TextInput,
  Title,
} from '@mantine/core';
import { useNavigate, useRouter } from '@tanstack/react-router';
import { IconAlertCircle } from '@tabler/icons-react';
import { signIn } from '../lib/pocketbase';

export function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const router = useRouter();

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await signIn(email, password);
      await router.invalidate();
      await navigate({ to: '/' });
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to sign in. Please try again.',
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Center h="100vh" bg="var(--mantine-color-body)">
      <Card withBorder shadow="md" radius="md" w={380} padding="xl">
        <Stack gap="md">
          <div>
            <Title order={2}>qb-auto</Title>
            <Text c="dimmed" size="sm">
              Sign in with your PocketBase superuser account.
            </Text>
          </div>

          {error && (
            <Alert icon={<IconAlertCircle size={16} />} color="red" variant="light">
              {error}
            </Alert>
          )}

          <form onSubmit={handleSubmit}>
            <Stack gap="sm">
              <TextInput
                label="Email"
                type="email"
                required
                autoComplete="username"
                value={email}
                onChange={(e) => setEmail(e.currentTarget.value)}
              />
              <PasswordInput
                label="Password"
                required
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.currentTarget.value)}
              />
              <Button type="submit" loading={submitting} fullWidth mt="sm">
                Sign in
              </Button>
            </Stack>
          </form>
        </Stack>
      </Card>
    </Center>
  );
}
