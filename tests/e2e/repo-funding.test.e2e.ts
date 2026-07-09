// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/view_file.tmpl
// web_src/css/modules/message.css
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';
import {accessibilityCheck} from './shared/accessibility.ts';

test('Sponsor config: error readout on file view', async ({browser}) => {
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

const widthCases = [208, 310, 400, 600] as const;
for (const width of widthCases) {
  test(`Sponsor config: errors readable at ${width}px wide`, async ({browser}) => {
    const context = await browser.newContext({screen: {width, height: 600}});
    const page = await context.newPage();

    const response = await page.goto('/user2/funding_some_valid/src/branch/main/.forgejo/FUNDING.yml', {waitUntil: 'domcontentloaded'});
    expect(response?.status()).toBe(200);

    const errors = page.locator('.ui.error.message').filter({hasText: 'Errors parsing funding config:'});
    await expect(errors).toBeVisible();
    await expect(errors).toBeInViewport({ratio: 1});

    await accessibilityCheck({page}, ['.ui.error.message'], [], []);
  });
}

test('Sponsor config: no error on valid config file view', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const response = await page.goto('/user2/funding_basic_complete/src/branch/main/.forgejo/FUNDING.yml', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  await expect(page.locator('.ui.error.message')).toBeHidden();
});
