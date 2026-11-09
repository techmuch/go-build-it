// Learn (FR-134–FR-139, ADR-012): every lesson is completed in the practice project and each
// step checks itself off; the doctor's harness tier runs; screenshots for the docs.
// Runs only from scripts/learn_e2e.sh (a demo whose only harness is the scripted one).
import { test, expect, type Page, type APIRequestContext } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import * as fs from 'node:fs';
import * as os from 'node:os';
import * as path from 'node:path';

const SHOTS = path.resolve(process.cwd(), '../docs/user/screenshots');
test.skip(!process.env.GBI_LEARN, 'run scripts/learn_e2e.sh');
test.describe.configure({ mode: 'serial' });
test.setTimeout(180_000);

async function shot(page: Page, name: string) {
  await page.waitForTimeout(400);
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  expect(overflow, `${name} overflows horizontally by ${overflow}px`).toBeLessThanOrEqual(1);
  await page.screenshot({ path: path.join(SHOTS, `${name}.png`) });
}

const coachStep = (page: Page, n: number) => page.locator(`[data-testid=coach] li[data-step="${n}"]`);
async function done(page: Page, n: number) {
  await expect(coachStep(page, n)).toHaveAttribute('data-done', 'true', { timeout: 45_000 });
}
async function tick(page: Page, n: number) {
  await coachStep(page, n).getByTestId('mark-done').click();
  await done(page, n);
}
async function startLesson(page: Page, slug: string) {
  await page.goto('/learn');
  await page.locator(`[data-lesson=${slug}] button`).first().click();
  await expect(page.getByTestId('coach')).toBeVisible();
  await expect(page.getByTestId('coach')).not.toContainText('Loading');
}
async function finished(page: Page) {
  await expect(page.getByTestId('lesson-finished')).toBeVisible({ timeout: 45_000 });
}
async function api(req: APIRequestContext, method: string, url: string, data?: unknown) {
  const r = await req.fetch(url, { method, data });
  expect(r.ok(), `${method} ${url}: ${r.status()} ${await r.text()}`).toBeTruthy();
  return r.json();
}
async function waitFor<T>(fn: () => Promise<T | undefined>, what: string, ms = 60_000): Promise<T> {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    const v = await fn();
    if (v) return v;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error('timed out waiting for ' + what);
}
async function addAndRun(req: APIRequestContext, title: string, description: string) {
  const t = await api(req, 'POST', '/api/tasks', { project: 'practice', title, description, role: 'developer' });
  await api(req, 'POST', `/api/tasks/${t.id}/run`, {});
  return t.id as string;
}

