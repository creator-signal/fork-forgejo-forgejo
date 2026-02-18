// @ts-check
import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';
import {screenshot} from './shared/screenshots.ts';

test.use({user: 'user2'});

test('Change git note', async ({page}) => {
  let response = await page.goto('/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d');
  expect(response?.status()).toBe(200);

  // An add button should not be present, because the commit already has a commit note
  await expect(page.locator('#commit-notes-add-button')).toHaveCount(0);

  let renderedarea = page.locator('#commit-notes-display-area pre.commit-body');
  await expect(renderedarea).toBeVisible();
  let textarea = page.locator('textarea[name="notes"]');
  await expect(textarea).not.toBeVisible();

  await page.locator('#commit-notes-edit-button').click();

  await expect(renderedarea).not.toBeVisible();
  await expect(textarea).toBeVisible();
  await textarea.fill('This is a new note <script>alert("xss")</script>.\nSee https://frogejo.org.');
  await screenshot(page, page.locator('.ui.container.fluid.padded'));

  await page.locator('#notes-save-button').click();

  await expect(renderedarea).toBeVisible();
  await expect(textarea).not.toBeVisible();
  await expect(renderedarea).toHaveText('This is a new note <script>alert("xss")</script>.\nSee https://frogejo.org.');
  await expect(renderedarea.locator('a')).toHaveAttribute('href', 'https://frogejo.org');
  await screenshot(page, page.locator('.ui.container.fluid.padded'));

  response = await page.goto('/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d');
  expect(response?.status()).toBe(200);

  textarea = page.locator('textarea[name="notes"]');
  await expect(textarea).toHaveText('This is a new note <script>alert("xss")</script>.\nSee https://frogejo.org.');
  await expect(textarea.locator('a')).toHaveCount(0);
  await screenshot(page, page.locator('.ui.container.fluid.padded'));
});
