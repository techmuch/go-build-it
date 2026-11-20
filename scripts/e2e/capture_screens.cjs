// Captures every screen of the running demo into docs/mockups/screens/ — the as-built baseline
// shown on the Mockups tab. It only opens pages (opening herdr setup runs the doctor).
// Name screens to capture just those: node ../scripts/e2e/capture_screens.cjs d10 d19
//
//   GBI_HOME=$(mktemp -d) PORT=7431 scripts/demo.sh        # seeded demo, fake harness
//   cd web && node ../scripts/e2e/capture_screens.cjs      # writes docs/mockups/screens/*.png
const path = require('node:path');
const fs = require('node:fs');
const { chromium } = require(path.resolve(process.cwd(), 'node_modules/playwright'));

const URL = process.env.GBI_E2E_URL || 'http://127.0.0.1:7431';
const OUT = path.resolve(__dirname, '../../docs/mockups/screens');
const ONLY = process.argv.slice(2);
const CAP = 2000; // tallest capture, in CSS pixels

// [file name, address, optional step run before the capture]
const desktop = [
  ['d01-catalog', '/'],
  ['d02-project', '/projects/ledgerly'],
  ['d03-milestone', '/milestones/ledgerly-M0'],
  ['d04-artifacts-requirements', '/projects/trailmap/artifacts'],
  ['d05-artifacts-mockups', '/projects/trailmap/artifacts/mockups/docs/mockups/phone.html#v02'],
  ['d06-artifacts-trace', '/projects/trailmap/artifacts/requirements?trace=1'],
  ['d07-artifacts-compare', '/projects/trailmap/artifacts/compare/v02'],
  ['d08-artifacts-files', '/projects/trailmap/artifacts/files'],
  ['d09-artifacts-tracker', '/projects/trailmap/artifacts/tracker'],
  ['d10-artifact-editor', '/projects/trailmap/edit?path=docs%2Frequirements.md'],
  ['d11-project-team', '/projects/ledgerly/team'],
  ['d12-project-member', '/projects/ledgerly/team/' + encodeURIComponent('forge@ledgerly')],
  ['d13-project-queue', '/projects/ledgerly/queue'],
  ['d14-project-sessions', '/projects/ledgerly/sessions'],
  ['d15-project-releases', '/projects/ledgerly/releases'],
  ['d16-project-history', '/projects/ledgerly/history'],
  ['d17-project-settings', '/projects/ledgerly/settings'],
  ['d18-questions', '/questions'],
  ['d19-question', '/questions/Q-2'],
  ['d20-options', '/options/OS-1'],
  ['d21-rehearsal', '/rehearsals/G-1'],
  ['d22-review', '/reviews/R-1'],
  ['d23-sessions', '/sessions'],
  ['d24-session', '/sessions/S-3'],
  ['d25-session-terminal', '/sessions/S-3?view=terminal', (p) => p.waitForTimeout(2500)],
  ['d26-queue', '/queue'],
  ['d27-archetypes', '/archetypes/developer'],
  ['d28-standards', '/standards/git-workflow'],
  ['d29-history', '/history'],
  ['d30-event', '/history', async (p) => { await p.getByRole('link', { name: 'Open ›' }).first().click(); }],
  ['d31-settings', '/settings'],
  ['d32-settings-herdr', '/settings/herdr'],
  ['d33-changelog', '/changelog'],
  ['d34-learn', '/learn'],
  ['d35-lesson', '/learn/around'],
  ['d36-docs', '/docs'],
  ['d37-palette', '/', async (p) => { await p.locator('.pcard').first().waitFor(); await p.keyboard.press('Meta+k'); await p.getByPlaceholder('Search projects, tasks, docs, commands…').fill('review'); }],
];
const mobile = [
  ['m01-home', '/m/'],
  ['m02-project', '/m/projects/ledgerly'],
  ['m03-inbox', '/m/inbox'],
  ['m04-answer', '/m/q/Q-1'],
  ['m05-review', '/m/review/R-1'],
  ['m06-rehearsal', '/m/rehearsal/G-1'],
  ['m07-options', '/m/options/OS-1'],
  ['m08-sessions', '/m/sessions'],
  ['m09-terminal', '/m/sessions/S-3', (p) => p.waitForTimeout(2000)],
  ['m10-documents', '/m/projects/trailmap/docs'],
  ['m11-document', '/m/projects/trailmap/docs/requirements'],
  ['m12-compare', '/m/projects/trailmap/compare/v02'],
  ['m13-more', '/m/more'],
  ['m14-changelog', '/m/changelog'],
  ['m15-learn', '/m/learn'],
];

async function run(ctx, list, width, height) {
  const page = await ctx.newPage();
  const report = [];
  for (const [name, addr, step] of list) {
    if (ONLY.length && !ONLY.some((o) => name.startsWith(o))) continue;
    await page.goto(URL + addr);
    await page.waitForTimeout(1200);
    if (step) await step(page);
    await page.waitForTimeout(500);
    // Grow the window to the page's height (up to CAP) so fixed parts, like the sidebar, fill it.
    const full = await page.evaluate(() => document.documentElement.scrollHeight);
    const h = Math.min(Math.max(full, height), CAP);
    await page.setViewportSize({ width, height: h });
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.waitForTimeout(300);
    await page.screenshot({ path: path.join(OUT, name + '.png') });
    await page.setViewportSize({ width, height });
    report.push({ name, addr, height: h, cut: full > CAP, title: await page.title(), at: page.url().replace(URL, '') });
    console.log(name, h, full > CAP ? '(cut)' : '');
  }
  await page.close();
  return report;
}

(async () => {
  fs.mkdirSync(OUT, { recursive: true });
  const b = await chromium.launch({ channel: 'chrome' });
  const d = await b.newContext({ viewport: { width: 1400, height: 900 } });
  const m = await b.newContext({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 });
  const report = [...await run(d, desktop, 1400, 900), ...await run(m, mobile, 390, 844)];
  if (!ONLY.length) fs.writeFileSync(path.join(OUT, 'screens.json'), JSON.stringify(report, null, 1) + '\n');
  await b.close();
})();
