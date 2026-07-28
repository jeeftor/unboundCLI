import { chromium } from 'playwright';

const BASE = 'https://caddy-sync.vookie.net';

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  await page.goto(BASE, { waitUntil: 'networkidle' });
  await page.waitForTimeout(3000);

  // Screenshot the full page to see what we're working with
  await page.screenshot({ path: '/tmp/visualize-dashboard.png' });

  // Check what's in the entries table
  const entriesHTML = await page.evaluate(() => {
    const el = document.getElementById('entries');
    return el ? el.innerHTML.substring(0, 2000) : 'NO #entries element';
  });
  console.log('Entries HTML (first 2000 chars):');
  console.log(entriesHTML);

  // Check for any error messages
  const errors = await page.evaluate(() => {
    return Array.from(document.querySelectorAll('.error, .error-message, [class*="error"]')).map(e => e.textContent?.trim()).filter(Boolean);
  });
  console.log('\nErrors:', errors);

  // Check the actual table structure
  const tableInfo = await page.evaluate(() => {
    const tables = document.querySelectorAll('table');
    return Array.from(tables).map(t => ({
      id: t.id,
      class: t.className,
      rowCount: t.querySelectorAll('tbody tr').length,
      firstRow: t.querySelector('tbody tr')?.textContent?.substring(0, 100),
    }));
  });
  console.log('\nTables:', JSON.stringify(tableInfo, null, 2));

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
