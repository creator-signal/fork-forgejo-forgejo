import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('User Activity Year Filter', async ({page}) => {
  await page.goto('/user2?tab=activity');

  const activityTab = page.locator('.tabular.menu .item.active');
  await expect(activityTab).toContainText('Activity');

  // Check if year selection menu exists (now as overflow-menu)
  const yearMenu = page.locator('overflow-menu');
  await expect(yearMenu).toBeVisible();

  // Check "Last 12 months" is active by default
  const last12MonthsItem = yearMenu.locator('.item.active');
  await expect(last12MonthsItem).toContainText('Last 12 months');

  // Check current year exists in the menu
  const currentYear = new Date().getFullYear().toString();
  const currentYearItem = yearMenu.locator(`.item:has-text("${currentYear}")`);
  await expect(currentYearItem).toBeVisible();

  // Click on the current year
  await currentYearItem.click();

  // URL should contain year parameter
  await expect(page).toHaveURL(new RegExp(`year=${currentYear}`));

  // Current year should now be active
  await expect(yearMenu.locator('.item.active')).toContainText(currentYear);

  // Heatmap should still be visible
  await expect(page.locator('#user-heatmap')).toBeVisible();
});

test('Dashboard Activity Year Filter', async ({page}) => {
  await page.goto('/');

  // Check if year selection menu exists on dashboard
  const yearMenu = page.locator('overflow-menu');
  await expect(yearMenu).toBeVisible();

  // Check "Last 12 months" is active by default
  const last12MonthsItem = yearMenu.locator('.item.active');
  await expect(last12MonthsItem).toContainText('Last 12 months');

  // Click on the current year
  const currentYear = new Date().getFullYear().toString();
  const currentYearItem = yearMenu.locator(`.item:has-text("${currentYear}")`);
  await currentYearItem.click();

  // URL should contain year parameter
  await expect(page).toHaveURL(new RegExp(`year=${currentYear}`));
});
