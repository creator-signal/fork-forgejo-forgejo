// @watch start
// templates/user/settings/appearance.tmpl
// web_src/css/features/heatmap.css
// @watch end

import {expect} from '@playwright/test';
import {test, login_user, load_logged_in_context} from './utils_e2e.ts';
import {screenshot} from './shared/screenshots.ts';

test.beforeAll(async ({browser}, workerInfo) => {
  // Login user5 for tests
  await login_user(browser, workerInfo, 'user5');
});

test('Heatmap: first day of week setting', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user5');
  const page = await context?.newPage();
  await page.goto('/user/settings/appearance');

  // Find the first day of week dropdown
  const firstDOWDropdown = page.locator('form[action="/user/settings/appearance/first_dow"] .ui.dropdown');
  await expect(firstDOWDropdown).toBeVisible();

  // Check default value is Monday (1)
  await expect(firstDOWDropdown).toContainText('Monday');

  // Change to Sunday (0)
  await firstDOWDropdown.click();
  await page.getByText('Sunday', {exact: true}).first().click();
  // Click the Change first day of week button
  await page.getByRole('button', {name: 'Change first day of week'}).click();

  // Verify success message
  await expect(page.getByText('First day of week has been updated.')).toBeVisible();

  await screenshot(page);
});
