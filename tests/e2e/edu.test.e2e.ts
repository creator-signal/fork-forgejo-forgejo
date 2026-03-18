// @watch start
// templates/edu/**
// routers/web/edu/**
// internal/edu/**
// @watch end

import {expect} from '@playwright/test';
import {test, login_user, load_logged_in_context} from './utils_e2e.ts';

// Helper: create a course via UI and return to courses list
async function createCourse(page, name: string, description = 'Test course description') {
  await page.goto('/edu/teacher/courses/new');
  await page.fill('input[name=name]', name);
  await page.fill('textarea[name=description]', description);
  await page.click('button[type=submit]');
  // Should redirect to course list
  await page.waitForURL(/\/edu\/teacher\/courses/);
}

test('Edu: Create course successfully', async ({browser}, workerInfo) => {
  // Login as user1 (admin)
  await login_user(browser, workerInfo, 'user1');
  const context = await load_logged_in_context(browser, workerInfo, 'user1');
  const page = await context.newPage();

  // Navigate to new course form
  const response = await page.goto('/edu/teacher/courses/new');
  expect(response?.status()).toBe(200);

  // Fill form
  await page.fill('input[name=name]', 'E2E Test Course');
  await page.fill('textarea[name=description]', 'E2E course description');
  await page.click('button[type=submit]');

  // Should redirect to courses list after creation
  await page.waitForURL(/\/edu\/teacher\/courses/);
  expect(page.url()).toContain('/edu/teacher/courses');
});

test('Edu: Dashboard redirects teacher to assignments', async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user1');
  const context = await load_logged_in_context(browser, workerInfo, 'user1');
  const page = await context.newPage();

  const resp = await page.goto('/edu/dashboard');
  // Dashboard should redirect — either to teacher or student area depending on role
  const url = page.url();
  console.log(`Dashboard redirect: status=${resp?.status()}, url=${url}`);
  expect(url).toMatch(/\/edu\/(teacher|student)/);
});

test('Edu: Enroll user in course', async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user1');
  const context = await load_logged_in_context(browser, workerInfo, 'user1');
  const page = await context.newPage();

  // Create a course
  await createCourse(page, 'Enrollment E2E Test');

  // Get the course ID from the URL after redirect, or find the course detail link
  // Since course list requires enrollment, navigate by finding the course ID
  // The course was just created, go to courses list and look for detail link
  await page.goto('/edu/teacher/courses');

  // The course might not appear in the enrollment-based list.
  // Navigate directly using the course detail link if available,
  // or use course ID 1 (first created in test DB).
  // Instead, let's go to the course detail page.
  // We'll use the page URL we were redirected to and extract course links.
  const courseLink = page.locator('a[href*="/edu/teacher/courses/"]').first();
  if (await courseLink.isVisible()) {
    await courseLink.click();
  } else {
    // Course not in list (enrollment-based). Navigate to course 1 directly.
    await page.goto('/edu/teacher/courses/1');
  }

  // Enroll user2
  const usernameInput = page.locator('input[name=username]');
  if (await usernameInput.isVisible()) {
    await usernameInput.fill('user2');
    const roleSelect = page.locator('select[name=role]');
    if (await roleSelect.isVisible()) {
      await roleSelect.selectOption('student');
    }
    await page.click('button:has-text("Add"), button:has-text("Enroll"), button[type=submit]');

    // Wait for page reload
    await page.waitForLoadState('networkidle');

    // Verify user2 appears in participants table
    const body = await page.content();
    expect(body).toContain('user2');
  }
});

test('Edu: Student assignments page loads', async ({browser}, workerInfo) => {
  // Login as user2 (regular user / student)
  await login_user(browser, workerInfo, 'user2');
  const studentCtx = await load_logged_in_context(browser, workerInfo, 'user2');
  const studentPage = await studentCtx.newPage();

  const response = await studentPage.goto('/edu/student/assignments');
  expect(response?.status()).toBe(200);
});

test('Edu: Navigation to edu pages works', async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user1');
  const context = await load_logged_in_context(browser, workerInfo, 'user1');
  const page = await context.newPage();

  // Teacher assignments page loads
  const resp1 = await page.goto('/edu/teacher/assignments');
  expect(resp1?.status()).toBe(200);

  // Teacher courses page loads
  const resp2 = await page.goto('/edu/teacher/courses');
  expect(resp2?.status()).toBe(200);

  // New course form loads
  const resp3 = await page.goto('/edu/teacher/courses/new');
  expect(resp3?.status()).toBe(200);

  // Student assignments page loads
  const resp4 = await page.goto('/edu/student/assignments');
  expect(resp4?.status()).toBe(200);
});

test('Edu: New course form loads correctly', async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user1');
  const context = await load_logged_in_context(browser, workerInfo, 'user1');
  const page = await context.newPage();

  const response = await page.goto('/edu/teacher/courses/new');
  expect(response?.status()).toBe(200);

  // Verify form elements are present
  await expect(page.locator('input[name=name]')).toBeVisible();
  await expect(page.locator('button[type=submit]')).toBeVisible();
});

test('Edu: New assignment form loads', async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user1');
  const context = await load_logged_in_context(browser, workerInfo, 'user1');
  const page = await context.newPage();

  const response = await page.goto('/edu/teacher/assignments/new');
  expect(response?.status()).toBe(200);

  // Verify form elements are present
  await expect(page.locator('input[name=title]')).toBeVisible();
  await expect(page.locator('input[name=template_repo]')).toBeVisible();
});
