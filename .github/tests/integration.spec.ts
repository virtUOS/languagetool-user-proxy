import { test, expect } from '@playwright/test';

test('metrics endpoint requires auth and returns prometheus data', async ({ request }) => {
  const unauthed = await request.get('http://127.0.0.1:8080/metrics');
  expect(unauthed.status()).toBe(401);

  const authed = await request.get('http://127.0.0.1:8080/metrics', {
    headers: { Authorization: 'Basic ' + Buffer.from('prometheus:secret').toString('base64') },
  });
  expect(authed.status()).toBe(200);
  expect(authed.headers()['content-type']).toContain('text/plain');
  expect(await authed.text()).toContain('go_goroutines');
});

test('login, use API key, regenerate, verify', async ({ page, request }) => {
  await page.goto('/');
  await page.waitForURL(/5556\/dex/);

  await page.fill('input[name="login"]', 'test@example.com');
  await page.fill('input[name="password"]', 'test');
  await page.click('button[type="submit"]');

  const grantButton = page.locator('button:has-text("Grant Access")');
  if (await grantButton.isVisible({ timeout: 3000 }).catch(() => false)) {
    await grantButton.click();
  }

  await page.waitForURL('http://127.0.0.1:8080/');

  // Each regeneration triggers two dialogs in sequence:
  // 1. confirm() fires synchronously on click — handled inside Promise.all
  // 2. alert() fires after the fetch resolves — waited for explicitly afterwards
  // Handling them explicitly avoids a race where the alert fires during teardown.
  const [resp1] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/regenerate-key') && r.request().method() === 'POST',
    ),
    page.waitForEvent('dialog').then((d) => d.accept()), // confirm()
    page.click('#regenerateBtn'),
  ]);
  await page.waitForEvent('dialog').then((d) => d.accept()); // alert()
  const { key: key1 } = await resp1.json();
  expect(key1).toHaveLength(64);

  const check1 = await request.post(`http://127.0.0.1:8080/${key1}/v2/check`, {
    form: { text: 'Hello world', language: 'en' },
  });
  expect(check1.status()).toBe(200);

  const [resp2] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/regenerate-key') && r.request().method() === 'POST',
    ),
    page.waitForEvent('dialog').then((d) => d.accept()), // confirm()
    page.click('#regenerateBtn'),
  ]);
  await page.waitForEvent('dialog').then((d) => d.accept()); // alert()
  const { key: key2 } = await resp2.json();
  expect(key2).toHaveLength(64);
  expect(key2).not.toBe(key1);

  const check2 = await request.post(`http://127.0.0.1:8080/${key2}/v2/check`, {
    form: { text: 'Hello world', language: 'en' },
  });
  expect(check2.status()).toBe(200);

  const check3 = await request.post(`http://127.0.0.1:8080/${key1}/v2/check`, {
    form: { text: 'Hello world', language: 'en' },
  });
  expect(check3.status()).toBe(401);
});

test('rate limiting returns 429 after burst is exceeded', async ({ page, request }) => {
  await page.goto('/');
  await page.waitForURL(/5556\/dex/);

  await page.fill('input[name="login"]', 'test@example.com');
  await page.fill('input[name="password"]', 'test');
  await page.click('button[type="submit"]');

  const grantButton = page.locator('button:has-text("Grant Access")');
  if (await grantButton.isVisible({ timeout: 3000 }).catch(() => false)) {
    await grantButton.click();
  }

  await page.waitForURL('http://127.0.0.1:8080/');

  // Regenerate to get a fresh key with a full burst bucket
  const [resp] = await Promise.all([
    page.waitForResponse(
      (r) => r.url().includes('/regenerate-key') && r.request().method() === 'POST',
    ),
    page.waitForEvent('dialog').then((d) => d.accept()),
    page.click('#regenerateBtn'),
  ]);
  await page.waitForEvent('dialog').then((d) => d.accept());
  const { key } = await resp.json();

  // Fire 20 concurrent requests — default burst is 10, so at least some must be 429
  const responses = await Promise.all(
    Array.from({ length: 20 }, () =>
      request.post(`http://127.0.0.1:8080/${key}/v2/check`, {
        form: { text: 'Hello world', language: 'en' },
      })
    )
  );

  const statuses = responses.map((r) => r.status());
  const tooMany = responses.filter((r) => r.status() === 429);

  expect(statuses.every((s) => s === 200 || s === 429)).toBe(true);
  expect(tooMany.length).toBeGreaterThan(0);
  for (const r of tooMany) {
    expect(r.headers()['retry-after']).toBe('1');
  }
});
