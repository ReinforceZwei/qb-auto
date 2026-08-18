import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { RouterProvider } from '@tanstack/react-router';
import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';
import './index.css';
import { router } from './router';
import { theme } from './theme';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="auto">
      <Notifications />
      <RouterProvider router={router} />
    </MantineProvider>
  </StrictMode>,
);
