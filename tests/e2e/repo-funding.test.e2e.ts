// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/org/header.tmpl
// templates/repo/header.tmpl
// templates/repo/view_file.tmpl
// templates/shared/funding.tmpl
// templates/shared/sponsor_button.tmpl
// templates/shared/user/profile_big_avatar.tmpl
// web_src/css/modules/dialog.css
// web_src/css/modules/message.css
// web_src/css/repo/header.css
// web_src/css/user.css
// @watch end

import {expect, type Locator} from '@playwright/test';
import {test} from './utils_e2e.ts';
import {screenshot} from './shared/screenshots.ts';
import { accessibilityCheck } from './shared/accessibility.ts';

async function expectSponsorEntry(entry: Locator, expectedProvider: string, expectedText: string, expectedUrl: string) {
  await expect(entry.locator('a')).toHaveAttribute('href', expectedUrl);
  await expect(entry.locator('a')).toHaveText(expectedText);
  await expect(entry.locator('.icon')).toHaveAccessibleName(expectedProvider);
}

test('Sponsor modal (repo)', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  // hidden on repo without funding config
  let response = await page.goto('/user2/long-diff-test', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);
  await expect(page.locator('#sponsor-modal')).toBeHidden();
  await expect(page.getByRole('button').filter({hasText: 'Sponsor'})).toBeHidden();

  // shown on repo with funding config
  response = await page.goto('/user2/funding_basic_complete', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
  await expect(sponsorModal).toBeVisible();
  await expect(sponsorModal).toHaveAccessibleName('Sponsor user2/funding_basic_complete');
  await expect(sponsorModal.locator('.ui.error.message')).toBeHidden();

  const items = await sponsorModal.getByRole('listitem').all();
  expect(items).toHaveLength(13);
  await expectSponsorEntry(items[0], 'community_bridge', 'funding.communitybridge.org/projects/example', 'https://funding.communitybridge.org/projects/example');
  await expectSponsorEntry(items[1], 'github', 'github.com/sponsors/example', 'https://github.com/sponsors/example');
  await expectSponsorEntry(items[2], 'github', 'github.com/sponsors/example2', 'https://github.com/sponsors/example2');
  await expectSponsorEntry(items[3], 'issuehunt', 'issuehunt.io/r/example', 'https://issuehunt.io/r/example');
  await expectSponsorEntry(items[4], 'ko_fi', 'ko-fi.com/example', 'https://ko-fi.com/example');
  await expectSponsorEntry(items[5], 'liberapay', 'liberapay.com/example', 'https://liberapay.com/example');
  await expectSponsorEntry(items[6], 'patreon', 'patreon.com/example', 'https://patreon.com/example');
  await expectSponsorEntry(items[7], 'open_collective', 'opencollective.com/example', 'https://opencollective.com/example');
  await expectSponsorEntry(items[8], 'buy_me_a_coffee', 'buymeacoffee.com/example', 'https://buymeacoffee.com/example');
  await expectSponsorEntry(items[9], 'polar', 'polar.sh/example', 'https://polar.sh/example');
  await expectSponsorEntry(items[10], 'thanks_dev', 'thanks.dev/u/gh/example', 'https://thanks.dev/u/gh/example');
  await expectSponsorEntry(items[11], 'custom', 'https://example.com', 'https://example.com');
  await expectSponsorEntry(items[12], 'custom', 'example.com', 'http://example.com');

  await screenshot(page);
});

const urls = [
  '/user2/funding_empty',
  '/funded_user/whoops_all_invalid_funding',
] as const;
for (const url of urls) {
  test(`Sponsor button (repo): hidden when the funding config is empty or invalid (${url})`, async ({ browser }) => {
    // this test doesn't need JS
    const context = await browser.newContext({javaScriptEnabled: false});
    const page = await context.newPage();

    let response = await page.goto(url, {waitUntil: 'domcontentloaded'});
    expect(response?.status()).toBe(200);
    await expect(page.locator('#sponsor-modal')).toBeHidden();
    await expect(page.getByRole('button').filter({hasText: 'Sponsor'})).toBeHidden();
  })
}

test('Sponsor button (user): appears when a user profile has a valid funding config', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  let response = await page.goto('/user2', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);
  await expect(page.locator('#sponsor-modal')).toBeHidden();
  await expect(page.getByRole('button').filter({hasText: 'Sponsor'})).toBeHidden();

  response = await page.goto('/funded_user', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  const sponsorButton = page.getByRole('button').filter({hasText: 'Sponsor'});
  await expect(sponsorButton).toBeVisible();
  await sponsorButton.click();

  await expect(sponsorModal).toBeVisible();
  await expect(sponsorModal.getByRole('heading')).toHaveText('Sponsor Plz sponsor :3');

  const items = await sponsorModal.getByRole('listitem').all();
  expect(items).toHaveLength(3);
  await expectSponsorEntry(items[0], 'ko_fi', 'ko-fi.com/example', 'https://ko-fi.com/example');
  await expectSponsorEntry(items[1], 'liberapay', 'liberapay.com/example', 'https://liberapay.com/example');
  await expectSponsorEntry(items[2], 'custom', 'http://localhost:3003/', 'http://localhost:3003/');
});

