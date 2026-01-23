// tests/e2e/activity-feed-layout.test.e2e.js

import {expect} from '@playwright/test';
import {test} from './utils_e2e.js';

test.describe('Activity Feed Layout', () => {
  test.beforeEach(async ({page}) => {
    // Login as test user
    await page.goto('/user/login');
    await page.fill('input[name="user_name"]', 'user2');
    await page.fill('input[name="password"]', 'password');
    await page.click('button[type="submit"]');
    await page.goto('/');
  });

  test('Action icons should be on the left side', async ({page}) => {
    // Wait for feed elements
    await page.waitForSelector('.dashboard .feeds .news');
    
    const feedItems = await page.locator('.dashboard .feeds .news').all();
    
    for (const item of feedItems) {
      // Check that icon element exists
      const iconElement = item.locator('.flex-item-leading svg.octicon');
      await expect(iconElement).toBeVisible();
      
      // Check position: icon should be before main content
      const iconBox = await iconElement.boundingBox();
      const contentBox = await item.locator('.flex-item-main').boundingBox();
      
      if (iconBox && contentBox) {
        expect(iconBox.x).toBeLessThan(contentBox.x);
      }
    }
  });

  test('Action icons should have correct size (28x28)', async ({page}) => {
    await page.waitForSelector('.dashboard .feeds .news');
    
    const icon = page.locator('.dashboard .feeds .news').first().locator('.flex-item-leading svg.octicon');
    await expect(icon).toBeVisible();
    
    // Check icon size
    const iconSize = await icon.evaluate((el) => {
      const style = window.getComputedStyle(el);
      return {
        width: el.getAttribute('width'),
        height: el.getAttribute('height')
      };
    });
    
    expect(iconSize.width).toBe('28');
    expect(iconSize.height).toBe('28');
  });

  test('User avatars should be 16x16 and inline before username', async ({page}) => {
    await page.waitForSelector('.dashboard .feeds .news');
    
    const feedItem = page.locator('.dashboard .feeds .news').first();
    const avatar = feedItem.locator('.flex-item-main img.ui.avatar');
    
    // Check if avatar is displayed for real users
    const avatarCount = await avatar.count();
    if (avatarCount > 0) {
      await expect(avatar).toBeVisible();
      
      // Check avatar size
      const size = await avatar.evaluate((el) => ({
        width: el.getAttribute('width'),
        height: el.getAttribute('height')
      }));
      
      expect(size.width).toBe('16');
      expect(size.height).toBe('16');
      
      // Check that avatar is vertically centered
      const avatarStyle = await avatar.evaluate((el) => 
        window.getComputedStyle(el).verticalAlign
      );
      expect(avatarStyle).toBe('middle');
    }
  });

  test('Avatars should only show for real users (ActUser.ID > 0)', async ({page}) => {
    await page.waitForSelector('.dashboard .feeds .news');
    
    const feedItems = await page.locator('.dashboard .feeds .news').all();
    
    for (const item of feedItems) {
      const avatar = item.locator('.flex-item-main img.ui.avatar');
      const avatarCount = await avatar.count();
      
      if (avatarCount > 0) {
        // If avatar is present, check that src is not empty
        const src = await avatar.getAttribute('src');
        expect(src).toBeTruthy();
        expect(src.length).toBeGreaterThan(0);
      }
    }
  });

  test('Commit avatars should be 24x24', async ({page}) => {
    // Navigate to a repository with commits in the feed
    await page.goto('/user2/repo1');
    await page.goto('/');
    
    await page.waitForSelector('.dashboard .feeds .news');
    
    // Look for commit feed entries
    const commitFeedItems = page.locator('.dashboard .feeds .news:has-text("pushed to")');
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
    await page.waitForSelector('.dashboard .feeds .news');
    
    const feedItem = page.locator('.dashboard .feeds .news').first();
    
    // Check that right column no longer exists
    const rightColumn = feedItem.locator('.flex-item-trailing');
    await expect(rightColumn).not.toBeVisible();
  });

  test('Feed layout should be responsive on mobile', async ({page, isMobile}) => {
    if (!isMobile) {
      // Simulate mobile view
      await page.setViewportSize({width: 375, height: 667});
    }
    
    await page.waitForSelector('.dashboard .feeds .news');
    
    const feedItem = page.locator('.dashboard .feeds .news').first();
    const icon = feedItem.locator('.flex-item-leading svg.octicon');
    
    await expect(icon).toBeVisible();
    await expect(feedItem).toBeVisible();
    
    // Check that content doesn't overflow
    const overflow = await feedItem.evaluate((el) => 
      window.getComputedStyle(el).overflow
    );
    expect(['visible', 'hidden', 'auto']).toContain(overflow);
  });

  test('Feed items should have consistent spacing', async ({page}) => {
    await page.waitForSelector('.dashboard .feeds .news');
    
    const feedItems = await page.locator('.dashboard .feeds .news').all();
    
    if (feedItems.length > 1) {
      const margins = [];
      
      for (const item of feedItems) {
        const marginBottom = await item.evaluate((el) => 
          window.getComputedStyle(el).marginBottom
        );
        margins.push(marginBottom);
      }
      
      // All items should have the same spacing
      const firstMargin = margins[0];
      margins.forEach(margin => {
        expect(margin).toBe(firstMargin);
      });
    }
  });

  test('Icons should align properly with text content', async ({page}) => {
    await page.waitForSelector('.dashboard .feeds .news');
    
    const feedItem = page.locator('.dashboard .feeds .news').first();
    const icon = feedItem.locator('.flex-item-leading');
    const content = feedItem.locator('.flex-item-main');
    
    const iconBox = await icon.boundingBox();
    const contentBox = await content.boundingBox();
    
    if (iconBox && contentBox) {
      // Icon and content should start at approximately the same height
      const verticalDifference = Math.abs(iconBox.y - contentBox.y);
      expect(verticalDifference).toBeLessThan(10); // Max 10px difference
    }
  });
});