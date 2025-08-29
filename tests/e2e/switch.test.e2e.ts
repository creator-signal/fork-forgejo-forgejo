// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// web_src/css/modules/switch.css
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test('Switch CSS properties', async ({browser}) => {
  // This test doesn't need JS and runs a little faster without it
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const noBg = 'rgba(0, 0, 0, 0)';
  const activeBg = 'rgb(226, 226, 229)';

  const normalMargin = "0px";
  const normalPadding = "15.75px";

  const specialLeftMargin = "-4px";
  const specialPadding = "19.75px";

  async function evaluateSwitchItem(page, selector, isActive, background, marginLeft, marginRight, paddingLeft, paddingRight ) {
    const item = page.locator(selector);
    if (isActive) {
      await expect(item).toHaveClass(/active/);
    } else {
      await expect(item).not.toHaveClass(/active/);
    }
    expect(await item.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(background);
    expect(await item.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(marginLeft);
    expect(await item.evaluate((el) => getComputedStyle(el).marginRight)).toBe(marginRight);
    expect(await item.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(paddingLeft);
    expect(await item.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(paddingRight);
  }


  await page.goto('/user2/repo1/pulls');

  await evaluateSwitchItem(page, '#issue-filters .switch > .item:nth-child(1)', true, activeBg, normalMargin, normalMargin, normalPadding, normalPadding);
  await evaluateSwitchItem(page, '#issue-filters .switch > .item:nth-child(2)', false, noBg, specialLeftMargin, normalMargin, specialPadding, normalPadding);
  await evaluateSwitchItem(page, '#issue-filters .switch > .item:nth-child(3)', false, noBg, normalMargin, normalMargin, normalPadding, normalPadding);

  await page.goto('/user2/repo1/pulls?state=closed');

  await evaluateSwitchItem(page, '#issue-filters .switch > .item:nth-child(1)', false, noBg, normalMargin, specialLeftMargin, normalPadding, specialPadding);
  await evaluateSwitchItem(page, '#issue-filters .switch > .item:nth-child(2)', true, activeBg, normalMargin, normalMargin, normalPadding, normalPadding);
  await evaluateSwitchItem(page, '#issue-filters .switch > .item:nth-child(3)', false, noBg, specialLeftMargin, normalMargin, specialPadding, normalPadding);

  await page.goto('/user2/repo1/pulls?state=all');

  await evaluateSwitchItem(page, '#issue-filters .switch > .item:nth-child(1)', false, noBg, normalMargin, normalMargin, normalPadding, normalPadding);
  await evaluateSwitchItem(page, '#issue-filters .switch > .item:nth-child(2)', false, noBg, normalMargin, specialLeftMargin, normalPadding, specialPadding);
  await evaluateSwitchItem(page, '#issue-filters .switch > .item:nth-child(3)', true, activeBg, normalMargin, normalMargin, normalPadding, normalPadding);
});
