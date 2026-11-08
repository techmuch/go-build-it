import { defineConfig } from '@playwright/test';

// UAT walkthrough (FR-132, FR-133). Runs against a seeded demo instance:
//   GBI_HOME=$(mktemp -d) PORT=7431 ../scripts/demo.sh && GBI_E2E_URL=http://127.0.0.1:7431 npx playwright test
export default defineConfig({
  testDir: './e2e',
  timeout: 60_000,
  fullyParallel: false,
  workers: 1,
  reporter: [['list'], ['json', { outputFile: 'test-results/results.json' }]],
  use: {
    baseURL: process.env.GBI_E2E_URL || 'http://127.0.0.1:7431',
    channel: 'chrome', // use the installed Chrome; no browser download
    headless: true,
  },
});
