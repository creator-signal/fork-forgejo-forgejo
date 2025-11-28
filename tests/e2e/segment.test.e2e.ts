// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// @watch start
// web_src/css/modules/header-floating.css
// templates/admin/dashboard.tmpl
// @watch end

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.use({user: 'user1'});

test('Segments: floating headers', async ({page}) => {

  await page.goto('/admin');

  const pageContentBox = await page.locator('.page-content').boundingBox();

  const maintHeaderBox = await page.getByText('Maintenance operations').boundingBox();
  const maintSegmentBox = await page.locator('.ui.table.segment:has(form[action="/admin"])').boundingBox();

  const systemHeaderBox = await page.getByText('System status').boundingBox();
  const systemSegmentBox = await page.locator('.ui.table.segment[hx-get="/admin/system_status"]').boundingBox();

  // The first floating header should not have any top margin
  expect(maintHeaderBox.y).toBe(pageContentBox.y);

  // The distance between a segment and it's header
  expect(maintSegmentBox.y - (maintHeaderBox.y + maintHeaderBox.height)).toBeCloseTo(10.5);
  expect(systemSegmentBox.y - (systemHeaderBox.y + systemHeaderBox.height)).toBeCloseTo(10.5);

  // The distance between a segment's header and the previous segment
  expect(systemHeaderBox.y - (maintSegmentBox.y + maintSegmentBox.height)).toBeCloseTo(17.5);
});
