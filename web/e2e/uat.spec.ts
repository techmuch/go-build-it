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
    // The requirements entry point renders requirements.md inside the sandbox (FR-144, FR-145).
    await expect(page.frameLocator('.docframe iframe').getByText('FR-08').first()).toBeVisible();
    await expect(page.getByText('Glanceable in sunlight')).toBeVisible();
    await shot(page, 'artifacts');
    await page.getByRole('link', { name: '✎ Edit source' }).click();
    await expect(page.getByText('Edit with')).toBeVisible();
    await page.getByLabel('Message').fill('Add FR-11 for sunlight readability');
    await page.getByRole('button', { name: 'Send' }).click();
    await expect(page.getByText('Proposed changes')).toBeVisible({ timeout: 20000 });
    await shot(page, 'artifact-editor');
  });

  test('03b project documents: mockups, compare, traceability, support files, tracker', async ({ page }) => {
    // Mockups: the entry links to its sets; the viewer follows the link and keeps the address (FR-141, FR-144).
    await page.goto('/projects/trailmap/artifacts/mockups');
    const doc = page.frameLocator('.docframe iframe');
    // Links are clicked inside the page (a mouse click into the sandboxed frame is unreliable in Playwright).
    await doc.locator('a[href="phone.html"]').evaluate((a: HTMLElement) => a.click());
    await expect(page).toHaveURL(/artifacts\/mockups\/docs\/mockups\/phone\.html/);
    await page.goto('/projects/trailmap/artifacts/mockups/docs/mockups/phone.html#v02');
    await expect(page.getByText('On this view')).toBeVisible();
    await expect(page.locator('.card', { hasText: 'On this view' }).getByText('v02 · Trail map')).toBeVisible();
    await shot(page, 'documents-mockups');
    // A requirement link in a mockup switches to the Requirements tab at that anchor.
    await page.frameLocator('.docframe iframe').locator('a[href="../requirements.html#FR-08"]').first().evaluate((a: HTMLElement) => a.click());
    await expect(page).toHaveURL(/artifacts\/requirements#FR-08$/);

    // Traceability (FR-146).
    await page.goto('/projects/trailmap/artifacts/requirements?trace=1');
    await expect(page.getByText('v04 names FR-22, which is not a requirement')).toBeVisible();
    await shot(page, 'documents-trace');

    // Compare (FR-147): side by side, overlay, then decisions.
    await page.goto('/projects/trailmap/artifacts/compare/v02');
    await expect(page.locator('.compare img')).toBeVisible();
    await shot(page, 'documents-compare');
    await page.getByRole('button', { name: 'Overlay' }).click();
    await page.getByLabel('Overlay position').fill('60');
    await page.evaluate(() => window.scrollTo(0, 0));
    await shot(page, 'documents-compare-overlay');
    await page.getByPlaceholder('What is different, or why it is fine').fill('The offline badge is missing and the download button lost its size (FR-07, FR-08).');
    await page.getByRole('button', { name: 'File a task' }).click();
    await expect(page.getByText('Task filed', { exact: true })).toBeVisible();
    await expect(page.locator('.docbar').getByText('task filed')).toBeVisible();
    await page.goto('/projects/trailmap/artifacts/compare/v03');
    await expect(page.locator('.docbar').getByText('drifted')).toBeVisible();
    await page.getByRole('button', { name: '✓ Accept the real screen' }).click();
    await expect(page.getByText('Accepted v03', { exact: true })).toBeVisible();
    await expect(page.locator('.docbar').getByText('accepted')).toBeVisible();

    // Support files (FR-142): the unlabelled file gets a label; nothing is guessed.
    await page.goto('/projects/trailmap/artifacts/files');
    await expect(page.getByText('1 unlabelled').first()).toBeVisible();
    await shot(page, 'documents-files');
    await page.getByRole('button', { name: 'Unlabelled 1' }).click();
    await page.getByRole('row', { name: /interviews\.md/ }).getByRole('button', { name: 'Label…' }).click();
    await page.getByRole('row', { name: /interviews\.md/ }).getByLabel('research').check();
    await page.getByRole('row', { name: /interviews\.md/ }).getByRole('button', { name: 'Save' }).click();
    await expect(page.getByText('Every file has a label.')).toBeVisible();

    // Tracker (FR-148): the entry reads the snapshot; Write snapshot refreshes it.
    await page.goto('/projects/trailmap/artifacts/tracker');
    await page.getByRole('button', { name: 'Write snapshot' }).click();
    await expect(page.frameLocator('.docframe iframe').getByText('Match v02 · Trail map to its mockup').first()).toBeVisible();
    await shot(page, 'documents-tracker');
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
    // The member's herdr pane, streamed live (FR-66): watch, take control, hand back.
    await expect(page.locator('.live-term .xterm')).toBeVisible();
    await expect(page.locator('.live-term').getByText('watching · read-only')).toBeVisible({ timeout: 15000 });
    await expect(page.locator('.xterm-rows')).toContainText(/\S/, { timeout: 15000 });
    await page.getByRole('button', { name: '⌨ Take control' }).click();
    await expect(page.locator('.live-term').getByText('you have control')).toBeVisible({ timeout: 15000 });
    await shot(page, 'session-terminal');
    await page.getByRole('button', { name: 'Hand back' }).click();
    await expect(page.locator('.live-term').getByText('watching · read-only')).toBeVisible({ timeout: 15000 });
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

  test('10 persona archetypes with change review and coach', async ({ page }) => {
    await page.goto('/archetypes/developer');
    await expect(page.getByText('Persona coach')).toBeVisible();
    await expect(page.getByText(/Change review/)).toBeVisible();
    await shot(page, 'archetypes');
  });

  test('10b project team: tuning, diff and promote', async ({ page }) => {
    await page.goto('/projects/ledgerly/team/' + encodeURIComponent('forge@ledgerly'));
    await expect(page.getByRole('button', { name: 'Save tuning' })).toBeVisible();
    await expect(page.getByText('Promote to the archetype')).toBeVisible();
    await page.getByRole('button', { name: 'Diff vs archetype' }).click();
    await expect(page.getByText('fixture for every new bank').first()).toBeVisible();
    await shot(page, 'team');
  });

  test('10c a member waiting for you: answer in the live terminal', async ({ page, request }) => {
    await page.goto('/questions');
    await expect(page.getByText('Members waiting for you')).toBeVisible();
    await page.getByText(/is waiting for you \(T-/).first().click();
    await expect(page).toHaveURL(/view=terminal/);
    await expect(page.locator('.xterm-rows')).toContainText('Allow edit', { timeout: 20000 });
    await page.getByRole('button', { name: '⌨ Take control' }).click();
    await expect(page.locator('.live-term').getByText('you have control')).toBeVisible({ timeout: 15000 });
    await page.locator('.xterm-helper-textarea').focus();
    await page.keyboard.type('y');
    await page.keyboard.press('Enter');
    await expect.poll(async () => (await (await request.get('/api/sessions?live=true')).json()).filter((s: any) => s.status === 'blocked').length, { timeout: 30000 }).toBe(0);
    await shot(page, 'member-waiting');
    await page.getByRole('button', { name: 'Hand back' }).click();
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
    await expect(page.locator('.pcard').first()).toBeVisible(); // shortcuts are live once the page has mounted
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

  test('M14 project documents and compare', async ({ page }) => {
    await page.goto('/m/projects/trailmap/docs');
    await expect(page.getByText('Requirements', { exact: true })).toBeVisible();
    await shot(page, 'm-documents');
    await page.goto('/m/projects/trailmap/compare/v02');
    await page.getByRole('button', { name: 'Real app' }).click();
    await expect(page.locator('img[alt="Screenshot of v02"]')).toBeVisible();
    await shot(page, 'm-compare');
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
