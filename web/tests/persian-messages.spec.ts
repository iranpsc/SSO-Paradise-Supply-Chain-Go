import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.route("**/api/account", route => route.fulfill({ status: 401, json: { message: "Unauthenticated" } }));
});

test("English API errors and validation fields never appear in the form", async ({ page }) => {
  await page.route("**/api/login", route => route.fulfill({ status: 422, json: {
    message: "Validation failed", errors: { email: ["The email field is invalid."] },
  } }));
  await page.goto("/login");
  await page.locator('[name="email"]').fill("member_name");
  await page.locator('[name="password"]').fill("TestPass!2026");
  await page.locator('button[type="submit"]').click();
  await expect(page.locator("#email-error")).toHaveText("مقدار واردشده برای این فیلد معتبر نیست.");
  await expect(page.locator("p[role=alert]")).toContainText("اطلاعات واردشده معتبر نیست.");
  await expect(page.locator("p[role=alert]")).not.toContainText(/[a-z]/i);
});

test("wallet rejection is translated instead of displaying provider text", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(window, "ethereum", { value: { request: async () => {
      throw Object.assign(new Error("User rejected the request."), { code: 4001 });
    } } });
  });
  await page.goto("/login");
  await page.getByRole("button", { name: "ورود با MetaMask" }).click();
  await expect(page.locator("p[role=alert]")).toHaveText("درخواست اتصال یا امضا در کیف پول لغو شد.");
});

test("native browser required and email validation are Persian and clear on edit", async ({ page }) => {
  await page.goto("/password/email");
  const email = page.locator('[name="email"]');
  await expect.poll(() => email.evaluate(element => {
    const input = element as HTMLInputElement;
    input.checkValidity();
    return input.validationMessage;
  })).toBe("تکمیل این فیلد الزامی است.");
  await email.fill("not-an-email");
  await email.evaluate(element => (element as HTMLInputElement).checkValidity());
  await expect(email).toHaveJSProperty("validationMessage", "قالب مقدار واردشده معتبر نیست.");
  await email.fill("member@example.test");
  expect(await email.evaluate(element => (element as HTMLInputElement).checkValidity())).toBe(true);
  await expect(email).toHaveJSProperty("validationMessage", "");
});
