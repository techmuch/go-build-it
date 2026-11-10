// Change log and guided update through a real restart (FR-06a–FR-06e, ADR-011).
// Runs only from scripts/update_e2e.sh, which starts an old build against a fake release
// server serving a new, signed build. Skipped in the normal UAT run.
import { test, expect, type Page } from '@playwright/test';
import path from 'path';

const FROM = process.env.GBI_UPDATE_FROM || '';
const MID = process.env.GBI_UPDATE_MID || '';
const TO = process.env.GBI_UPDATE_TO || '';
const LIVE = Number(process.env.GBI_UPDATE_LIVE || 0); // members working during the update
const SHOTS = path.resolve(process.cwd(), '../docs/user/screenshots');

test.skip(!TO, 'run scripts/update_e2e.sh');
test.describe.configure({ mode: 'serial' });

const stepStatuses = (page: Page) => page.$$eval('[data-testid=update-job] [data-step]', (els) => els.map((e) => `${e.getAttribute('data-step')}:${e.getAttribute('data-status')}`));

test.describe('change log & update · desktop', () => {
  test.use({ viewport: { width: 1400, height: 900 } });

  test('the version opens the change log; g v too', async ({ page }) => {
    await page.goto('/');
    const v = page.getByTestId('version-link');
    await expect(v).toHaveText('v' + FROM);
    await v.click();
    await expect(page).toHaveURL(/\/changelog$/);
    await expect(page.getByRole('heading', { name: 'Change log' })).toBeVisible();
    await expect(page.getByTestId('installed-version')).toHaveText(FROM);
    await expect(page.getByTestId('latest-version')).toHaveText(TO);
    await expect(page.getByText('2 releases newer')).toBeVisible();
    // Every newer release's notes are listed, newest first, above the embedded change log.
    const articles = page.locator('.clog article');
    await expect(articles.nth(0)).toHaveAttribute('aria-label', 'Release ' + TO);
    await expect(articles.nth(1)).toHaveAttribute('aria-label', 'Release ' + MID);
    await expect(articles.nth(0)).toContainText('Everything new in ' + TO);
    // The embedded change log follows: a released entry's notes are there (stable text, unlike Unreleased).
    await expect(page.locator('.clog article[aria-label="Release 2026.11.09.1"]')).toContainText('Doctor tiers');

    // The check is remembered: the sidebar now says an update is ready.
    await page.goto('/');
    await expect(page.locator('.side .foot')).toContainText('update ready: ' + TO);
    await page.keyboard.press('g');
    await page.keyboard.press('v');
    await expect(page).toHaveURL(/\/changelog$/);
    // Tab reaches Check again and Update now.
    const update = page.getByTestId('update-now');
    await update.focus();
    await expect(update).toBeFocused();
    await expect(update).toHaveText('⬆ Update now to ' + TO);
    await page.screenshot({ path: path.join(SHOTS, 'changelog.png') });
  });

  test('Update now shows each step completing, through the restart', async ({ page }) => {
    await page.goto('/changelog');
    if (LIVE) await expect(page.getByText(/working\. They keep running in herdr while go-build-it restarts/)).toBeVisible();
    await page.getByTestId('update-now').click();
    const job = page.getByTestId('update-job');
    await expect(job).toBeVisible();
    await expect(job).toContainText('Updating to ' + TO);

    // Record what the user sees until the update is done (the page reloads into the new UI).
    const seen = new Set<string>();
    let shot = false;
    const deadline = Date.now() + 90_000;
    while (Date.now() < deadline) {
      try {
        for (const s of await stepStatuses(page)) seen.add(s);
        if (!shot && seen.has('download:running')) {
          await page.screenshot({ path: path.join(SHOTS, 'changelog-updating.png') });
          shot = true;
        }
        if ((await page.getByTestId('installed-version').textContent()) === TO) break;
      } catch { /* navigating: the page reloads once the new version is up */ }
      await page.waitForTimeout(100);
    }
    await expect(page.getByTestId('installed-version')).toHaveText(TO);
    await expect(page.getByTestId('update-job')).toHaveAttribute('data-status', 'done');
    await expect(page.getByTestId('update-job')).toContainText('Updated to ' + TO);
    const final = await stepStatuses(page);
    expect(final).toEqual(['find', 'download', 'signature', 'checksum', 'test', 'install', 'restart', 'reconnect'].map((s) => s + ':done'));
    // The steps were watched running, not only finished.
    expect([...seen].filter((s) => s.endsWith(':running')).length).toBeGreaterThanOrEqual(3);
    expect(seen.has('restart:running')).toBe(true);
    await expect(page.locator('[data-step=reconnect]')).toContainText('herdr session gbi');
    if (LIVE) await expect(page.locator('[data-step=reconnect]')).toContainText(`${LIVE} ${LIVE === 1 ? 'member' : 'members'} still working`);
    await expect(page.getByTestId('version-link')).toHaveText('v' + TO);
    await expect(page.getByText('Previous: ' + FROM)).toBeVisible();
    await page.screenshot({ path: path.join(SHOTS, 'changelog-updated.png') });
  });
});

test.describe('change log · phone', () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });

  test('the version on More opens the change log', async ({ page }) => {
    await page.goto('/m/more');
    await page.getByTestId('version-link').click();
    await expect(page).toHaveURL(/\/m\/changelog$/);
    await expect(page.getByTestId('installed-version')).toHaveText(TO);
    await expect(page.getByTestId('update-job')).toContainText('Updated to ' + TO);
    await expect(page.locator('.clog article').first()).toBeVisible();
    await page.screenshot({ path: path.join(SHOTS, 'm-changelog.png') });
  });
});
