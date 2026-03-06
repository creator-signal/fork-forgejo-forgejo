// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

import {expect} from '@playwright/test';
import {test, login_user, login} from './utils_e2e.ts';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('User: Dynamic profile fields', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/user/settings');

  // Add social media field
  await page.click('.dynamic-fields-container[data-prefix="social"] .add-field-button');
  const socialNameInput = page.locator('input[name="social_name[]"]').first();
  const socialValueInput = page.locator('input[name="social_value[]"]').first();
  await socialNameInput.fill('Mastodon');
  await socialValueInput.fill('https://mastodon.social/@user2');

  // Add company field
  await page.click('.dynamic-fields-container[data-prefix="company"] .add-field-button');
  const companyNameInput = page.locator('input[name="company_name[]"]').first();
  const companyValueInput = page.locator('input[name="company_value[]"]').first();
  await companyNameInput.fill('Forgejo Contributors');
  await companyValueInput.fill('Contributor');

  // Save changes
  await page.click('button:has-text("Update profile")');
  await expect(page.getByText('Your profile has been updated.')).toBeVisible();

  // Verify on profile page
  await page.goto('/user2');
  
  // Check social media
  const socialSection = page.locator('li:has(.octicon-globe)');
  await expect(socialSection).toBeVisible();
  await expect(socialSection.getByText('Mastodon:')).toBeVisible();
  await expect(socialSection.getByRole('link', { name: 'https://mastodon.social/@user2' })).toBeVisible();

  // Check company
  const companySection = page.locator('li:has(.octicon-briefcase)');
  await expect(companySection).toBeVisible();
  await expect(companySection.getByText('Forgejo Contributors')).toBeVisible();
  // Ensure value is NOT shown for company
  await expect(companySection.getByText('Contributor')).toBeHidden();

  // Verify dynamic removal
  await page.goto('/user/settings');
  await page.click('.dynamic-fields-container[data-prefix="social"] .remove-field-button');
  await page.click('button:has-text("Update profile")');
  await expect(page.getByText('Your profile has been updated.')).toBeVisible();

  await page.goto('/user2');
  await expect(page.locator('li:has(.octicon-globe)')).toBeHidden();
});
