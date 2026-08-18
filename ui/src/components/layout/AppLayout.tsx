import { AppShell, Burger, Button, Group, Title } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Link, Outlet, useNavigate, useRouter } from '@tanstack/react-router';
import {
  IconApi,
  IconHistory,
  IconLayoutDashboard,
  IconLogout,
} from '@tabler/icons-react';
import classes from './AppLayout.module.css';
import { signOut } from '../../lib/pocketbase';

const NAV_LINKS = [
  { to: '/', label: 'Monitor', icon: IconLayoutDashboard },
  { to: '/history', label: 'Job history', icon: IconHistory },
  { to: '/playground', label: 'API playground', icon: IconApi },
] as const;

export function AppLayout() {
  const [opened, { toggle }] = useDisclosure();
  const navigate = useNavigate();
  const router = useRouter();

  const handleSignOut = () => {
    signOut();
    void router.invalidate();
    void navigate({ to: '/login' });
  };

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{
        width: 240,
        breakpoint: 'sm',
        collapsed: { mobile: !opened },
      }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group gap="sm">
            <Burger
              opened={opened}
              onClick={toggle}
              hiddenFrom="sm"
              size="sm"
            />
            <Title order={3}>qb-auto</Title>
          </Group>
          <Button
            variant="subtle"
            color="gray"
            leftSection={<IconLogout size={16} />}
            onClick={handleSignOut}
          >
            Sign out
          </Button>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="sm">
        {NAV_LINKS.map((link) => (
          <Link
            key={link.to}
            to={link.to}
            className={classes.link}
            activeProps={{ className: classes.linkActive }}
          >
            <link.icon size={18} />
            <span>{link.label}</span>
          </Link>
        ))}
      </AppShell.Navbar>

      <AppShell.Main>
        <Outlet />
      </AppShell.Main>
    </AppShell>
  );
}
