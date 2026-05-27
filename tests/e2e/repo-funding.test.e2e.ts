// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/view_file.tmpl
// templates/shared/funding.tmpl
// templates/shared/sponsor_button.tmpl
// web_src/css/modules/dialog.css
// web_src/css/modules/message.css
// web_src/css/repo/header.css
// web_src/css/user.css
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';
import {screenshot} from './shared/screenshots.ts';
import {accessibilityCheck} from './shared/accessibility.ts';

test('Sponsor modal', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const response = await page.goto('/user2/funding_basic_complete', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
  await expect(sponsorModal).toBeVisible();
  await expect(sponsorModal.locator('.ui.error.message')).toBeHidden();

  // const custom = sponsorModal.getByRole("listitem").filter({hasText: 'https://example.com'});
  const ko_fi = sponsorModal.getByRole('listitem').filter({hasText: 'ko-fi.com/example'});
  // await expect(custom.locator('svg')).toHaveAccessibleName('custom'); // TODO: not sure how to do svg alt text yet
  await expect(ko_fi.locator('img')).toHaveAccessibleName('ko_fi');

  await screenshot(page);
});

test('Sponsor button (repo): accessibility', async ({page}) => {
  const response = await page.goto('/user2/funding_basic_complete', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorButton = page.getByRole('button').filter({hasText: 'Sponsor'});
  await expect(sponsorButton).toBeVisible();
  await expect(sponsorButton).toHaveAccessibleName('Sponsor user2/funding_basic_complete');
  await expect(page.locator('#sponsor-modal')).toBeHidden();

  await accessibilityCheck({page}, ['button.sponsor'], [], []);
});

test('Sponsor button (user): accessibility', async ({page}) => {
  const response = await page.goto('/funded_user', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorButton = page.getByRole('button').filter({hasText: 'Sponsor'});
  await expect(sponsorButton).toBeVisible();
  await expect(sponsorButton).toHaveAccessibleName('Sponsor Plz sponsor :3');
  await expect(page.locator('#sponsor-modal')).toBeHidden();

  await accessibilityCheck({page}, ['button.sponsor'], [], []);
});

test('Sponsor modal: accessibility (valid config)', async ({page}) => {
  const response = await page.goto('/user2/funding_basic_complete', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
  await expect(sponsorModal).toBeVisible();
  await expect(sponsorModal.locator('.ui.error.message')).toBeHidden();

  await accessibilityCheck({page}, ['dialog#sponsor-modal'], [], []);
});

test('Sponsor modal: accessibility (config errors)', async ({page}) => {
  const response = await page.goto('/user2/funding_some_valid', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
  await expect(sponsorModal).toBeVisible();
  await expect(sponsorModal.getByRole('heading')).toHaveText('Sponsor user2/funding_some_valid');
  await expect(sponsorModal.locator('.ui.error.message', {hasText: 'The funding config contains errors'})).toBeVisible();

  await accessibilityCheck({page}, ['dialog#sponsor-modal'], [], []);
});

test('Sponsor modal: closes on Esc', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const response = await page.goto('/user2/funding_basic_complete', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
  await expect(sponsorModal).toBeVisible();

  await page.keyboard.press('Escape');
  await expect(sponsorModal).toBeHidden();
});

test('Sponsor modal: closes on outside click', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const response = await page.goto('/user2/funding_basic_complete', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
  await expect(sponsorModal).toBeVisible();

  // not sure if it's possible to select ::backdrop here, so we manually click just outside of the bounding box for the same effect
  const box = await sponsorModal.boundingBox();
  await page.mouse.click(box.x + 1, box.y + 1); // clicking the modal itself does nothing
  await expect(sponsorModal).toBeVisible();
  await page.mouse.click(box.x - 1, box.y);
  await expect(sponsorModal).toBeHidden();
});

test('Sponsor modal: closes on Close button', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const response = await page.goto('/user2/funding_basic_complete', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
  await expect(sponsorModal).toBeVisible();

  await page.getByLabel('Close').click();
  await expect(sponsorModal).toBeHidden();
});

test('Sponsor modal: links to config file on error', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const response = await page.goto('/user2/funding_some_valid', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();

  const sponsorButton = page.getByRole('button').filter({hasText: 'Sponsor'});
  await expect(sponsorButton).toHaveAccessibleName('Sponsor user2/funding_some_valid');
  await sponsorButton.click();
  await expect(sponsorModal).toBeVisible();
  await expect(sponsorModal.getByRole('heading')).toHaveText('Sponsor user2/funding_some_valid');

  await expect(sponsorModal.locator('.ui.error.message', {hasText: 'The funding config contains errors'})).toBeVisible();
  await page.getByText('funding config').click();
  await page.waitForURL('/user2/funding_some_valid/src/branch/main/.forgejo/FUNDING.yml');

  const errors = page.locator('.ui.error.message').filter({hasText: 'Error parsing funding config:'});
  await expect(sponsorModal).toBeHidden();
  await expect(errors).toBeVisible();
  await expect(errors).toContainText('ko_fi has an invalid type. Expected string or string array');
});

test('Sponsor button (user): appears when a user profile has a valid funding config', async ({ browser }) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  let response = await page.goto('/user2', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);
  await expect(page.locator('#sponsor-modal')).toBeHidden();
  await expect(page.getByRole('button').filter({ hasText: 'Sponsor' })).toBeHidden();

  response = await page.goto('/funded_user', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  const sponsorButton = page.getByRole('button').filter({hasText: 'Sponsor'});
  await expect(sponsorButton).toBeVisible();
  await sponsorButton.click();

  await expect(sponsorModal).toBeVisible();
  await expect(sponsorModal.getByRole('heading')).toHaveText('Sponsor Plz sponsor :3');
})

// TODO: check the Close button floats to the end (right or left depending on text RTL-ness)
// TODO: check the modal text is a reasonable size and spacing, even on mobile
// TODO: check the various error layouts
// TODO: check with ridiculously long repo/user names
// TODO: check with attempted XSS cases
