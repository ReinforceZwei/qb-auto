import { Button, Center, Stack, Text, Title } from '@mantine/core';
import { Link } from '@tanstack/react-router';

export function NotFoundPage() {
  return (
    <Center h="100vh">
      <Stack align="center" gap="xs">
        <Title order={1}>404</Title>
        <Text c="dimmed">Page not found</Text>
        <Button component={Link} to="/" variant="subtle" mt="sm">
          Back to monitor
        </Button>
      </Stack>
    </Center>
  );
}
