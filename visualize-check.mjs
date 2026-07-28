import { chromium } from 'playwright';

const BASE = 'https://caddy-sync.vookie.net';

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  await page.goto(BASE, { waitUntil: 'networkidle' });
  await page.waitForTimeout(3000);

  // Test auth.vookie.net (bypass-only, no forward_auth)
  const hostname = 'auth.vookie.net';
  console.log(`=== ${hostname} ===`);

  const row = page.locator(`#entries tr[data-hostname="${hostname}"]`);
  await row.locator('button.row-visualize-btn').click();
  await page.waitForTimeout(3000);

  const modal = page.locator('.visualize-modal');

  // Get node text correctly (span without class)
  const nodes = await modal.locator('.auth-flow-node').all();
  console.log(`\nFlow nodes (${nodes.length}):`);
  for (let i = 0; i < nodes.length; i++) {
    const text = await nodes[i].textContent();
    const classes = await nodes[i].evaluate(el => Array.from(el.classList).join(' '));
    console.log(`  [${i}] text="${text?.trim()}" class="${classes}"`);
  }

  // Get arrow labels correctly
  const arrows = await modal.locator('.auth-flow-arrow').all();
  console.log(`\nArrows (${arrows.length}):`);
  for (let i = 0; i < arrows.length; i++) {
    const text = await arrows[i].textContent();
    console.log(`  [${i}] text="${text?.trim()}"`);
  }

  // Verdict
  const verdictName = await modal.locator('.visualize-auth-verdict-name').textContent();
  const verdictClass = await modal.locator('.visualize-auth-verdict').first().evaluate(el => Array.from(el.classList).join(' '));
  console.log(`\nVerdict: ${verdictName} (${verdictClass})`);

  // Check if the modal scrolls / has overflow issues
  const modalBox = await modal.boundingBox();
  const viewportHeight = 900;
  console.log(`\nModal height: ${modalBox?.height}px (viewport: ${viewportHeight}px)`);

  // Check if the modal body scrolls
  const bodyScroll = await modal.locator('.modal-body').evaluate(el => ({
    scrollHeight: el.scrollHeight,
    clientHeight: el.clientHeight,
    overflow: getComputedStyle(el).overflowY,
  }));
  console.log(`Modal body: scrollHeight=${bodyScroll.scrollHeight} clientHeight=${bodyScroll.clientHeight} overflow=${bodyScroll.overflow}`);

  // Screenshot
  await page.screenshot({ path: '/tmp/visualize-auth-review.png', fullPage: true });
  console.log('Screenshot: /tmp/visualize-auth-review.png');

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