test('Sponsor button (org): appears when an org profile has a valid funding config', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  let response = await page.goto('/org25', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);
  await expect(page.locator('#sponsor-modal')).toBeHidden();
  await expect(page.getByRole('button').filter({hasText: 'Sponsor'})).toBeHidden();

  response = await page.goto('/org6', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  const sponsorButton = page.getByRole('button').filter({hasText: 'Sponsor'});
  await expect(sponsorButton).toBeVisible();
  await sponsorButton.click();

  await expect(sponsorModal).toBeVisible();
  await expect(sponsorModal.getByRole('heading')).toHaveText('Sponsor Org Six');

  const items = await sponsorModal.getByRole('listitem').all();
  expect(items).toHaveLength(2);
  await expectSponsorEntry(items[0], 'buy_me_a_coffee', 'buymeacoffee.com/example', 'https://buymeacoffee.com/example');
  await expectSponsorEntry(items[1], 'custom', 'example.com', 'http://example.com');
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

test('Sponsor button (org): accessibility', async ({page}) => {
  const response = await page.goto('/org6', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorButton = page.getByRole('button').filter({hasText: 'Sponsor'});
  await expect(sponsorButton).toBeVisible();
  await expect(sponsorButton).toHaveAccessibleName('Sponsor Org Six');
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

  const items = await sponsorModal.getByRole('listitem').all();
  expect(items).toHaveLength(1);
  await expectSponsorEntry(items[0], 'custom', 'https://example.com', 'https://example.com');

  await accessibilityCheck({page}, ['dialog#sponsor-modal'], [], []);
});

const widths = [208, 310, 400, 600] as const;
for (const testCase of [
  {
    kind: 'tall',
    name: 'funding_with_a_really_ridiculously_long_title_that_doesnt_really_happen_all_that_often_normally_but_could_really_mess_with_things_if_not_handled_properly',
  },
  {
    kind: 'wide',
    name: 'funding_with_a_really_ridiculously_long_title_that_doesnt_really_happen_all_that_often_normally_but_could_really_mess_with_things_if_not_handled_properly',
  },
] as const) {
  for (const width of widths) {
    test(`Sponsor modal (${testCase.kind}): usable at ${width}px wide`, async ({browser}) => {
      const context = await browser.newContext({screen: {width, height: 600}});
      const page = await context.newPage();

      const response = await page.goto(`/user2/${testCase.name}`, {waitUntil: 'domcontentloaded'});
      expect(response?.status()).toBe(200);

      const sponsorModal = page.locator('#sponsor-modal');
      await expect(sponsorModal).toBeHidden();
      await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
      await expect(sponsorModal).toBeVisible();
      await expect(sponsorModal.getByRole('heading')).toBeInViewport({ratio: 1}); // shouldn't have to scroll to access a scrolling modal!

      await expect(sponsorModal.getByRole('heading')).toHaveText(`Sponsor user2/${testCase.name}`);
      await expect(sponsorModal.getByRole('heading')).toBeInViewport({ratio: 1}); // title should remain at least partly visible (perhaps shortened with ellipsis) unless we scroll

      await expect(sponsorModal.locator('.ui.error.message')).toBeHidden();

      const item = sponsorModal.getByRole('listitem').first();
      await expect(item).toBeInViewport({ratio: 1});

      const close = sponsorModal.getByLabel('Close');
      await expect(close).toBeInViewport({ratio: 1});

      await accessibilityCheck({ page }, ['dialog#sponsor-modal'], [], []);
    });
  }
}

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

  const items = await sponsorModal.getByRole('listitem').all();
  expect(items).toHaveLength(1);
  await expectSponsorEntry(items[0], 'custom', 'https://example.com', 'https://example.com');

  await expect(sponsorModal.locator('.ui.error.message', {hasText: 'The funding config contains errors'})).toBeVisible();
  await page.getByText('funding config').click();
  await page.waitForURL('/user2/funding_some_valid/src/branch/main/.forgejo/FUNDING.yml');

  const errors = page.locator('.ui.error.message').filter({hasText: 'Errors parsing funding config:'});
  await expect(sponsorModal).toBeHidden();
  await expect(errors).toBeVisible();
  await expect(errors).toContainText("Invalid type for key 'ko_fi', expected a string or string array");
  await expect(errors).toContainText("Funding provider tidelift is not allowed"); // sad day, probably :(
});

test('Sponsor modal (repo): mitigates XSS', async ({browser}) => {
  // this test doesn't need JS
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  const response = await page.goto('/user2/funding_evil', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const sponsorModal = page.locator('#sponsor-modal');
  await expect(sponsorModal).toBeHidden();
  await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
  await expect(sponsorModal).toBeVisible();
  await expect(sponsorModal.locator('.ui.error.message', {hasText: 'The funding config contains errors'})).toBeVisible();

  // list items should contain encoded strings as given in config; these strings should be interpreted as text, NOT as HTML markup
  // strings that don't produce valid URLs or whose value does not match the regex are omitted with error
  const items = await sponsorModal.getByRole('listitem').all();
  expect(items).toHaveLength(3);
  await expectSponsorEntry(items[0], 'custom', '#" style="background: url(localhost)', 'http://#%22%20style=%22background:%20url%28localhost%29');
  await expectSponsorEntry(items[1], 'custom', 'https://example.com/" class="rogue injection', 'https://example.com/%22%20class=%22rogue%20injection');
  await expectSponsorEntry(items[2], 'custom', '<script>alert`1`</script>', 'http://%3Cscript%3Ealert%601%60%3C/script%3E');

  // no real injected <script>
  await expect(sponsorModal.locator('a *')).toBeHidden();
  await expect(sponsorModal.locator('script')).toBeHidden();
});
