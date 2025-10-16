// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

import {expect} from '@playwright/test';
import {test} from './utils_e2e.ts';

test.use({user: 'user2'});

interface TestCase {
  files: string[];
}

async function doUploadDirectly({page}, testCase: TestCase) {
  await page.goto(`/user2/file-uploads/_upload/main/`);
  const dropzone = page.getByRole('button', {name: 'Drop files or click here to upload.'});

  // create the virtual files
  const dataTransfer = await page.evaluateHandle((testCase: TestCase) => {
    const dt = new DataTransfer();
    for (const filename of testCase.files) {
      dt.items.add(new File([`File content of ${filename}`], filename, {type: 'text/plain'}));
    }
    return dt;
  }, testCase);
  // and drop them to the upload area
  await dropzone.dispatchEvent('drop', {dataTransfer});

  // ToDo: Potential race condition: We do not currently wait for the upload to complete.
  // See https://codeberg.org/forgejo/forgejo/pulls/6687#issuecomment-5068272 and
  // https://codeberg.org/forgejo/forgejo/issues/5893#issuecomment-5068266 for details.
  // Workaround is to wait (the uploads are just a few bytes and usually complete instantly)
  //
  // eslint-disable-next-line playwright/no-wait-for-timeout
  await page.waitForTimeout(100);

  await page.getByRole('button', {name: 'Commit changes'}).click();
}

test.describe('Test delete button availability -- user2', () => {
  test(`Delete button should be visible for user2`, async ({page}) => {
    const testFiles: TestCase =
      {
        files: [
          'dir1/file1.txt',
          'dir1/file2.txt',
          'dir1/file3.txt',
          'dir1/file4.txt',
        ],
      };

    await doUploadDirectly({page}, testFiles);
    await expect(page.getByRole('link', {name: 'dir1'})).toBeVisible();
    await page.goto(`/user2/file-uploads/src/branch/main/dir1`);

    await expect(page.locator('[data-tooltip-content="Delete path"] svg.octicon-trash')).toBeVisible();
    await expect(page.locator('[data-tooltip-content="Delete path"]')).toBeVisible();
    await expect(page.locator('[data-tooltip-content="Delete path"]')).toHaveAttribute('aria-label', 'Delete path');
  });

  test(`Delete button on repo homepage should not be visible for user2`, async ({page}) => {
    await page.goto(`/user2/file-uploads`);

    await expect(page.locator('[data-tooltip-content="Delete path"] svg.octicon-trash')).toHaveCount(0);
    await expect(page.locator('[data-tooltip-content="Delete path"]')).toHaveCount(0);
  });
});
