import { test, expect } from '@playwright/test'
import { loginAsUser } from './helpers/auth'

test('ocr test page extracts text via mock provider', async ({ page }) => {
  await loginAsUser(page)
  await page.getByRole('link', { name: 'OCR test' }).click()
  await expect(page.getByRole('heading', { name: /OCR/i })).toBeVisible()

  await page.locator('input[type="file"]').setInputFiles('e2e/fixtures/sample.png')
  await page.getByRole('button', { name: 'Run OCR' }).click()
  await expect(page.getByText(/Acme Plumbing/i)).toBeVisible({ timeout: 30_000 })
})
