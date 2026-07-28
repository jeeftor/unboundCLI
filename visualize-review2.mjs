import { chromium } from 'playwright';

const BASE = 'https://caddy-sync.vookie.net';

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  await page.goto(BASE, { waitUntil: 'networkidle' });
  await page.waitForTimeout(3000);

  const targets = [
    'jellyfin.vookie.net',      // CF Access only
    'audiobook.vookie.net',     // Double-login (error)
    'auth.vookie.net',          // CF Access bypass, no forward_auth (warning)
  ];

  for (const hostname of targets) {
    console.log(`\n${'='.repeat(80)}`);
    console.log(`=== ${hostname} ===`);

    const row = page.locator(`#entries tr[data-hostname="${hostname}"]`);
    await row.locator('button.row-visualize-btn').click();

    // Wait for auth inventory to load (look for the loading spinner to disappear)
    await page.waitForTimeout(5000);

    const modal = page.locator('.visualize-modal');
    if (await modal.count() === 0) {
      console.log('Modal did not open!');
      continue;
    }

    // Verdict
    const verdictName = await modal.locator('.visualize-auth-verdict-name').textContent().catch(() => 'N/A');
    const verdictClass = await modal.locator('.visualize-auth-verdict').first().evaluate(el => Array.from(el.classList).join(' ')).catch(() => '');
    console.log(`Verdict: ${verdictName} (${verdictClass})`);

    // Flow nodes — get text content directly
    const nodes = await modal.locator('.auth-flow-node').all();
    console.log(`\nFlow nodes (${nodes.length}):`);
    for (let i = 0; i < nodes.length; i++) {
      const text = (await nodes[i].textContent())?.trim().replace(/\s+/g, ' ');
      const classes = await nodes[i].evaluate(el => Array.from(el.classList).filter(c => c !== 'auth-flow-node').join(' '));
      console.log(`  [${i}] ${text} (${classes})`);
    }

    // Arrows
    const arrows = await modal.locator('.auth-flow-arrow').all();
    console.log(`\nArrows (${arrows.length}):`);
    for (let i = 0; i < arrows.length; i++) {
      const text = (await arrows[i].textContent())?.trim().replace(/\s+/g, ' ');
      console.log(`  [${i}] "${text}"`);
    }

    // Flow tables
    const tables = await modal.locator('.flow-table').all();
    for (let t = 0; t < tables.length; t++) {
      const title = await modal.locator('.visualize-section-title').nth(t + 2).textContent().catch(() => `Table ${t}`);
      const rows = await tables[t].locator('tbody tr').all();
      console.log(`\n${title?.trim()} (${rows.length} rows):`);
      for (let r = 0; r < rows.length; r++) {
        const cells = await rows[r].locator('td').allTextContents();
        const isWarn = await rows[r].evaluate(el => el.classList.contains('flow-warn'));
        console.log(`  ${cells[0]} | ${cells[1]} | ${cells[2]} | ${cells[3]}${isWarn ? ' [WARN]' : ''}`);
      }
    }

    // Screenshot
    const safeName = hostname.replace(/[^a-z0-9]/gi, '_');
    await page.screenshot({ path: `/tmp/visualize-${safeName}.png`, fullPage: true });
    console.log(`Screenshot: /tmp/visualize-${safeName}.png`);

    await page.locator('.visualize-modal .modal-close').click();
    await page.waitForTimeout(500);
  }

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
