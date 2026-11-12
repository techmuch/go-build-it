// Keyboard navigation (docs/user/18-keyboard.md). Runs against the seeded demo like uat.spec.ts.
import { test, expect, type Page } from '@playwright/test';

const focused = (page: Page) => page.evaluate(() => {
  const el = document.activeElement as HTMLElement | null;
  return { tag: el?.tagName || '', text: (el?.innerText || el?.getAttribute('aria-label') || '').trim().slice(0, 80), cls: el?.className || '' };
});

test.describe('keyboard · desktop', () => {
  test.use({ viewport: { width: 1400, height: 900 } });

  test('skip link, card grid arrows and Enter', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.pcard').first()).toBeVisible();
    await page.keyboard.press('Tab');
    expect((await focused(page)).text).toBe('Skip to content');
    await page.keyboard.press('Enter');
    expect(await page.evaluate(() => document.activeElement?.id)).toBe('main');
    await page.locator('.pcard').first().focus();
    const first = (await focused(page)).text;
    await page.keyboard.press('ArrowRight');
    const second = (await focused(page)).text;
    expect(second).not.toBe(first);
    expect((await focused(page)).cls).toContain('pcard');
    await page.keyboard.press('ArrowLeft');
    expect((await focused(page)).text).toBe(first);
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/\/projects\//);
    // The logo and title lead back to the landing page, by mouse and by keyboard.
    await page.locator('.side a.brand').click();
    await expect(page).toHaveURL(/127\.0\.0\.1:\d+\/$/);
    await expect(page.locator('.pcard').first()).toBeVisible();
    await page.goto('/settings');
    await page.locator('.side a.brand').focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('.pcard').first()).toBeVisible();
  });

  test('g-shortcuts, list arrows, Enter opens, ? help restores focus', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.pcard').first()).toBeVisible();
    await page.keyboard.press('g');
    await page.keyboard.press('q');
    await expect(page).toHaveURL(/\/questions$/);
    const items = page.locator('.list-item').filter({ hasNotText: 'is waiting for you' }); // question rows
    await expect(items.first()).toBeVisible();
    await items.first().focus();
    await page.keyboard.press('ArrowDown');
    await expect(items.nth(1)).toBeFocused();
    await page.keyboard.press('k');
    await expect(items.first()).toBeFocused();
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/\/questions\/.+/);
    await expect(page.getByText('Why it matters')).toBeVisible();

    const search = page.getByRole('button', { name: /Search, run a command/ });
    await search.focus();
    await page.keyboard.press('?');
    await expect(page.getByRole('dialog', { name: 'Keyboard shortcuts' })).toBeVisible();
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    expect(await page.evaluate(() => !!document.activeElement?.closest('.keys-help'))).toBe(true); // focus stays inside
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog', { name: 'Keyboard shortcuts' })).toHaveCount(0);
    await expect(search).toBeFocused();
  });

  test('single-key shortcuts are ignored while typing', async ({ page }) => {
    await page.goto('/questions');
    await page.locator('.list-item').filter({ hasNotText: 'is waiting for you' }).first().click();
    const box = page.locator('.dict textarea').first();
    await box.click();
    await page.keyboard.type('g q ? /');
    await expect(box).toHaveValue('g q ? /');
    await expect(page.getByRole('dialog')).toHaveCount(0);
  });

  test('viewer: thumbnails, ↑/↓ in the strip, ←/→, saved pins, Esc returns focus', async ({ page }) => {
    await page.goto('/questions');
    await page.getByText('Download maps per trail or per region?').first().click();
    const opener = page.getByText('map-layout-a.html ›');
    await opener.click();
    const dlg = page.getByRole('dialog', { name: /Viewer/ });
    await expect(dlg).toBeVisible();
    const strip = page.getByRole('listbox', { name: /Attachments/ });
    const opts = strip.getByRole('option');
    const n = await opts.count();
    test.skip(n < 2, 'needs an item with at least two attachments');
    // Real thumbnails, not just labels.
    await expect(strip.locator('.thumb').first()).toBeVisible();
    expect(await strip.locator('.thumb iframe, .thumb img, .thumb pre').count()).toBe(n);
    // With several attachments the strip takes focus first; ↓/↑ switch the shown attachment.
    await expect(opts.first()).toBeFocused();
    await expect(dlg.getByText(`1 of ${n}`)).toBeVisible();
    await page.keyboard.press('ArrowDown');
    await expect(opts.nth(1)).toBeFocused();
    await expect(opts.nth(1)).toHaveAttribute('aria-selected', 'true');
    await expect(dlg.getByText(`2 of ${n}`)).toBeVisible();
    await page.keyboard.press('ArrowUp');
    await expect(dlg.getByText(`1 of ${n}`)).toBeVisible();
    await page.keyboard.press('ArrowRight');
    await expect(dlg.getByText(`2 of ${n}`)).toBeVisible();
    await expect(opts.nth(1)).toBeFocused();
    await page.keyboard.press('ArrowLeft');
    // ←/→ do not switch while typing a comment.
    const comment = dlg.getByPlaceholder('Comment for the next pin…');
    await comment.fill('Keyboard pin');
    await page.keyboard.press('ArrowLeft');
    await expect(dlg.getByText(`1 of ${n}`)).toBeVisible();
    // Place a pin with the keyboard (Enter on the pin layer pins the centre) and check it is saved.
    await dlg.getByRole('button', { name: '📍 Pin' }).click();
    await dlg.getByRole('button', { name: /Place a pin/ }).focus();
    await page.keyboard.press('Enter');
    await expect(dlg.getByText('Keyboard pin')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(dlg).toHaveCount(0);
    await expect(opener).toBeFocused();
    await opener.click();
    await expect(page.getByRole('dialog', { name: /Viewer/ }).getByText('Keyboard pin')).toBeVisible();
    // Clean up so the seeded demo stays as it was.
    await page.getByRole('button', { name: 'Remove pin 1' }).click();
    await expect(page.getByRole('dialog', { name: /Viewer/ }).getByText('Keyboard pin')).toHaveCount(0);
    await page.keyboard.press('Escape');
  });

  test('tabs and segmented controls move with ←/→', async ({ page }) => {
    await page.goto('/projects/trailmap');
    const tab = page.locator('nav.tabs a').first();
    await tab.focus();
    await page.keyboard.press('ArrowRight');
    await expect(page.locator('nav.tabs a').nth(1)).toBeFocused();
    await page.keyboard.press('End');
    await expect(page.locator('nav.tabs a').last()).toBeFocused();
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/\/settings$/);
  });

  test('queue rows open with Enter and report expanded state', async ({ page }) => {
    await page.goto('/queue');
    const row = page.locator('tr[data-nav]').first();
    await row.focus();
    await page.keyboard.press('Enter');
    await expect(row).toHaveAttribute('aria-expanded', 'true');
    await page.keyboard.press(' ');
    await expect(row).toHaveAttribute('aria-expanded', 'false');
  });

  test('every link-styled action is a real button reachable with Tab', async ({ page }) => {
    for (const url of ['/', '/projects/trailmap', '/projects/trailmap/artifacts', '/questions', '/queue', '/reviews', '/team', '/settings']) {
      await page.goto(url);
      await page.waitForTimeout(400);
      const bad = await page.evaluate(() => Array.from(document.querySelectorAll('a:not([href])')).map((a) => (a as HTMLElement).innerText.slice(0, 40)));
      expect(bad, `${url} has links without href`).toEqual([]);
    }
  });
});
