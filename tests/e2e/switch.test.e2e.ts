// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// web_src/css/modules/switch.css
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test('Switch CSS properties', async ({browser}) => {
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const noBg = 'rgba(0, 0, 0, 0)';
  const activeBg = 'rgb(226, 226, 229)';

  const normalMargin = "0px";
  const normalPadding = "15.75px";

  const specialLeftMargin = "-4px";
  const specialPadding = "19.75px";

  await page.goto('/user2/repo1/pulls');

  var firstActive = page.locator('#issue-filters .switch > .item:nth-child(1)');
  await expect(firstActive).toHaveClass(/active/);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(activeBg);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(normalMargin);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).marginRight)).toBe(normalMargin);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(normalPadding);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(normalPadding);

  var secondItem = page.locator('#issue-filters .switch > .item:nth-child(2)');
  await expect(secondItem).not.toHaveClass(/active/);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(noBg);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(specialLeftMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginRight)).toBe(normalMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(specialPadding);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(normalPadding);

  var secondItem = page.locator('#issue-filters .switch > .item:nth-child(3)');
  await expect(secondItem).not.toHaveClass(/active/);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(noBg);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(normalMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginRight)).toBe(normalMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(normalPadding);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(normalPadding);

  await page.goto('/user2/repo1/pulls?state=closed');

  var firstActive = page.locator('#issue-filters .switch > .item:nth-child(1)');
  await expect(firstActive).not.toHaveClass(/active/);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(noBg);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(normalMargin);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).marginRight)).toBe(specialLeftMargin);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(normalPadding);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(specialPadding);

  var secondItem = page.locator('#issue-filters .switch > .item:nth-child(2)');
  await expect(secondItem).toHaveClass(/active/);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(activeBg);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(normalMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginRight)).toBe(normalMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(normalPadding);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(normalPadding);

  var secondItem = page.locator('#issue-filters .switch > .item:nth-child(3)');
  await expect(secondItem).not.toHaveClass(/active/);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(noBg);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(specialLeftMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginRight)).toBe(normalMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(specialPadding);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(normalPadding);

  await page.goto('/user2/repo1/pulls?state=all');

  var firstActive = page.locator('#issue-filters .switch > .item:nth-child(1)');
  await expect(firstActive).not.toHaveClass(/active/);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(noBg);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(normalMargin);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).marginRight)).toBe(normalMargin);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(normalPadding);
  expect(await firstActive.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(normalPadding);

  var secondItem = page.locator('#issue-filters .switch > .item:nth-child(2)');
  await expect(secondItem).not.toHaveClass(/active/);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(noBg);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(normalMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginRight)).toBe(specialLeftMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(normalPadding);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(specialPadding);

  var secondItem = page.locator('#issue-filters .switch > .item:nth-child(3)');
  await expect(secondItem).toHaveClass(/active/);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).backgroundColor)).toBe(activeBg);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginLeft)).toBe(normalMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).marginRight)).toBe(normalMargin);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingLeft)).toBe(normalPadding);
  expect(await secondItem.evaluate((el) => getComputedStyle(el).paddingRight)).toBe(normalPadding);
});