test.describe('Learn · desktop', () => {
  test.use({ viewport: { width: 1440, height: 900 } });

  test('the practice project is created from Learn', async ({ page }) => {
    await page.goto('/learn');
    await expect(page.getByText('0 of 11 lessons done')).toBeVisible();
    await page.getByTestId('create-practice').click();
    await expect(page.getByTestId('practice-card')).toContainText('birdwatch', { timeout: 20_000 });
    await expect(page.locator('.thinking a')).toHaveCount(7);
    await shot(page, 'learn');
    await page.goto('/projects/practice');
    await expect(page.locator('.tabs')).toContainText('practice · scripted');
  });

  test('lesson 0: the doctor starts the harnesses', async ({ page }) => {
    await startLesson(page, 'setup');
    await coachStep(page, 1).getByTestId('show-me').click();
    await expect(page).toHaveURL(/\/settings\/herdr$/);
    await done(page, 1);
    await expect(page.locator('[data-check="pane.roundtrip"]')).toHaveAttribute('data-status', 'ok', { timeout: 30_000 });
    await page.getByTestId('doctor-harness').click();
    await expect(page.locator('[data-check="harness.starts"]')).toHaveAttribute('data-status', 'ok', { timeout: 60_000 });
    await expect(page.locator('[data-tier=harness]')).toHaveClass(/ok/);
    await done(page, 2);
    await shot(page, 'doctor');
    await tick(page, 3);
    await finished(page);
  });

  test('lesson 1: find your way around', async ({ page }) => {
    await startLesson(page, 'around');
    await page.goto('/projects/practice'); await done(page, 1);
    await page.locator('#main').click({ position: { x: 5, y: 5 } });
    await page.keyboard.press('g'); await page.keyboard.press('u');
    await expect(page).toHaveURL(/\/queue$/); await done(page, 2);
    await page.goto('/docs'); await done(page, 3);
    await finished(page);
  });

  test('lesson 2: your first project', async ({ page, request }) => {
    await startLesson(page, 'project');
    await coachStep(page, 1).getByTestId('show-me').click();
    await expect(page.locator('[data-tour=edit-purpose]')).toHaveClass(/tour-pulse/);
    await page.locator('[data-tour=edit-purpose]').click();
    await page.locator('textarea').first().fill('A small bird-sighting log, quick to use in the field.');
    await page.getByRole('button', { name: 'Save' }).first().click();
    await done(page, 1);
    await api(request, 'POST', '/api/projects/practice/values', { text: 'Quick to record in the field', source: 'lesson 2' });
    await done(page, 2);
    await api(request, 'POST', '/api/projects/practice/milestones', { name: 'M2 · Species list', scope: 'Suggest species names.', gateName: 'Species review', gateKind: 'review' });
    await done(page, 3);
    await page.goto('/projects/practice/artifacts'); await done(page, 4);
    await finished(page);
  });

  test('lesson 3: your team', async ({ page, request }) => {
    await startLesson(page, 'team');
    await page.goto('/projects/practice/team'); await done(page, 1);
    await api(request, 'PATCH', '/api/members/' + encodeURIComponent('forge@practice'), { tuning: '- Keep forms usable with one hand.' });
    await done(page, 2);
    await page.goto('/archetypes'); await done(page, 3);
    const m = await api(request, 'POST', '/api/projects/practice/members', { archetype: 'qa' });
    expect(m.harness).toBe('fake'); // practice members never use a subscription
    await done(page, 4);
    await finished(page);
  });

  test('lesson 4: running work, with Show me on the queue', async ({ page, request }) => {
    await startLesson(page, 'work');
    await page.goto('/projects/practice/queue'); await done(page, 1);
    await coachStep(page, 2).getByTestId('show-me').click();
    await expect(page.locator('[data-tour=add-task]')).toHaveClass(/tour-pulse/);
    await shot(page, 'learn-coach');
    await page.locator('[data-tour=add-task]').click();
    await page.getByRole('dialog').locator('.form-row', { hasText: 'Title' }).locator('input').fill('Species list');
    await page.getByRole('dialog').getByRole('button', { name: 'Queue task' }).click();
    await done(page, 2);
    await page.locator('tr', { hasText: 'Sighting form' }).locator('[data-tour=run-task]').click();
    await done(page, 3);
    await done(page, 4);
    await tick(page, 5);
    await finished(page);
    await waitFor(async () => {
      const rs = await api(request, 'GET', '/api/reviews?project=practice');
      return rs.length ? rs : undefined;
    }, 'the review of Sighting form');
  });

  test('lesson 5: when the AI needs you', async ({ page, request }) => {
    await startLesson(page, 'needs-you');
    await addAndRun(request, 'Remember the last place', 'FAKE:ask');
    await done(page, 1);
    const q = await waitFor(async () => (await api(request, 'GET', '/api/questions?project=practice')).find((x: any) => x.status === 'open'), 'the question');
    await page.goto('/questions');
    await expect(page.getByText(q.title).first()).toBeVisible();
    await shot(page, 'harness-inbox');
    await api(request, 'POST', `/api/questions/${q.id}/answer`, { choice: 'A' });
    await done(page, 2);
    const tid = await addAndRun(request, 'Guarded export', 'FAKE:block');
    const s = await waitFor(async () => (await api(request, 'GET', '/api/sessions?project=practice&live=true')).find((x: any) => x.taskId === tid && x.status === 'blocked'), 'a blocked member');
    await page.goto(`/sessions/${s.id}?view=terminal`);
    await page.getByRole('button', { name: '⌨ Take control' }).click();
    await expect(page.locator('.live-term').getByText('you have control')).toBeVisible({ timeout: 15_000 });
    await done(page, 3);
    await page.keyboard.type('y'); await page.keyboard.press('Enter');
    await page.getByRole('button', { name: 'Hand back' }).click();
    await finished(page);
  });

  test('lesson 6: reviews, rehearsals and options', async ({ page, request }) => {
    await startLesson(page, 'review');
    await page.goto('/reviews'); await done(page, 1);
    const rs = await api(request, 'GET', '/api/reviews?project=practice');
    await api(request, 'POST', `/api/reviews/${rs[0].id}/verdict`, { verdict: 'approve', comment: 'Looks right.' });
    await done(page, 2);
    await api(request, 'POST', '/api/options', { project: 'practice', title: 'Sighting list layout', options: [{ key: 'A', title: 'Cards', summary: 'Big cards' }, { key: 'B', title: 'Compact list', summary: 'One line each' }] });
    await done(page, 3);
    const g = await api(request, 'POST', '/api/rehearsals', { project: 'practice', title: 'Record a sighting', kind: 'tabletop', steps: [{ title: 'Open the form', scenario: 'Tap New sighting' }, { title: 'Save', scenario: 'Save it' }] });
    await api(request, 'POST', `/api/rehearsals/${g.id}/findings`, { step: 0, text: 'The species field needs a bigger touch target.' });
    await done(page, 4);
    await finished(page);
  });

  test('lesson 7: the history', async ({ page }) => {
    await startLesson(page, 'history');
    await page.goto('/history'); await done(page, 1);
    await page.getByRole('link', { name: 'Open ›' }).first().click();
    await expect(page).toHaveURL(/\/history\/\d+/); await done(page, 2);
    await tick(page, 3);
    await finished(page);
  });

  test('lesson 9: drive gbi from your AI harness (the harness adds a task)', async ({ page, request }) => {
    await startLesson(page, 'harness');
    for (const n of [1, 2, 3]) await tick(page, n);
    // What a harness runs: gbi tasks add (here through the same API).
    await api(request, 'POST', '/api/tasks', { project: 'practice', title: 'Export sightings as CSV', role: 'developer', description: 'One row per sighting: species, place, time, note.' });
    await done(page, 4);
    await finished(page);
    await page.goto('/projects/practice/queue');
    await expect(page.getByText('Export sightings as CSV')).toBeVisible();
    await shot(page, 'harness-task');
  });

  test('lesson 10: a real project', async ({ page, request }) => {
    await startLesson(page, 'real');
    await page.goto('/settings'); await done(page, 1);
    await page.goto('/changelog'); await done(page, 2);
    const repo = fs.mkdtempSync(path.join(os.tmpdir(), 'gbi-real-'));
    execFileSync('git', ['init', '-q', '-b', 'main'], { cwd: repo });
    fs.writeFileSync(path.join(repo, 'README.md'), '# realapp\n');
    execFileSync('git', ['-c', 'user.name=t', '-c', 'user.email=t@example.com', 'add', '-A'], { cwd: repo });
    execFileSync('git', ['-c', 'user.name=t', '-c', 'user.email=t@example.com', 'commit', '-q', '-m', 'init'], { cwd: repo });
    await api(request, 'POST', '/api/projects', { path: repo, id: 'realapp', purpose: 'A real project' });
    await done(page, 3);
    await tick(page, 4);
    await finished(page);
  });

  test('the lesson page and progress survive a reload; Esc hides the coach and g l brings it back', async ({ page }) => {
    await page.goto('/learn/work');
    await expect(page.getByRole('heading', { name: '4 · Running work' })).toBeVisible();
    await expect(page.locator('.lesson-full li[data-done=true]')).toHaveCount(5);
    await shot(page, 'learn-lesson');
    await page.getByTestId('open-coach').click();
    await expect(page.getByTestId('coach')).toBeVisible();
    await page.reload();
    await expect(page.getByTestId('coach')).toBeVisible();
    await expect(page.locator('[data-testid=coach] li[data-done=true]')).toHaveCount(5);
    await page.getByTestId('coach').locator('button').first().focus();
    await page.keyboard.press('Escape');
    await expect(page.getByTestId('coach')).toBeHidden();
    await page.locator('#main').click({ position: { x: 5, y: 5 } });
    await page.keyboard.press('g'); await page.keyboard.press('l');
    await expect(page.getByTestId('coach')).toBeVisible();
  });
});

test.describe('Learn · phone', () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });

  test('lesson 8 on the phone: More → Learn', async ({ page }) => {
    await page.goto('/learn');
    await page.locator('[data-lesson=phone] button').first().click();
    await expect(page.getByTestId('coach')).toBeVisible();
    await page.goto('/m');
    await page.goto('/m/more');
    await page.getByTestId('m-learn').click();
    await expect(page).toHaveURL(/\/m\/learn$/);
    await page.locator('.m-li[data-lesson=phone]').click();
    await expect(page.getByTestId('m-lesson')).toContainText('✅ 8 · gbi on your phone', { timeout: 30_000 });
    await shot(page, 'm-learn');
  });

  test('every lesson is done', async ({ page }) => {
    await page.goto('/m/learn');
    await expect(page.getByText('11 of 11 done')).toBeVisible({ timeout: 20_000 });
  });
});
