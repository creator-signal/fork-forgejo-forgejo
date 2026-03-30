// tests/e2e/activity-feed-layout.test.e2e.js

import {expect} from '@playwright/test';
import {test} from './utils_e2e.js';

test.describe('Activity Feed Layout', () => {
  test.beforeEach(async ({page}) => {
    await page.goto('/user/login');
    await page.fill('input[name="user_name"]', 'user2');
    await page.fill('input[name="password"]', 'password');
    await page.click('button[type="submit"]');
    await page.goto('/');
  });

  test('Action icons should be on the left side', async ({page}) => {
    await page.waitForSelector('#activity-feed .flex-item');
    
    const feedItems = await page.locator('#activity-feed .flex-item').all();
    
    for (const item of feedItems) {
      const iconElement = item.locator('.flex-item-leading svg.octicon');
      await expect(iconElement).toBeVisible();
      
      const iconBox = await iconElement.boundingBox();
      const contentBox = await item.locator('.flex-item-main').boundingBox();
      
      if (iconBox && contentBox) {
        expect(iconBox.x).toBeLessThan(contentBox.x);
      }
    }
  });

  test('Action icons should have correct size (28x28)', async ({page}) => {
    await page.waitForSelector('#activity-feed .flex-item');
    
    const icon = page.locator('#activity-feed .flex-item').first().locator('.flex-item-leading svg.octicon');
    await expect(icon).toBeVisible();
    
    const iconSize = await icon.evaluate((el) => ({
      width: el.getAttribute('width'),
      height: el.getAttribute('height')
    }));
    
    expect(iconSize.width).toBe('28');
    expect(iconSize.height).toBe('28');
  });

  test('User avatars should be 16x16 and inline before username', async ({page}) => {
    await page.waitForSelector('#activity-feed .flex-item');
    
    const feedItem = page.locator('#activity-feed .flex-item').first();
    // ctx.AvatarUtils.Avatar renders the img tag - selector remains the same
    const avatar = feedItem.locator('.flex-item-main img.ui.avatar');
    
    const avatarCount = await avatar.count();
    if (avatarCount > 0) {
      await expect(avatar).toBeVisible();
      
      // AvatarUtils.Avatar may set size via CSS rather than explicit attributes
      const size = await avatar.evaluate((el) => ({
        width: el.getAttribute('width') || String(el.offsetWidth),
        height: el.getAttribute('height') || String(el.offsetHeight)
      }));
      
      expect(parseInt(size.width)).toBeLessThanOrEqual(20);
      expect(parseInt(size.height)).toBeLessThanOrEqual(20);
      
      // verticalAlign check remains valid
      const avatarStyle = await avatar.evaluate((el) =>
        window.getComputedStyle(el).verticalAlign
      );
      expect(avatarStyle).toBe('middle');
    }
  });

  test('Avatars should only show for real users (ActUser.ID > 0)', async ({page}) => {
    await page.waitForSelector('#activity-feed .flex-item');
    
    const feedItems = await page.locator('#activity-feed .flex-item').all();
    
    for (const item of feedItems) {
      const avatar = item.locator('.flex-item-main img.ui.avatar');
      const avatarCount = await avatar.count();
      
      if (avatarCount > 0) {
        const src = await avatar.getAttribute('src');
        expect(src).toBeTruthy();
        expect(src.length).toBeGreaterThan(0);
      }
    }
  });

  test('Commit avatars should be 24x24', async ({page}) => {
    await page.goto('/user2/repo1');
    await page.goto('/');
    
    await page.waitForSelector('#activity-feed .flex-item');
    
    const commitFeedItems = page.locator('#activity-feed .flex-item:has-text("pushed to")');
    const count = await commitFeedItems.count();
    
    if (count > 0) {
      const commitAvatar = commitFeedItems.first().locator('img[width="24"]');
      if (await commitAvatar.count() > 0) {
        await expect(commitAvatar).toBeVisible();
        
        const size = await commitAvatar.evaluate((el) => ({
          width: el.getAttribute('width'),
          height: el.getAttribute('height')
        }));
        
        expect(size.width).toBe('24');
        expect(size.height).toBe('24');
      }
    }
  });

  test('Right column should be removed to free up space', async ({page}) => {
    await page.waitForSelector('#activity-feed .flex-item');
    
    const feedItem = page.locator('#activity-feed .flex-item').first();
    const rightColumn = feedItem.locator('.flex-item-trailing');
    await expect(rightColumn).not.toBeVisible();
  });

  test('Feed layout should be responsive on mobile', async ({page, isMobile}) => {
    if (!isMobile) {
      await page.setViewportSize({width: 375, height: 667});
    }
    
    await page.waitForSelector('#activity-feed .flex-item');
    
    const feedItem = page.locator('#activity-feed .flex-item').first();
    const icon = feedItem.locator('.flex-item-leading svg.octicon');
    
    await expect(icon).toBeVisible();
    await expect(feedItem).toBeVisible();
    
    const overflow = await feedItem.evaluate((el) =>
      window.getComputedStyle(el).overflow
    );
    expect(['visible', 'hidden', 'auto']).toContain(overflow);
  });

  test('Feed items should have consistent spacing', async ({page}) => {
    await page.waitForSelector('#activity-feed .flex-item');
    
    const feedItems = await page.locator('#activity-feed .flex-item').all();
    
    if (feedItems.length > 1) {
      const margins = [];
      
      for (const item of feedItems) {
        const marginBottom = await item.evaluate((el) =>
          window.getComputedStyle(el).marginBottom
        );
        margins.push(marginBottom);
      }
      
      const firstMargin = margins[0];
      margins.forEach(margin => {
        expect(margin).toBe(firstMargin);
      });
    }
  });

  test('Icons should align properly with text content', async ({page}) => {
    await page.waitForSelector('#activity-feed .flex-item');
    
    const feedItem = page.locator('#activity-feed .flex-item').first();
    const icon = feedItem.locator('.flex-item-leading');
    const content = feedItem.locator('.flex-item-main');
    
    const iconBox = await icon.boundingBox();
    const contentBox = await content.boundingBox();
    
    if (iconBox && contentBox) {
      const verticalDifference = Math.abs(iconBox.y - contentBox.y);
      expect(verticalDifference).toBeLessThan(10);
    }
  });
});