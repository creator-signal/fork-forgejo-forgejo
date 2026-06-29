// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// templates/repo/home.tmpl
// templates/repo/diff/box.tmpl
// templates/repo/issue/view_content/context_menu.tmpl
// web_src/js/features/clipboard.js
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';
import {screenshot} from './shared/screenshots.ts';

test('copy src file path to clipboard', async ({page}) => {
  const response = await page.goto('/user2/repo1/src/branch/master/README.md');
  expect(response?.status()).toBe(200);

  await page.click('[data-clipboard-text]');

  await expect(async () => {
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboardText).toContain('README.md');
  }).toPass();

  await expect(page.getByText('Copied')).toBeVisible();
  await screenshot(page, page.getByText('Copied'), 50);
});

test('copy issue content to clipboard', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/1');
  expect(response?.status()).toBe(200);

  await page.click('#issue-1 .comment-container .context-menu');
  await page.locator('#issue-1 .comment-container .menu').getByText('Copy Markdown').click();

  await expect(async () => {
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboardText).toBe('content for the first issue');
  }).toPass();
});

test('copy comment content copies the original markdown', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/1');
  expect(response?.status()).toBe(200);

  await page.click('#issuecomment-1001 .comment-container .context-menu');
  await page.locator('#issuecomment-1001 .comment-container .menu').getByText('Copy Markdown').click();

  await expect(async () => {
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboardText).toContain('## Lorem Ipsum');
    expect(clipboardText).toContain('**I am not appealed**');
    expect(clipboardText).toContain('`feature`');
  }).toPass();
});

test('copy pull request content to clipboard', async ({page}) => {
  const response = await page.goto('/user2/repo1/pulls/5');
  expect(response?.status()).toBe(200);

  await page.click('#issue-11 .comment-container .context-menu');
  await page.locator('#issue-11 .comment-container .menu').getByText('Copy Markdown').click();

  await expect(async () => {
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboardText).toBe('content for the a pull request');
  }).toPass();
});

test('copy diff file path to clipboard', async ({page}) => {
  const response = await page.goto('/user2/repo1/src/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d/README.md');
  expect(response?.status()).toBe(200);

  await page.click('[data-clipboard-text]');

  await expect(async () => {
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboardText).toContain('README.md');
  }).toPass();

  await expect(page.getByText('Copied')).toBeVisible();
  await screenshot(page, page.getByText('Copied'), 50);
});
