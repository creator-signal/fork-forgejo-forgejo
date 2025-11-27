// @watch start
// templates/repo/editor/edit.tmpl
// templates/repo/editor/patch.tmpl
// web_src/js/features/repo-editor.js
// web_src/css/modules/dialog.ts
// web_src/css/modules/dialog.css
// @watch end

import {expect} from '@playwright/test';
import {dynamic_id, test} from './utils_e2e.ts';
import {screenshot} from './shared/screenshots.ts';

test.use({user: 'user2'});

test('Dialog modal', async ({page}) => {
  let response = await page.goto('/user2/repo1/_new/master', {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  const filename = `${dynamic_id()}.md`;

  await page.getByPlaceholder('Name your file…').fill(filename);
  await page.locator('.monaco-editor').click();
  await page.keyboard.type('Hi, nice to meet you. Can I talk about ');

  await page.locator('.quick-pull-choice input[value="direct"]').click();
  await page.getByRole('button', {name: 'Commit changes'}).click();

  response = await page.goto(`/user2/repo1/_edit/master/${filename}`, {waitUntil: 'domcontentloaded'});
  expect(response?.status()).toBe(200);

  await page.locator('.monaco-editor-container').click();
  await page.keyboard.press('ControlOrMeta+A');
  await page.keyboard.press('Backspace');

  await page.locator('#commit-button').click();
  await screenshot(page);
  await expect(page.locator('#edit-empty-content-modal')).toBeVisible();

  await page.locator('#edit-empty-content-modal .cancel').click();
  await expect(page.locator('#edit-empty-content-modal')).toBeHidden();

  await page.locator('#commit-button').click();
  await page.locator('#edit-empty-content-modal .ok').click();
  await expect(page).toHaveURL(`/user2/repo1/src/branch/master/${filename}`);
});

test('Dialog modal: width', async ({browser, isMobile}) => {
  // This test doesn't need JS and runs a little faster without it
  const context = await browser.newContext({javaScriptEnabled: false});
  const page = await context.newPage();

  await page.goto('/devtest/modal');

  // Open modal with short content
  const shortModal = page.locator('#short-modal article');
  await expect(shortModal).toBeHidden();
  await page.locator('button[command="show-modal"][commandfor="short-modal"]').click();
  await expect(shortModal).toBeVisible();


  // Check it's width
  let width = Math.round((await shortModal.boundingBox()).width);
  if (isMobile) {
    expect(width).toBeLessThan(400);
  } else {
    expect(width).toBe(400);
  }
});
