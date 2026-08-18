import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  redirect,
} from '@tanstack/react-router';
import { AppLayout } from './components/layout/AppLayout';
import { isAuthenticated } from './lib/pocketbase';
import { HistoryPage } from './routes/history';
import { MonitorPage } from './routes/index';
import { LoginPage } from './routes/login';
import { NotFoundPage } from './routes/not-found';
import { PlaygroundPage } from './routes/playground';

const rootRoute = createRootRoute({
  component: () => <Outlet />,
});

// Authenticated area: AppShell + guarded pages.
const appLayoutRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'app',
  beforeLoad: () => {
    if (!isAuthenticated()) {
      throw redirect({ to: '/login' });
    }
  },
  component: AppLayout,
});

const monitorRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/',
  component: MonitorPage,
});

const historyRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/history',
  component: HistoryPage,
});

const playgroundRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/playground',
  component: PlaygroundPage,
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  beforeLoad: () => {
    if (isAuthenticated()) {
      throw redirect({ to: '/' });
    }
  },
  component: LoginPage,
});

const notFoundRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '$',
  component: NotFoundPage,
});

const routeTree = rootRoute.addChildren([
  appLayoutRoute.addChildren([monitorRoute, historyRoute, playgroundRoute]),
  loginRoute,
  notFoundRoute,
]);

export const router = createRouter({ routeTree });
