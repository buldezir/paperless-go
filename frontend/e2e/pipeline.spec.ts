import { test, expect } from '@playwright/test'
import { loginAsUser, uploadFixture } from './helpers/auth'

test('pipeline reaches completed with OCR text', async ({ page }) => {
  await loginAsUser(page)
  await uploadFixture(page, 'sample.png')

  await expect
    .poll(
      async () => {
        const status = await page.getByText(/Status:/i).innerText()
        if (/completed|needs_review/i.test(status)) return 'done'
        if (/failed/i.test(status)) return 'failed'
        return 'pending'
      },
      { timeout: 90_000, intervals: [500, 1000, 2000] },
    )
    .toBe('done')

  await expect(page.getByLabel('OCR text')).toContainText(/Acme Plumbing/i)
})
