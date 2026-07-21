import { test, expect, type Page } from '@playwright/test'
import { credentials, loginAsUser } from './helpers/auth'

async function mockSetupStatus(
  page: Page,
  status: {
    needs_admin: boolean
    needs_config: boolean
    ocr_provider?: string
    google_vision_api_key_set?: boolean
    mistral_api_key_set?: boolean
    openai_api_key_set?: boolean
  },
) {
  await page.route('**/api/app/setup/status', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        needs_admin: status.needs_admin,
        needs_config: status.needs_config,
        ocr_provider: status.ocr_provider ?? 'mistral',
        google_vision_api_key_set: status.google_vision_api_key_set ?? false,
        mistral_api_key_set: status.mistral_api_key_set ?? false,
        openai_api_key_set: status.openai_api_key_set ?? false,
      }),
    })
  })
}

test('ready install shows normal login', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Paperless Go' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Create your admin account' })).toHaveCount(0)
})

test('needs_admin shows create admin wizard', async ({ page }) => {
  await mockSetupStatus(page, { needs_admin: true, needs_config: true })
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Create your admin account' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Create admin' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Sign in' })).toHaveCount(0)
})

test('needs_config after superuser login shows OCR step', async ({ page }) => {
  await mockSetupStatus(page, {
    needs_admin: false,
    needs_config: true,
    ocr_provider: 'mistral',
    mistral_api_key_set: false,
    openai_api_key_set: false,
  })
  await page.goto('/')
  await page.getByLabel('Email').fill(credentials.super.email)
  await page.getByLabel('Password').fill(credentials.super.password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('heading', { name: 'Configure OCR' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Documents', exact: true })).toHaveCount(0)
})

test('needs_config blocks regular user', async ({ page }) => {
  await mockSetupStatus(page, {
    needs_admin: false,
    needs_config: true,
    mistral_api_key_set: false,
    openai_api_key_set: false,
  })
  await page.goto('/')
  await page.getByLabel('Email').fill(credentials.user.email)
  await page.getByLabel('Password').fill(credentials.user.password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('heading', { name: 'Setup incomplete' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Log out' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Documents', exact: true })).toHaveCount(0)
})

test('configured app still reachable for seeded e2e users', async ({ page }) => {
  await loginAsUser(page)
  await expect(page.getByRole('link', { name: 'Documents', exact: true })).toBeVisible()
})
