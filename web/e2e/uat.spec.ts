// UAT walkthrough for the Owner role. Each step checks the view works and captures
// desktop + phone screenshots for the docs (FR-131, FR-133) and for review (FR-112).
import { test, expect, type Page } from '@playwright/test';
import * as path from 'node:path';

const SHOTS = path.resolve(process.cwd(), '../docs/user/screenshots');
async function shot(page: Page, name: string) {
  await page.waitForTimeout(500);
  // Layout guard: no view may scroll sideways, on desktop or phone.
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  expect(overflow, `${name} overflows horizontally by ${overflow}px`).toBeLessThanOrEqual(1);
  await page.screenshot({ path: path.join(SHOTS, `${name}.png`), fullPage: false });
}

test.describe('desktop · owner', () => {
  test.use({ viewport: { width: 1400, height: 900 } });

  test('01 catalog shows projects that need me first', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('.pcard').first()).toBeVisible();
    await expect(page.getByText('ledgerly').first()).toBeVisible();
    await expect(page.getByText('No stable release yet').first()).toBeVisible();
    await shot(page, 'catalog');
  });

  test('02 project overview and roadmap item', async ({ page }) => {
    await page.goto('/projects/ledgerly');
    await expect(page.getByText('M1 · CSV import').first()).toBeVisible();
    await shot(page, 'project-overview');
    await page.getByText('M1 · CSV import').first().click();
    await expect(page.getByText('Gate passes when').or(page.getByText('Dress rehearsal')).first()).toBeVisible();
    await shot(page, 'roadmap-item');
  });

  test('03 artifacts, values and the editor', async ({ page }) => {
    await page.goto('/projects/trailmap/artifacts');
    await expect(page.getByText('FR-08').first()).toBeVisible();
    await expect(page.getByText('Glanceable in sunlight')).toBeVisible();
    await shot(page, 'artifacts');
    await page.getByRole('link', { name: '✎ Edit' }).click();
    await expect(page.getByText('Edit with')).toBeVisible();
    await page.getByLabel('Message').fill('Add FR-11 for sunlight readability');
    await page.getByRole('button', { name: 'Send' }).click();
    await expect(page.getByText('Proposed changes')).toBeVisible({ timeout: 20000 });
    await shot(page, 'artifact-editor');
  });

  test('04 questions dashboard and viewer', async ({ page }) => {
    await page.goto('/questions');
    await page.getByText('Download maps per trail or per region?').first().click();
    await expect(page.getByText('Why it matters')).toBeVisible();
    await shot(page, 'questions');
    await page.getByText('map-layout-a.html ›').click();
    await expect(page.getByRole('dialog', { name: 'Viewer' })).toBeVisible();
    await shot(page, 'viewer');
    await page.keyboard.press('Escape');
  });

  test('05 sessions and terminal view', async ({ page }) => {
    await page.goto('/sessions');
    await page.getByRole('button', { name: 'Recent' }).click();
    await expect(page.locator('table.t tr.click').first()).toBeVisible();
    await shot(page, 'sessions');
    await page.locator('table.t tr.click').first().click();
    await page.getByRole('button', { name: 'Terminal' }).click();
    await expect(page.locator('.term')).toBeVisible();
    await shot(page, 'session-terminal');
  });

  test('06 task queue shows gate-blocked work', async ({ page }) => {
    await page.goto('/queue');
    await expect(page.getByText('Categorize transactions')).toBeVisible();
    await expect(page.getByText(/waits for gate/).first()).toBeVisible();
    await shot(page, 'queue');
  });

  test('07 option set with large preview and tray', async ({ page }) => {
    await page.goto('/options/OS-1');
    await expect(page.getByText('Combination tray')).toBeVisible();
    await page.getByLabel('Bottom sheet').check();
    await page.getByRole('button', { name: /B · Split/ }).click();
    await page.getByLabel('Elevation sparkline').check();
    await shot(page, 'option-set');
  });

  test('08 dress rehearsal with findings from several reviewers', async ({ page }) => {
    await page.goto('/rehearsals/G-1');
    await expect(page.getByText('Findings & team chat')).toBeVisible();
    await page.getByText('2. Map CSV columns').click();
    await expect(page.getByText('debit vs credit')).toBeVisible();
    await shot(page, 'rehearsal');
  });

  test('09 work review is an assessment', async ({ page }) => {
    await page.goto('/reviews/R-1');
    await expect(page.getByText('Approved mockup vs built')).toBeVisible();
    await expect(page.getByText('Requirements in scope')).toBeVisible();
    await shot(page, 'review');
  });

  test('10 team and persona coach', async ({ page }) => {
    await page.goto('/team/forge');
    await expect(page.getByText('Persona coach')).toBeVisible();
    await shot(page, 'team');
  });

  test('11 standards with agent chat', async ({ page }) => {
    await page.goto('/standards/git-workflow');
    await expect(page.getByText('Edit with')).toBeVisible();
    await shot(page, 'standards');
  });

  test('12 history and event detail', async ({ page }) => {
    await page.goto('/history');
    await expect(page.getByText('Event log')).toBeVisible();
    await shot(page, 'history');
    await page.getByRole('link', { name: 'Open ›' }).first().click();
    await expect(page.getByText('What happened as a result')).toBeVisible();
    await shot(page, 'event-detail');
  });

  test('13 settings and 14 docs', async ({ page }) => {
    await page.goto('/settings');
    await expect(page.locator('#harnesses')).toBeVisible();
    await shot(page, 'settings');
    await page.goto('/docs');
    await expect(page.getByText('Contents')).toBeVisible();
    await shot(page, 'docs');
  });

  test('⌘K palette shows CLI equivalents', async ({ page }) => {
    await page.goto('/');
    await page.keyboard.press('Meta+k');
    await page.getByPlaceholder('Search projects, tasks, docs, commands…').fill('review');
    await expect(page.locator('.palette .hit').first()).toBeVisible();
    await expect(page.locator('.palette code').first()).toContainText('gbi');
    await shot(page, 'palette');
  });
});

