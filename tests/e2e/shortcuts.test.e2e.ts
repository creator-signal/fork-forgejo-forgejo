// @watch start
// templates/shared/user/shortcuts_popup.tmpl
// web_src/js/features/user-shortcuts.ts
// @watch end

import {expect, firefox} from '@playwright/test';
import {test, login_user, login} from './utils_e2e.ts';

test.skip(({isMobile}) => isMobile, 'Desktop only');

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Code shortcuts', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  expect((await page.goto('/user2/repo2'))?.status()).toBe(200);

  await page.keyboard.press('j');
  await page.keyboard.press('j');
  await page.keyboard.press('j');
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL('/user2/repo2/src/branch/master/test.xml');

  await page.reload();
  await page.keyboard.press('w');
  await expect(page.locator('[name="search"]')).toBeVisible();

  await page.reload();
  await page.keyboard.press('l');
  await page.keyboard.press('2');
  await page.keyboard.press('Enter');
  await expect(page).toHaveURL(/#L2$/);

  // Shortcuts that handle page navigation are flaky on Firefox using Playwright
  if (browser.browserType() === firefox) return;

  await page.keyboard.press('b');
  await expect(page).toHaveURL('/user2/repo2/blame/branch/master/test.xml');
  await page.keyboard.press('b');
  await expect(page).toHaveURL('/user2/repo2/src/branch/master/test.xml');
  await page.keyboard.press('h');
  await expect(page).toHaveURL('/user2/repo2/commits/branch/master/test.xml');
  await page.goBack();
  await page.keyboard.press('r');
  await expect(page).toHaveURL('/user2/repo2/raw/branch/master/test.xml');
  await page.goBack();
  await page.keyboard.press('y');
  await expect(page).toHaveURL(
    '/user2/repo2/src/commit/1032bbf17fbc0d9c95bb5418dabe8f8c99278700/test.xml',
  );
});

test('Issue/pull request filter shortcuts', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  for (const tab of ['issues', 'pulls']) {
    const resp = await page.goto(`/user2/repo1/${tab}`);
    expect(resp?.status()).toBe(200);

    await page.keyboard.press('a');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/assignee=2/);
    await page.locator('body').click();

    await page.keyboard.press('l');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/labels=1/);
    await page.locator('body').click();

    await page.keyboard.press('m');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/milestone=1/);
    await page.locator('body').click();

    await page.keyboard.press('p');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/project=1/);
    await page.locator('body').click();

    await page.keyboard.press('s');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/sort=oldest/);
    await page.locator('body').click();

    await page.keyboard.press('t');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/type=created_by/);
    await page.locator('body').click();

    await page.keyboard.press('u');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL(/poster=0/);
  }
});

test('Navigate shortcuts', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/user2/repo1/pulls');

  await page.keyboard.press('j');
  await page.keyboard.press('j');
  await page.keyboard.press('x');
  await expect(
    page.locator('.keyboard-selected').getByRole('checkbox'),
  ).toBeChecked();
  await page.keyboard.press('k');
  await expect(
    page.locator('.keyboard-selected').getByRole('checkbox'),
  ).toBeChecked({checked: false});

  await page.keyboard.press('/');
  await expect(page.locator('[type="search"]')).toBeFocused();
});

test('Goto and create shortcuts', async ({browser}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  expect((await page.goto('/'))?.status()).toBe(200);

  await page.keyboard.press('g');
  await page.keyboard.press('n');
  await expect(page).toHaveURL('/notifications');

  // Goto shortcuts are flaky on Firefox using Playwright
  if (browser.browserType() === firefox) return;

  await page.keyboard.press('g');
  await page.keyboard.press('i');
  await expect(page).toHaveURL('/issues');

  await page.keyboard.press('g');
  await page.keyboard.press('p');
  await expect(page).toHaveURL('/pulls');

  expect((await page.goto('/user2/repo1'))?.status()).toBe(200);

  await page.keyboard.press('g');
  await page.keyboard.press('a');
  await expect(page).toHaveURL('/user2/repo1/actions');

  await page.keyboard.press('g');
  await page.keyboard.press('c');
  await expect(page).toHaveURL('/user2/repo1');

  await page.keyboard.press('g');
  await page.keyboard.press('i');
  await expect(page).toHaveURL('/user2/repo1/issues');
  await page.keyboard.press('c');
  await expect(page).toHaveURL('/user2/repo1/issues/new');
  await page.goto('/user2/repo1/issues');

  await page.keyboard.press('g');
  await page.keyboard.press('o');
  await expect(page).toHaveURL('/user2/repo1/projects');

  await page.keyboard.press('g');
  await page.keyboard.press('p');
  await expect(page).toHaveURL('/user2/repo1/pulls');
  await page.keyboard.press('c');
  await expect(page).toHaveURL('/user2/repo1/compare/master...master');
  await page.goto('/user2/repo1/pulls');

  await page.keyboard.press('g');
  await page.keyboard.press('r');
  await expect(page).toHaveURL('/user2/repo1/releases');
  await page.keyboard.press('c');
  await expect(page).toHaveURL('/user2/repo1/releases/new');
  await page.goto('/user2/repo1/releases');

  await page.keyboard.press('g');
  await page.keyboard.press('w');
  await expect(page).toHaveURL('/user2/repo1/wiki');
});

test('Open dialog and persist enable setting', async ({
  browser,
}, workerInfo) => {
  const page = await login({browser}, workerInfo);
  await page.goto('/');
  await page.keyboard.press('Shift+?');
  const dialog = page.locator('#shortcuts');
  await expect(dialog).toBeVisible();

  const checkbox = dialog.getByRole('checkbox');
  await expect(checkbox).toBeChecked();

  const saveResponse = page.waitForResponse(
    (resp) =>
      resp.url().includes('/user/settings/appearance/shortcuts') &&
      resp.request().method() === 'POST' &&
      resp.status() === 200,
  );
  await checkbox.uncheck();
  const response = await saveResponse;
  const body = (await response.json()) as {enable_shortcuts: boolean};
  expect(body.enable_shortcuts).toBe(false);

  await page.reload();
  await page.keyboard.press('Shift+?');
  await expect(checkbox).not.toBeChecked();

  await checkbox.check();
  await page.waitForResponse(
    (resp) =>
      resp.url().includes('/user/settings/appearance/shortcuts') &&
      resp.request().method() === 'POST' &&
      resp.status() === 200,
  );

  await page.keyboard.press('ArrowRight');
  await expect(dialog.locator('button.active')).toHaveText('Repo');
  await page.keyboard.press('ArrowLeft');
  await expect(dialog.locator('button.active')).toHaveText('Global');
  await page.keyboard.press('Escape');
  await dialog.isHidden();
});
