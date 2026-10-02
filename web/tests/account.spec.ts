import { test, expect } from "@playwright/test";
import { readdir, readFile } from "node:fs/promises";
import path from "node:path";

async function mailLink(email: string, route: string) {
  const directory = process.env.TEST_MAIL_DIRECTORY!;
  for (const file of await readdir(directory)) {
    const text = await readFile(path.join(directory, file), "utf8");
    if (text.includes(`To: ${email}`) && text.includes(route))
      return text.match(/http:\/\/localhost:3100\/[^\s]+/)![0];
  }
  throw new Error("Expected development mail was not delivered");
}
test("register, verify, edit, logout, login and recover a persisted account", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
 const email = `member-${Date.now()}@example.com`;
  await page.goto("/register");
  await page.locator('[name="username"]').fill(`member_${Date.now()}`);
  await page.getByLabel("نام", { exact: true }).fill("کاربر آزمون");
  await page.getByLabel(/ایمیل/).fill(email);
  await page.getByLabel("رمز عبور", { exact: true }).fill("SecurePass!2026");
  await page
    .getByLabel("تکرار رمز عبور", { exact: true })
    .fill("SecurePass!2026");
  await page.getByRole("button", { name: "ایجاد حساب کاربری" }).click();
  await expect(page).toHaveURL(/email\/verify/);
  await page.goto(await mailLink(email, "/email/verify"));
  await expect(
    page.getByRole("heading", { name: "کاربر آزمون، خوش آمدید" }),
  ).toBeVisible();
  await page.getByRole("link", { name: "مدیریت حساب", exact: true }).click();
  await page.getByLabel("نام", { exact: true }).fill("نام تازه");
  await page.getByRole("button", { name: "ذخیره تغییرات" }).click();
  await expect(page.getByRole("status")).toContainText("ذخیره شد");
  await page
    .getByRole("button", { name: "خروج", exact: true })
    .filter({ visible: true })
    .click();
  await expect(page).toHaveURL(/login/);
  await page.getByLabel(/ایمیل/).fill(email);
  await page.getByLabel("رمز عبور", { exact: true }).fill("SecurePass!2026");
  await page.getByRole("button", { name: "ورود به حساب" }).click();
  await expect(
    page.getByRole("heading", { name: "نام تازه، خوش آمدید" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "خروج", exact: true })
    .filter({ visible: true })
    .click();
  await page
    .getByRole("link", { name: "رمز عبور را فراموش کرده‌اید؟" })
    .click();
  await expect(
    page.getByRole("heading", { name: "بازیابی دسترسی" }),
  ).toBeVisible();
  await page.getByLabel(/ایمیل/).fill(email);
  await page.getByRole("button", { name: "ارسال پیوند بازیابی" }).click();
  await expect(page.getByRole("status")).toContainText("پیوند بازیابی");
  await page.goto(await mailLink(email, "/password/reset"));
  await page.getByLabel("رمز جدید", { exact: true }).fill("AnotherPass!2026");
  await page
    .getByLabel("تکرار رمز عبور", { exact: true })
    .fill("AnotherPass!2026");
  await page.getByRole("button", { name: "ذخیره رمز جدید" }).click();
  await expect(page).toHaveURL(/\/home$/);
  expect(errors).toEqual([]);
});

for (const width of [390, 768, 1440]) {
  test(`original shell and responsive content at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/login");
    await expect(
      page.getByRole("heading", { name: "ورود", exact: true }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await page
      .getByRole("button", { name: "باز کردن فهرست", exact: true })
      .filter({ visible: true })
      .click();
    await expect(page.locator("#open00")).toBeVisible();
    await page.locator("#open00 .enable-dark-mode").click();
    await expect(page.locator("html")).toHaveClass(/dark/);
    await page.keyboard.press("Escape");
    await expect(page.locator("#open00")).toBeHidden();
    await page.screenshot({
      path: `test-results/login-${width}.png`,
      fullPage: true,
    });
  });
}
