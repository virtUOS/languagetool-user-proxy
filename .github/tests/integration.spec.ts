import { test, expect } from '@playwright/test';

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
