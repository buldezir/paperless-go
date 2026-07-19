import { test, expect } from '@playwright/test'
import { loginAsUser, uploadFixture } from './helpers/auth'

test('upload txt document and see it on list', async ({ page }) => {
  await loginAsUser(page)
  await uploadFixture(page, 'sample.txt')
  await expect(page).toHaveURL(/\/document\//)

  await page.getByRole('link', { name: 'Documents', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Documents' })).toBeVisible()
  await expect(page.locator('main a[href*="/document/"]').first()).toBeVisible()
})

test('edit document title on detail page', async ({ page }) => {
  await loginAsUser(page)
  await uploadFixture(page, 'sample.txt')

  await page.getByRole('button', { name: 'Unblock editing' }).click()
  await page.getByLabel('Title', { exact: true }).fill('E2E Edited Title')
  await page.getByRole('button', { name: 'Save corrections' }).click()
  await expect(page.getByRole('heading', { name: 'E2E Edited Title' })).toBeVisible()
})
