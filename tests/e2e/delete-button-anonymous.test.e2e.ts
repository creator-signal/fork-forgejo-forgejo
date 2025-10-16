// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.describe('Test delete button availability -- anonymous', () => {
  test(`Delete button on repo home should be NOT visible for anonymous`, async ({page}) => {
    await page.goto(`/user2/file-uploads`);

    await expect(page.locator('[data-tooltip-content="Delete path"] svg.octicon-trash')).toHaveCount(0);
    await expect(page.locator('[data-tooltip-content="Delete path"]')).toHaveCount(0);
  });
});
