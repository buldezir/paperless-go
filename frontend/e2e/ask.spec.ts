import { test, expect } from '@playwright/test'
import { loginAsUser, uploadFixture } from './helpers/auth'

test('document ask returns assistant reply', async ({ page }) => {
  await loginAsUser(page)
  await uploadFixture(page, 'sample.png')

  await expect
    .poll(async () => page.getByText(/Status:/i).innerText(), {
      timeout: 90_000,
      intervals: [1000, 2000],
    })
    .toMatch(/completed|needs_review/i)

  await page.getByRole('link', { name: 'Ask AI' }).click()
  await expect(page).toHaveURL(/\/ask/)

  await page.getByPlaceholder('Ask a question about this document...').fill('What is the invoice total?')
  await page.getByRole('button', { name: 'Send' }).click()

  await expect(page.getByText(/250/)).toBeVisible({ timeout: 30_000 })
})
