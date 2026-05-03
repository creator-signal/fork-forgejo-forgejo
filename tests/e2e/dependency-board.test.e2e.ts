import {expect, type Locator, type Page} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.use({user: 'user11'});

const BOARD_URL = '/user11/dependency-board-test/issues/dependency-board';

async function gotoBoard(page: Page) {
  const response = await page.goto(BOARD_URL);
  expect(response?.status()).toBe(200);
  await expect(page.locator('.project-column .issue-card').first()).toBeVisible();
}

async function openDrawer(page: Page, cardText: string): Promise<Locator> {
  const card = page.locator('.issue-card').filter({hasText: cardText});
  await card.click({position: {x: 5, y: 5}});
  const drawer = page.locator('.dep-board-drawer');
  await expect(drawer).toBeVisible({timeout: 5000});
  return drawer;
}

test.describe('Dependency Board', () => {
  test.beforeEach(async ({page}) => gotoBoard(page));

  test('board renders with issue columns', async ({page}) => {
    await expect(page.locator('#issue-dependency-board')).toBeVisible();
    const columns = page.locator('.project-column');
    await expect(columns.first()).toBeVisible();
    for (const name of ['setup database', 'build API', 'add tests', 'write docs', 'deploy']) {
      await expect(page.locator('.issue-card').getByText(name).first()).toBeVisible();
    }
  });

  test('board renders milestone columns', async ({page}) => {
    const msCol = page.locator('.dep-board-milestone-col');
    await expect(msCol.getByText('v1.0')).toBeVisible();
    await expect(msCol.getByText('v2.0')).toBeVisible();
  });

  test('board is full-width layout', async ({page}) => {
    await expect(page.locator('.ui.container.fluid.padded')).toBeVisible();
  });

  test('navbar and filters on same row', async ({page}) => {
    const row = page.locator('.dep-board__navbar-row');
    await expect(row).toBeVisible();
    await expect(row.locator('a', {hasText: 'Dependency Board'})).toBeVisible();
    await expect(row.locator('.dep-board__filters .ui.dropdown')).toBeVisible();
    await expect(page.locator('#dep-board-filter-state')).toBeVisible();
  });
});

test.describe('Dependency Board Filtering', () => {
  test.beforeEach(async ({page}) => gotoBoard(page));

  test('filter by milestone', async ({page}) => {
    await page.evaluate(() => {
      const sel = document.querySelector('#dep-board-filter-milestone') as HTMLSelectElement;
      if (sel) {
        const opt = sel.querySelector('option[value]:not([value=""])') as HTMLOptionElement;
        if (opt) {
          sel.value = opt.value;
          sel.dispatchEvent(new Event('change', {bubbles: true}));
        }
      }
    });

    for (const name of ['setup database', 'build API', 'add tests', 'deploy']) {
      await expect(page.locator('.issue-card').getByText(name).first()).toBeVisible();
    }
    await expect(page.locator('.issue-card').getByText('write docs')).toBeHidden();
  });

  test('filter by state shows open by default', async ({page}) => {
    await expect(page.locator('.issue-card').getByText('setup database').first()).toBeVisible();
    await expect(page.locator('.issue-card').getByText('write docs').first()).toBeVisible();
  });
});

test.describe('Dependency Board Issue Selection', () => {
  test.beforeEach(async ({page}) => gotoBoard(page));

  test('clicking issue highlights connected issues', async ({page}) => {
    const buildApi = page.locator('.issue-card').filter({hasText: 'build API'});
    await buildApi.click();

    await expect(buildApi).toHaveClass(/dep-board-selected/);

    const setupDb = page.locator('.issue-card').filter({hasText: 'setup database'});
    const addTests = page.locator('.issue-card').filter({hasText: 'add tests'});
    await expect(setupDb).not.toHaveClass(/dep-board-dimmed/);
    await expect(addTests).not.toHaveClass(/dep-board-dimmed/);
  });
});

test.describe('Dependency Board Issue Pane', () => {
  test.beforeEach(async ({page}) => gotoBoard(page));

  test('clicking issue opens pane with details', async ({page}) => {
    const drawer = await openDrawer(page, 'setup database');

    await expect(drawer.getByText('setup database').first()).toBeVisible();
    await expect(drawer.getByText('initialize the database').first()).toBeVisible();
    await expect(drawer.locator('.dep-board-drawer-header button')).toBeVisible();
  });

  test('close pane button closes drawer', async ({page}) => {
    const drawer = await openDrawer(page, 'setup database');

    await drawer.locator('.dep-board-drawer-header button').click();
    await expect(drawer).toBeHidden();
  });

  test('edit title in pane does not navigate away', async ({page}) => {
    const drawer = await openDrawer(page, 'deploy');

    await drawer.locator('#issue-title-edit-show').click();

    const titleInput = drawer.locator('#issue-title-editor input');
    await titleInput.fill('deploy (updated)');
    await drawer.locator('#issue-title-editor .ui.primary.button').click();

    await expect(drawer).toBeVisible();
    await expect(page).toHaveURL(/dependency-board/);
  });

  test('create blocking issue button visible in pane', async ({page}) => {
    const drawer = await openDrawer(page, 'setup database');

    await expect(drawer.getByRole('button', {name: /Create blocking issue/i})).toBeVisible();
  });
});
