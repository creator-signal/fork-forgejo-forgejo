// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/view_file.tmpl
// web_src/css/modules/message.css
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test('Sponsor config: error readout', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const response = await page.goto('/user2/funding_some_valid/src/branch/main/.forgejo/FUNDING.yml', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const errors = page.locator('.ui.error.message').filter({hasText: 'Errors parsing funding config:'});
  await expect(errors).toBeVisible();
  await expect(errors).toContainText("Invalid type for key 'ko_fi', expected a string or string array");
  await expect(errors).toContainText('Unknown funding provider: ko-fi');
});

test('Sponsor config: no error on valid config', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const response = await page.goto('/user2/funding_basic_complete/src/branch/main/.forgejo/FUNDING.yml', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  await expect(page.locator('.ui.error.message')).toBeHidden();
});
