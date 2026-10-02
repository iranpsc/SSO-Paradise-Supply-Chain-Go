import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.route("**/api/account", (route) =>
    route.fulfill({ status: 401, json: { message: "Unauthenticated" } }),
  );
  await page.goto("/register");
  await page.locator('[name="username"]').fill("test_member");
});

test("password requirements appear on focus and track validity live", async ({
  page,
}) => {
  const submit = page.locator('button[type="submit"]');
  await expect(submit).toBeDisabled();
  await expect(page.locator("#password-requirements")).toHaveCount(0);
  await page.locator('[name="password_confirmation"]').focus();
  await expect(page.locator("#password-requirements")).toBeVisible();
  await expect(page.locator('[data-passed="false"]')).toHaveCount(6);
  await page.locator('[name="name"]').fill("HM-admin");
  await expect(page.locator("#name-error")).toBeVisible();
  await expect(page.locator('[name="name"]')).toHaveAttribute(
    "aria-invalid",
    "true",
  );
  await page.locator('[name="name"]').fill("کاربر تست");
  await expect(page.locator("#name-error")).toHaveCount(0);
  await page.locator('[name="email"]').fill("invalid");
  await expect(page.locator("#email-error")).toBeVisible();
  await page.locator('[name="email"]').fill("member@example.com");
  await page.locator('[name="password"]').fill("Ab1!abcd");
  await expect(page.locator('[data-passed="true"]')).toHaveCount(5);
  await page.locator('[name="password_confirmation"]').fill("different");
  await expect(page.locator("#password_confirmation-error")).toBeVisible();
  await expect(submit).toBeDisabled();
  await page.locator('[name="password_confirmation"]').fill("Ab1!abcd");
  await expect(page.locator('[data-passed="true"]')).toHaveCount(6);
  await expect(submit).toBeEnabled();
  await page.locator('[name="password"]').fill("ab1!abcd");
  await expect(page.locator('[data-rule="upper"]')).toHaveAttribute(
    "data-passed",
    "false",
  );
  await expect(page.locator("#password_confirmation-error")).toBeVisible();
  await expect(submit).toBeDisabled();
  await page.locator('.auth-tabs a[href="/login"]').click();
  await expect(page.locator('.auth-tabs a[href="/login"]')).toHaveAttribute(
    "aria-current",
    "page",
  );
  await page.locator('.auth-tabs a[href="/register"]').click();
  await expect(page.locator('.auth-tabs a[href="/register"]')).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(page.locator("#password-requirements")).toHaveCount(0);
  await expect(submit).toBeDisabled();
});

test("submission locks the form and renders server errors below inputs", async ({
  page,
}) => {
  let release!: () => void;
  const waiting = new Promise<void>((resolve) => {
    release = resolve;
  });
  let requests = 0;
  await page.route("**/api/register", async (route) => {
    requests++;
    await waiting;
    await route.fulfill({
      status: 422,
      json: {
        message: "اطلاعات معتبر نیست.",
        errors: {
          email: "این ایمیل قبلاً ثبت شده است.",
          referral: "کد معرف معتبر نیست.",
        },
      },
    });
  });
  await page.locator('[name="name"]').fill("کاربر تست");
  await page.locator('[name="email"]').fill("member@example.com");
  await page.locator('[name="password"]').fill("Ab1!abcd");
  await page.locator('[name="password_confirmation"]').fill("Ab1!abcd");
  await page.locator('[name="referral"]').fill("hm-123");
  const submit = page.locator('button[type="submit"]');
  await expect(submit).toBeEnabled();
  await submit.click();
  await expect(submit).toBeDisabled();
  await expect(submit).toHaveAttribute("aria-busy", "true");
  await expect(page.locator(".submit-spinner")).toBeVisible();
  await expect(page.locator('[name="email"]')).toBeDisabled();
  release();
  await expect(page.locator("#email-error")).toContainText("قبلاً ثبت شده");
  await expect(page.locator("#referral-error")).toBeVisible();
  await expect(page.locator('[name="email"]')).toHaveCSS(
    "border-top-color",
    "rgb(220, 38, 38)",
  );
  await expect(submit).toBeDisabled();
  await page.locator('[name="email"]').fill("another@example.com");
  await expect(submit).toBeDisabled();
  await page.locator('[name="referral"]').fill("");
  await expect(submit).toBeEnabled();
  expect(requests).toBe(1);
});
