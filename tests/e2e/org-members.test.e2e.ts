// Copyright 2026 The Forgejo Authors
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start

// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.describe('Toggle visibility, remove, leave', () => {
  test('Anon PoV', async ({browser}) => {
    const context = await browser.newContext({javaScriptEnabled: false});
    const page = await context.newPage();

    page.goto('/org/org3/members');
    /* No interactive buttons - though such evaluation is easy to break in rename */
    await expect(await page.locator('.members .list .link-action')).not.toBeVisible();
    await expect(await page.locator('.members .list .delete-button')).not.toBeVisible();
  });

});


