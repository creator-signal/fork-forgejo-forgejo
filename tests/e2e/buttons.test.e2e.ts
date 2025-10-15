// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// web_src/css/modules/switch.css
// web_src/css/modules/button.css
// web_src/css/modules/dropdown.css
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.use({user: 'user2'});

test('Buttons and other controls have consistent height', async ({page}) => {
  await page.goto('/user1');

  // The height of dropdown opener and the button should be matching, even in mobile browsers with coarse pointer
  let buttonHeight = (await page.locator('#profile-avatar-card .actions .primary-action').boundingBox()).height;
  const openerHeight = (await page.locator('#profile-avatar-card .actions .dropdown').boundingBox()).height;
  expect(openerHeight).toBe(buttonHeight);

  await page.goto('/notifications');

  // The height should also be consistent with the button on the previous page
  const switchHeight = (await page.locator('.switch').boundingBox()).height;
  expect(buttonHeight).toBe(switchHeight);

  buttonHeight = (await page.locator('.button-row .button[href="/notifications/subscriptions"]').boundingBox()).height;
  expect(buttonHeight).toBe(switchHeight);

  const purgeButtonHeight = (await page.locator('form[action="/notifications/purge"]').boundingBox()).height;
  expect(buttonHeight).toBe(purgeButtonHeight);
});

test('Button colors', async ({page}) => {
  //const context = await browser.newContext({javaScriptEnabled: false});
  //const page = await context.newPage();
  let response = await page.goto('/devtest/buttons');
  expect(response?.status()).toBe(200);

  const transparent = "rgba(0, 0, 0, 0)";
  const primaryBg = await page.locator('button.primary').evaluate((el) => getComputedStyle(el).backgroundColor);
  const secondaryBg = await page.locator('button.secondary').evaluate((el) => getComputedStyle(el).backgroundColor);
  const dangerBg = await page.locator('button.danger').evaluate((el) => getComputedStyle(el).backgroundColor);

  // Evaluate that all buttons have background-color specified
  expect(primaryBg).not.toBe(transparent);
  expect(secondaryBg).not.toBe(transparent);
  expect(dangerBg).not.toBe(transparent);

  // Evaluate that their background-colors are different
  expect(primaryBg).not.toBe(secondaryBg);
  expect(primaryBg).not.toBe(dangerBg);
});
