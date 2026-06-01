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
  await expect(sponsorModal).toHaveAccessibleName('Sponsor user2/funding_basic_complete');
  await expect(sponsorModal.locator('.ui.error.message')).toBeHidden();

  const items = await sponsorModal.getByRole('listitem').all();
  expect(items).toHaveLength(12);

  const buy_me_a_coffee = items[0];
  await expect(buy_me_a_coffee.locator('a')).toHaveAttribute('href', 'https://buymeacoffee.com/example');
  await expect(buy_me_a_coffee.locator('a')).toHaveText('buymeacoffee.com/example');
  await expect(buy_me_a_coffee.locator('img')).toHaveAccessibleName('buy_me_a_coffee');

  const community_bridge = items[1];
  await expect(community_bridge.locator('a')).toHaveAttribute('href', 'https://funding.communitybridge.org/projects/example');
  await expect(community_bridge.locator('a')).toHaveText('funding.communitybridge.org/projects/example');
  await expect(community_bridge.locator('img')).toHaveAccessibleName('community_bridge');

  const custom1 = items[2];
  await expect(custom1.locator('a')).toHaveAttribute('href', 'https://example.com');
  await expect(custom1.locator('a')).toHaveText('https://example.com');
  // await expect(custom.locator('svg')).toHaveAccessibleName('custom'); // TODO: not sure how to do svg alt text yet

  const custom2 = items[3];
  await expect(custom2.locator('a')).toHaveAttribute('href', 'http://example.com');
  await expect(custom2.locator('a')).toHaveText('example.com');
  // await expect(custom.locator('svg')).toHaveAccessibleName('custom'); // TODO: same

  const github1 = items[4];
  await expect(github1.locator('a')).toHaveAttribute('href', 'https://github.com/sponsors/example');
  await expect(github1.locator('a')).toHaveText('github.com/sponsors/example');
  await expect(github1.locator('img')).toHaveAccessibleName('github');

  const github2 = items[5];
  await expect(github2.locator('a')).toHaveAttribute('href', 'https://github.com/sponsors/example2');
  await expect(github2.locator('a')).toHaveText('github.com/sponsors/example2');
  await expect(github2.locator('img')).toHaveAccessibleName('github');

  const issuehunt = items[6];
  await expect(issuehunt.locator('a')).toHaveAttribute('href', 'https://issuehunt.io/r/example');
  await expect(issuehunt.locator('a')).toHaveText('issuehunt.io/r/example');
  await expect(issuehunt.locator('img')).toHaveAccessibleName('issuehunt');

  const ko_fi = items[7];
  await expect(ko_fi.locator('a')).toHaveAttribute('href', 'https://ko-fi.com/example');
  await expect(ko_fi.locator('a')).toHaveText('ko-fi.com/example');
  await expect(ko_fi.locator('img')).toHaveAccessibleName('ko_fi');

  const liberapay = items[8];
  await expect(liberapay.locator('a')).toHaveAttribute('href', 'https://liberapay.com/example');
  await expect(liberapay.locator('a')).toHaveText('liberapay.com/example');
  await expect(liberapay.locator('img')).toHaveAccessibleName('liberapay');

  const open_collective = items[9];
  await expect(open_collective.locator('a')).toHaveAttribute('href', 'https://opencollective.com/example');
  await expect(open_collective.locator('a')).toHaveText('opencollective.com/example');
  await expect(open_collective.locator('img')).toHaveAccessibleName('open_collective');

  const patreon = items[10];
  await expect(patreon.locator('a')).toHaveAttribute('href', 'https://patreon.com/example');
  await expect(patreon.locator('a')).toHaveText('patreon.com/example');
  await expect(patreon.locator('img')).toHaveAccessibleName('patreon');

  const polar = items[11];
  await expect(polar.locator('a')).toHaveAttribute('href', 'https://polar.sh/example');
  await expect(polar.locator('a')).toHaveText('polar.sh/example');
  await expect(polar.locator('img')).toHaveAccessibleName('polar');

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

  const items = await sponsorModal.getByRole('listitem').all();
  expect(items).toHaveLength(1);
  await expect(items[0].locator('a')).toHaveAttribute('href', 'https://example.com');
  await expect(items[0].locator('a')).toHaveText('https://example.com');
  // await expect(items[0].locator('svg')).toHaveAccessibleName('custom'); // TODO: not sure how to do svg alt text yet

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

