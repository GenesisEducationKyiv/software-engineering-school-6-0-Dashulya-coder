import { test, expect } from '@playwright/test';

test.describe('Page structure', () => {
  test('has email input, repo input, and submit button', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('email-input')).toBeVisible();
    await expect(page.getByTestId('repo-input')).toBeVisible();
    await expect(page.getByTestId('submit-btn')).toBeVisible();
  });
});

test.describe('Happy path', () => {
  test('shows success message and clears fields on 200', async ({ page }) => {
    await page.route('**/api/subscribe', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Subscription successful. Confirmation email sent.' }),
      });
    });

    await page.goto('/');
    await page.getByTestId('email-input').fill('user@example.com');
    await page.getByTestId('repo-input').fill('cli/cli');
    await page.getByTestId('submit-btn').click();

    await expect(page.getByTestId('alert')).toContainText('Subscription successful');
    await expect(page.getByTestId('email-input')).toHaveValue('');
    await expect(page.getByTestId('repo-input')).toHaveValue('');
  });
});

test.describe('Error flows', () => {
  test('shows conflict message on 409', async ({ page }) => {
    await page.route('**/api/subscribe', async route => {
      await route.fulfill({
        status: 409,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'email already subscribed to this repository' }),
      });
    });

    await page.goto('/');
    await page.getByTestId('email-input').fill('user@example.com');
    await page.getByTestId('repo-input').fill('cli/cli');
    await page.getByTestId('submit-btn').click();

    await expect(page.getByTestId('alert')).toContainText('already subscribed');
  });

  test('shows not found message on 404', async ({ page }) => {
    await page.route('**/api/subscribe', async route => {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'repository not found' }),
      });
    });

    await page.goto('/');
    await page.getByTestId('email-input').fill('user@example.com');
    await page.getByTestId('repo-input').fill('owner/nonexistent');
    await page.getByTestId('submit-btn').click();

    await expect(page.getByTestId('alert')).toContainText('not found');
  });

  test('shows network error message on fetch failure', async ({ page }) => {
    await page.route('**/api/subscribe', async route => {
      await route.abort('failed');
    });

    await page.goto('/');
    await page.getByTestId('email-input').fill('user@example.com');
    await page.getByTestId('repo-input').fill('cli/cli');
    await page.getByTestId('submit-btn').click();

    await expect(page.getByTestId('alert')).toContainText('Network error');
  });
});

test.describe('Button state', () => {
  test('button is disabled while request is in flight', async ({ page }) => {
    let resolve!: () => void;
    const deferred = new Promise<void>(res => { resolve = res; });

    await page.route('**/api/subscribe', async route => {
      await deferred;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Subscription successful. Confirmation email sent.' }),
      });
    });

    await page.goto('/');
    await page.getByTestId('email-input').fill('user@example.com');
    await page.getByTestId('repo-input').fill('cli/cli');

    const clickPromise = page.getByTestId('submit-btn').click();

    await expect(page.getByTestId('submit-btn')).toBeDisabled();

    resolve();
    await expect(page.getByTestId('submit-btn')).toBeEnabled();
    await clickPromise;
  });
});