test.describe('mobile · owner', () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });

  test('M1 home, M3 inbox, M4 answer', async ({ page }) => {
    await page.goto('/m/');
    await expect(page.locator('.m-card').first()).toBeVisible();
    await shot(page, 'm-home');
    await page.getByText('Inbox').click();
    await expect(page.getByText('Blocking · work paused')).toBeVisible();
    await shot(page, 'm-inbox');
    await page.getByText('UK bank CSV dates — DD/MM?').click();
    await expect(page.getByText('Assumed:')).toBeVisible();
    await shot(page, 'm-answer');
  });

  test('M5 review, M6 rehearsal, M7 options, M8 sessions, M9 more', async ({ page }) => {
    await page.goto('/m/review/R-1'); await expect(page.getByText('Standards')).toBeVisible(); await shot(page, 'm-review');
    await page.goto('/m/rehearsal/G-1'); await expect(page.getByText('Findings')).toBeVisible(); await shot(page, 'm-rehearsal');
    await page.goto('/m/options/OS-1'); await expect(page.getByText('Keep these aspects')).toBeVisible(); await shot(page, 'm-options');
    await page.goto('/m/sessions'); await expect(page.getByRole('heading', { name: 'Sessions' })).toBeVisible(); await shot(page, 'm-sessions');
    await page.goto('/m/more'); await expect(page.getByText('Docs')).toBeVisible(); await shot(page, 'm-more');
  });

  test('M10 terminal with key bar', async ({ page }) => {
    await page.goto('/m/sessions/S-1');
    await expect(page.locator('.m-keys')).toBeVisible();
    await shot(page, 'm-terminal');
  });
});

test.describe('owner actions', () => {
  test.use({ viewport: { width: 1400, height: 900 } });
  test('answer a blocking question and the task resumes', async ({ page, request }) => {
    await page.goto('/questions/Q-1');
    await page.getByRole('button', { name: 'Answer & resume work' }).click();
    await expect(page.getByText(/Answered/)).toBeVisible();
    const t = await (await request.get('/api/tasks/T-2')).json();
    expect(['queued', 'running', 'review']).toContain(t.task.status);
  });
});

test.describe('reviewer · maria over the tailnet', () => {
  // Requests via Tailscale Serve arrive on loopback with identity headers (FR-17).
  test.use({ viewport: { width: 1400, height: 900 }, extraHTTPHeaders: { 'Tailscale-User-Login': 'maria@github', 'Tailscale-User-Name': 'maria' } });

  test('identity badge, attributed findings, no gate approval', async ({ page, request }) => {
    await page.goto('/');
    await expect(page.getByText('Tailnet · maria@github')).toBeVisible();
    await page.goto('/rehearsals/G-1');
    await page.getByPlaceholder('Feedback for this step… (type or dictate)').fill('"Payee" vs "Description" is confusing for my bank.');
    await page.getByRole('button', { name: 'Add finding' }).click();
    await expect(page.getByText('reviewers:').filter({ hasText: 'maria' })).toBeVisible();
    await expect(page.getByText('Only an owner can approve (ADR-006)').or(page.getByText(/not reviewed yet/))).toBeVisible();
    await shot(page, 'reviewer-rehearsal');
    const res = await request.post('/api/rehearsals/G-1/decide', { data: { decision: 'approve' } });
    expect(res.status()).toBe(400);
    expect(await res.text()).toMatch(/owner|not been reviewed/);
  });
});

test.describe('project manager · persona via the CLI API', () => {
  test('a persona can queue tasks and ask questions, but not answer them', async ({ request }) => {
    const h = { 'X-GBI-Persona': 'pilot' };
    const t = await (await request.post('/api/tasks', { headers: h, data: { project: 'trailmap', title: 'Split offline sync into two tasks', role: 'architect' } })).json();
    expect(t.id).toMatch(/^T-/);
    const q = await (await request.post('/api/questions', { headers: h, data: { project: 'trailmap', title: 'Ship regions later?', assumption: 'yes, after v1' } })).json();
    expect(q.asker).toBe('pilot');
    const ans = await request.post(`/api/questions/${q.id}/answer`, { headers: h, data: { answer: 'sure' } });
    expect(ans.status()).toBe(403);
  });
});