for (const colorScheme of ['light', 'dark'] as const) {
  test(`Sponsor modal: all images load (${colorScheme} mode)`, async ({browser}) => {
    const context = await browser.newContext({colorScheme});
    const page = await context.newPage();

    const response = await page.goto('/user2/funding_basic_complete', {waitUntil: 'domcontentloaded'});
    expect(response?.status()).toBe(200);

    await page.getByRole('button').filter({hasText: 'Sponsor'}).click();
    await expect(page.locator('#sponsor-modal')).toBeVisible();

    await page.waitForFunction(() => {
      return Array.from(document.querySelectorAll('#sponsor-modal img')).every((img: HTMLImageElement) => img.complete && img.naturalWidth > 0);
    });
  });
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
  await expect(items[0].locator('a')).toHaveAttribute('href', 'https://example.com');
  await expect(items[0].locator('a')).toHaveText('https://example.com');
  // await expect(items[0].locator('svg')).toHaveAccessibleName('custom'); // TODO: not sure how to do svg alt text yet

  await expect(sponsorModal.locator('.ui.error.message', {hasText: 'The funding config contains errors'})).toBeVisible();
  await page.getByText('funding config').click();
  await page.waitForURL('/user2/funding_some_valid/src/branch/main/.forgejo/FUNDING.yml');

  const errors = page.locator('.ui.error.message').filter({hasText: 'Error parsing funding config:'});
  await expect(sponsorModal).toBeHidden();
  await expect(errors).toBeVisible();
  await expect(errors).toContainText("Invalid type for key 'ko_fi', expected a string or string array");
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
  // strings that don't produce valid URLs are omitted with error
  const items = await sponsorModal.getByRole('listitem').all();
  expect(items).toHaveLength(5);

  await expect(items[0].locator('a')).toHaveAttribute('href', 'http://#%22%20style=%22background:%20url%28localhost%29');
  await expect(items[0].locator('a')).toHaveText('#" style="background: url(localhost)');
  // await expect(items[0].locator('svg')).toHaveAccessibleName('custom'); // TODO: not sure how to do svg alt text yet

  await expect(items[1].locator('a')).toHaveAttribute('href', 'https://example.com/%22%20class=%22rogue%20injection');
  await expect(items[1].locator('a')).toHaveText('https://example.com/" class="rogue injection');
  // await expect(items[1].locator('svg')).toHaveAccessibleName('custom'); // TODO: same

  await expect(items[2].locator('a')).toHaveAttribute('href', 'http://%3Cscript%3Ealert%601%60%3C/script%3E');
  await expect(items[2].locator('a')).toHaveText('<script>alert`1`</script>');
  // await expect(items[2].locator('svg')).toHaveAccessibleName('custom'); // TODO: same

  await expect(items[3].locator('a')).toHaveAttribute('href', 'https://ko-fi.com/%22%3E%3Cscript%3Ealert%281%29%3B%3C%2Fscript%3E%3Ca%20class=%22');
  await expect(items[3].locator('a')).toHaveText('ko-fi.com/"><script>alert(1);</script><a class="');
  await expect(items[3].locator('img')).toHaveAccessibleName('ko_fi');
  await expect(items[3].locator('a *')).toBeHidden(); // no real injected <script>
  await expect(sponsorModal.locator('script')).toBeHidden();

  await expect(items[4].locator('a')).toHaveAttribute('href', 'https://liberapay.com/text%2Fother');
  await expect(items[4].locator('a')).toHaveText('liberapay.com/text/other');
  await expect(items[4].locator('img')).toHaveAccessibleName('liberapay');
});

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

  const custom = items[0];
  await expect(custom.locator('a')).toHaveAttribute('href', 'http://localhost:3003/');
  await expect(custom.locator('a')).toHaveText('http://localhost:3003/');
  // await expect(custom.locator('svg')).toHaveAccessibleName('custom'); // TODO: not sure how to do svg alt text yet

  const ko_fi = items[1];
  await expect(ko_fi.locator('a')).toHaveAttribute('href', 'https://ko-fi.com/example');
  await expect(ko_fi.locator('a')).toHaveText('ko-fi.com/example');
  await expect(ko_fi.locator('img')).toHaveAccessibleName('ko_fi');

  const liberapay = items[2];
  await expect(liberapay.locator('a')).toHaveAttribute('href', 'https://liberapay.com/example');
  await expect(liberapay.locator('a')).toHaveText('liberapay.com/example');
  await expect(liberapay.locator('img')).toHaveAccessibleName('liberapay');
});
