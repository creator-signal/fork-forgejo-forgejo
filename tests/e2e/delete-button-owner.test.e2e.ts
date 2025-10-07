// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.describe('Test delete button availability -- user2', () => {
  test.use({user: 'user2'});

  test(`Delete button should be visible for user2`, async ({page}) => {
    await page.goto(`/user2/file-uploads`);

    await expect(page.locator('[data-tooltip-content="Delete file"] svg.octicon-trash')).toBeVisible();
    await expect(page.locator('[data-tooltip-content="Delete file"]')).toBeVisible();
    await expect(page.locator('[data-tooltip-content="Delete file"]')).toHaveAttribute('aria-label', 'Delete file');
  });
});
