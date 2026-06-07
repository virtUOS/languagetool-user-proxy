import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  use: { baseURL: 'http://127.0.0.1:8080' },
  workers: 1,
  retries: 0,
  projects: [
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
  ],
});
