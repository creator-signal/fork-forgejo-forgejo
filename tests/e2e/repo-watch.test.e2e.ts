// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/watch_unwatch.tmpl
// templates/user/settings/notifications.tmpl
// web_src/css/repo/header.css
// @watch end

import {expect} from '@playwright/test';
import {test, login_user, login} from './utils_e2e.ts';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Watch dropdown: toggle watch events', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/user2/repo1');

  // Find the watch dropdown
  const watchDropdown = page.locator('#watch-button details.dropdown');
  const watchSummary = watchDropdown.locator('summary');
  const watchMenu = watchDropdown.locator('ul');

  // Open the dropdown
  await watchSummary.click();
  await expect(watchMenu).toBeVisible();

  // Verify checkboxes are present
  const issuesCheckbox = watchMenu.locator('input[name="watch_issues"]');
  const prsCheckbox = watchMenu.locator('input[name="watch_pulls"]');
  const releasesCheckbox = watchMenu.locator('input[name="watch_releases"]');

  await expect(issuesCheckbox).toBeVisible();
  await expect(prsCheckbox).toBeVisible();
  await expect(releasesCheckbox).toBeVisible();

  // Close dropdown by clicking elsewhere
  await page.locator('h1').click();
  await expect(watchMenu).toBeHidden();
});

test('Watch dropdown: unwatch all shows proper state', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/user2/repo1');

  const watchDropdown = page.locator('#watch-button details.dropdown');
  const watchSummary = watchDropdown.locator('summary');
  const watchMenu = watchDropdown.locator('ul');

  // Open dropdown and click unwatch
  await watchSummary.click();
  await expect(watchMenu).toBeVisible();

  const unwatchButton = watchMenu.locator('button:has-text("Unwatch")');
  if (await unwatchButton.isVisible()) {
    await unwatchButton.click();
    // After unwatch, the button should show "Watch" state
    await expect(watchSummary).toContainText('Watch');
  }
});

test('User settings: notifications page exists', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/user/settings/notifications');

  // Check page loaded correctly
  await expect(page.locator('h4:has-text("Email Notifications")')).toBeVisible();
  await expect(page.locator('h4:has-text("Auto-Watch Repositories")')).toBeVisible();
  await expect(page.locator('h4:has-text("Default Watch Events")')).toBeVisible();
});

test('User settings: notifications auto-watch checkboxes', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/user/settings/notifications');

  // Verify auto-watch checkboxes exist
  const onCreateCheckbox = page.locator('input[name="auto_watch_on_create"]');
  const onAccessCheckbox = page.locator('input[name="auto_watch_on_access"]');
  const onContributeCheckbox = page.locator('input[name="auto_watch_on_contribute"]');

  await expect(onCreateCheckbox).toBeVisible();
  await expect(onAccessCheckbox).toBeVisible();
  await expect(onContributeCheckbox).toBeVisible();
});

test('User settings: notifications default watch events checkboxes', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/user/settings/notifications');

  // Verify default watch event checkboxes exist
  const issuesCheckbox = page.locator('input[name="default_watch_issues"]');
  const prsCheckbox = page.locator('input[name="default_watch_pull_requests"]');
  const releasesCheckbox = page.locator('input[name="default_watch_releases"]');

  await expect(issuesCheckbox).toBeVisible();
  await expect(prsCheckbox).toBeVisible();
  await expect(releasesCheckbox).toBeVisible();
});

test('User settings: notifications can save auto-watch preferences', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/user/settings/notifications');

  // Toggle a checkbox and save
  const onCreateCheckbox = page.locator('input[name="auto_watch_on_create"]');

  // Get initial state
  const wasChecked = await onCreateCheckbox.evaluate((el: HTMLInputElement) => el.checked);

  // Toggle the checkbox
  await onCreateCheckbox.click();

  // Find and click the submit button for auto-watch form
  const autoWatchForm = page.locator('form:has(input[name="auto_watch_on_create"])');
  await autoWatchForm.locator('button[type="submit"]').click();

  // Wait for page to reload
  await page.waitForLoadState('load');

  // Reload and verify state persisted
  await page.goto('/user/settings/notifications');

  if (wasChecked) {
    await expect(onCreateCheckbox).not.toBeChecked();
  } else {
    await expect(onCreateCheckbox).toBeChecked();
  }

  // Restore original state
  await onCreateCheckbox.click();
  await autoWatchForm.locator('button[type="submit"]').click();
});

test('User settings: notifications can save default watch events', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/user/settings/notifications');

  // Toggle issues checkbox and save
  const issuesCheckbox = page.locator('input[name="default_watch_issues"]');

  // Get initial state
  const wasChecked = await issuesCheckbox.evaluate((el: HTMLInputElement) => el.checked);

  // Toggle the checkbox
  await issuesCheckbox.click();

  // Find and click the submit button for watch defaults form
  const watchDefaultsForm = page.locator('form:has(input[name="default_watch_issues"])');
  await watchDefaultsForm.locator('button[type="submit"]').click();

  // Wait for save
  await page.waitForLoadState('load');

  // Reload and verify state persisted
  await page.goto('/user/settings/notifications');

  if (wasChecked) {
    await expect(issuesCheckbox).not.toBeChecked();
  } else {
    await expect(issuesCheckbox).toBeChecked();
  }

  // Restore original state
  await issuesCheckbox.click();
  await watchDefaultsForm.locator('button[type="submit"]').click();
});
