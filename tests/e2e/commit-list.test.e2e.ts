// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/pulls/commits_list.tmpl
// web_src/css/repo.css
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';
import {screenshot} from './shared/screenshots.ts';

test.describe(`PR commits`, () => {
  test.use({user: 'user2'});

  test('Mobile responsive layout checks', async ({page, isMobile}) => {
    test.skip(!isMobile);

    const response = await page.goto('/user2/repo1/pulls/3/commits');
    expect(response?.status()).toBe(200);

    const commitGroup = page.locator('.commit-group:first-of-type');

    // Date group visibility test
    await expect(commitGroup).toBeVisible();
    await expect(commitGroup.locator('h4')).toBeVisible();

    // Mobile-specific visibility test
    const commit = commitGroup.locator('.commit:first-child');
    await expect(commit.locator('.mobile-shabox')).toBeVisible();
    await expect(commit.locator('.shabox')).toBeHidden();
    await expect(commit.locator('.commit-buttons')).toBeHidden();

    // Mobile-specific grid positioning
    await expect(commit.locator('.author')).toHaveCSS('grid-column-start', '1');
    await expect(commit.locator('.date')).toHaveCSS('grid-column-start', '2');
    await expect(commit.locator('.message')).toHaveCSS('grid-column-end', 'span 2');
    await expect(commit.locator('.mobile-shabox')).toHaveCSS('grid-column-start', '1');
    await expect(commit.locator('details')).toHaveCSS('grid-column-start', '2');

    // Horizontal scrolling to check for overflow
    await expect(page.locator('.commit-group-commits').first()).not.toHaveCSS('overflow-x', 'scroll');

    await screenshot(page);
  });

  test('Dropdown check in mobile viewport', async ({page, isMobile}) => {
    test.skip(!isMobile);

    const response = await page.goto('/user2/repo1/pulls/3/commits');
    expect(response?.status()).toBe(200);

    const commit = page.locator('.commit-group:first-of-type .commit:first-child');

    // Click dropdown btn
    const dropdown = commit.locator('details.dropdown');
    await expect(dropdown).toBeVisible();
    await dropdown.locator('summary').click();

    // List menu items of dropdown
    const menuItem = commit.locator('details.dropdown ul li a'); // repo_path; always visible
    await expect(menuItem).toHaveAttribute('href', '/user2/repo1/src/commit/5f22f7d0d95d614d25a5b68592adb345a4b5c7fd');
    await menuItem.click();
    await page.waitForURL(/.*\/user2\/repo1\/src\/commit\/5f22f7d0d95d614d25a5b68592adb345a4b5c7f/);
    await screenshot(page);
  });

  test('Desktop responsive layout checks', async ({page, isMobile}) => {
    test.skip(isMobile);

    const response = await page.goto('/user2/repo1/pulls/3/commits');
    expect(response?.status()).toBe(200);

    const commitGroup = page.locator('.commit-group:first-of-type');

    // Date group visibility test
    await expect(commitGroup).toBeVisible();
    await expect(commitGroup.locator('h4')).toBeVisible();

    const commit = commitGroup.locator('.commit:first-child');

    // Desktop grid is the default 5‑column template; just assert it’s a grid (I hope it's fine?)
    await expect(commit).toHaveCSS('display', 'grid');

    // Desktop-specific visibility test
    await expect(commit.locator('.mobile-shabox')).toBeHidden();
    await expect(commit.locator('.shabox')).toBeVisible();
    await expect(commit.locator('.commit-buttons')).toBeVisible();

    await screenshot(page);
  });
})
