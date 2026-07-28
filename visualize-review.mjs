import { chromium } from 'playwright';

const BASE = 'https://caddy-sync.vookie.net';

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  await page.goto(BASE, { waitUntil: 'networkidle' });
  await page.waitForTimeout(3000);

  // Test specific hosts with different patterns
  const targets = [
    'jellyfin.vookie.net',      // CF Access only
    'audiobook.vookie.net',     // Double-login (error)
    'auth.vookie.net',          // CF Access with bypass
  ];

  for (const hostname of targets) {
    console.log(`\n${'='.repeat(80)}`);
    console.log(`=== ${hostname} ===`);
    console.log('='.repeat(80));

    const row = page.locator(`#entries tr[data-hostname="${hostname}"]`);
    const flowBtn = row.locator('button.row-visualize-btn');
    if (await flowBtn.count() === 0) {
      console.log(`No Flow button, skipping`);
      continue;
    }

    await flowBtn.click();
    await page.waitForTimeout(3000);

    const modal = page.locator('.visualize-modal');
    if (await modal.count() === 0) {
      console.log('Modal did not open!');
      continue;
    }

    // Get the full text content, section by section
    const sections = await modal.locator('.visualize-section').all();
    console.log(`\nSections: ${sections.length}`);

    // Verdict
    const verdictName = await modal.locator('.visualize-auth-verdict-name').textContent().catch(() => 'N/A');
    const verdictClass = await modal.locator('.visualize-auth-verdict').first().evaluate(el => Array.from(el.classList).join(' ')).catch(() => '');
    const verdictSummary = await modal.locator('.visualize-auth-verdict-summary').textContent().catch(() => 'N/A');
    const verdictDetail = await modal.locator('.visualize-auth-verdict-detail').textContent().catch(() => 'N/A');
    console.log(`\n--- VERDICT ---`);
    console.log(`  Name: ${verdictName}`);
    console.log(`  Class: ${verdictClass}`);
    console.log(`  Summary: ${verdictSummary}`);
    console.log(`  Detail: ${verdictDetail}`);

    // Flow diagram nodes
    const nodes = await modal.locator('.auth-flow-node').all();
    console.log(`\n--- FLOW DIAGRAM (${nodes.length} nodes) ---`);
    for (let i = 0; i < nodes.length; i++) {
      const label = await nodes[i].locator('.auth-flow-node-label').textContent().catch(() => '');
      const sub = await nodes[i].locator('.auth-flow-node-sub').textContent().catch(() => '');
      const classes = await nodes[i].evaluate(el => Array.from(el.classList).join(' '));
      console.log(`  [${i}] ${label} | sub=${sub} | class=${classes}`);
    }

    // Arrows
    const arrows = await modal.locator('.auth-flow-arrow').all();
    console.log(`\n--- ARROWS (${arrows.length}) ---`);
    for (let i = 0; i < arrows.length; i++) {
      const label = await arrows[i].locator('.auth-flow-arrow-label').textContent().catch(() => '');
      const classes = await arrows[i].evaluate(el => Array.from(el.classList).join(' '));
      console.log(`  [${i}] label=${label} | class=${classes}`);
    }

    // Flow tables
    const tables = await modal.locator('.flow-table').all();
    console.log(`\n--- FLOW TABLES (${tables.length}) ---`);
    for (let t = 0; t < tables.length; t++) {
      const title = await modal.locator('.visualize-section-title').nth(t + 2).textContent().catch(() => `Table ${t}`);
      console.log(`\n  Table ${t}: ${title}`);
      const rows = await tables[t].locator('tbody tr').all();
      for (let r = 0; r < rows.length; r++) {
        const cells = await rows[r].locator('td').allTextContents();
        const isWarn = await rows[r].evaluate(el => el.classList.contains('flow-warn'));
        console.log(`    ${cells[0]} | ${cells[1]} | ${cells[2]} | ${cells[3]}${isWarn ? ' [WARN]' : ''}`);
      }
    }

    // Auth detail panels
    const details = await modal.locator('.visualize-auth-detail').all();
    console.log(`\n--- AUTH DETAILS (${details.length}) ---`);
    for (let i = 0; i < details.length; i++) {
      const title = await details[i].locator('.visualize-auth-detail-title').textContent().catch(() => '');
      const dlText = await details[i].evaluate(el => {
        const dl = el.querySelector('dl');
        if (!dl) return '';
        const items = Array.from(dl.querySelectorAll('dt, dd'));
        return items.map((item, idx) => (idx % 2 === 0 ? `\n    ${item.textContent}` : `: ${item.textContent}`)).join('');
      });
      console.log(`  ${title}${dlText}`);
    }

    // Status tiles
    const tiles = await modal.locator('.visualize-status-tile').all();
    console.log(`\n--- STATUS TILES (${tiles.length}) ---`);
    for (let i = 0; i < tiles.length; i++) {
      const label = await tiles[i].locator('.visualize-status-label').textContent().catch(() => '');
      const detail = await tiles[i].locator('.visualize-status-detail').textContent().catch(() => '');
      const classes = await tiles[i].evaluate(el => Array.from(el.classList).join(' '));
      console.log(`  ${label}: ${detail} (${classes})`);
    }

    // Notes
    const notes = await modal.locator('.visualize-auth-notes li').allTextContents().catch(() => []);
    if (notes.length > 0) {
      console.log(`\n--- NOTES ---`);
      notes.forEach(n => console.log(`  - ${n}`));
    }

    // Screenshot
    const safeName = hostname.replace(/[^a-z0-9]/gi, '_');
    await page.screenshot({ path: `/tmp/visualize-${safeName}.png` });
    console.log(`\nScreenshot: /tmp/visualize-${safeName}.png`);

    // Close
    await page.locator('.visualize-modal .modal-close').click();
    await page.waitForTimeout(500);
  }

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
