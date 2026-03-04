// @watch start
// templates/explore/navbar.tmpl
// templates/user/dashboard/navbar.tmpl
// templates/repo/header.tmpl
// templates/org/header.tmpl
// web_src/js/components/DashboardRepoList.vue
// web_src/js/modules/tippy.js
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e';

test.describe(`Visual properties`, () => {
  test(`Overflow menu`, async ({browser, isMobile}) => {
    test.skip(!isMobile, 'Overflow menu button only appears on mobile');

    const context = await browser.newContext({javaScriptEnabled: true});
    const page = await context.newPage();

    await page.goto(`/panc/test1`);
    const selectorPrefix = '.tippy-box .tippy-content .tippy-target';
    const overflowMenuButton = page.locator(`.overflow-menu-button`);
    await overflowMenuButton.click();

    const menuItems = page.locator(`${selectorPrefix} > a.item`);
    const itemCount = await menuItems.count();
    for (let i = 0; i < itemCount; i++) {
      const item = page.locator(`${selectorPrefix} > a.item`).nth(i);
      await item.click();
      await page.waitForLoadState('networkidle');

      await overflowMenuButton.click();
      const activeItem = page.locator(`${selectorPrefix} > a.item.active`);
      await expect(activeItem).toHaveCSS(`background-color`, `rgb(226, 226, 229)`);
    }
  });
});
