import {expect, type Page, type Locator} from '@playwright/test';

// returns element that should be covered before taking the screenshot
async function masks(page: Page) : Array<Locator> {
  return [
    page.locator('.ui.avatar'),
    page.locator('.sha'),
    page.locator('#repo_migrating'),
    // update order of recently created repos is not fully deterministic
    page.locator('.flex-item-main').filter({hasText: 'relative time in repo'}),
    page.locator('#activity-feed'),
    page.locator('#user-heatmap'),
    // dynamic IDs in fixed-size inputs
    page.locator('input[value*="dyn-id-"]'),
  ];
}

// replaces elements on the page that cause flakiness
async function screenshot_prepare(page: Page) {
  await page.waitForLoadState('domcontentloaded');
  // Mock/replace dynamic content which can have different size (and thus cannot simply be masked below)
  await page.locator('footer .left-links').evaluate((node) => node.innerHTML = 'MOCK');
  // replace timestamps in repos to mask them later down
  await page.locator('.flex-item-body > relative-time').filter({hasText: /now|minute/}).evaluateAll((nodes) => {
    for (const node of nodes) node.outerHTML = 'relative time in repo';
  });
  // dynamically generated UUIDs
  await page.getByText('dyn-id-').evaluateAll((nodes) => {
    for (const node of nodes) node.innerHTML = node.innerHTML.replaceAll(/dyn-id-[a-f0-9-]+/g, 'dynamic-id');
  });
  // repeat above, work around https://github.com/microsoft/playwright/issues/34152
  await page.getByText('dyn-id-').evaluateAll((nodes) => {
    for (const node of nodes) node.innerHTML = node.innerHTML.replaceAll(/dyn-id-[a-f0-9-]+/g, 'dynamic-id');
  });
  // dynamically created test users
  await page.getByText('e2e-test-').evaluateAll((nodes) => {
    for (const node of nodes) node.innerHTML = node.innerHTML.replaceAll(/e2e-test-[0-9-]+/g, 'e2e-test-user');
  });
  await page.locator('relative-time').evaluateAll((nodes) => {
    for (const node of nodes) node.outerHTML = 'time element';
  });
  // used for instance for security keys
  await page.locator('absolute-date').evaluateAll((nodes) => {
    for (const node of nodes) node.outerHTML = 'time element';
  });
}

export async function screenshot(page: Page, locator?: Locator) {
  // Optionally include visual testing
  if (process.env.VISUAL_TEST) {
    await screenshot_prepare(page);
    if (locator === undefined) {
      await screenshot_full(page);
    } else {
      await screenshot_selective(page, locator);
    }
  }
}

async function screenshot_selective(page: Page, locator: Locator) {
  clip = await locator.boundingBox();
  await expect(page).toHaveScreenshot({
    fullPage: true,
    timeout: 20000,
    clip,
    mask: await masks(page),
  });
}

async function screenshot_full(page: Page) {
  await expect(page).toHaveScreenshot({
    fullPage: true,
    timeout: 20000,
    mask: await masks(page),
  });
}
