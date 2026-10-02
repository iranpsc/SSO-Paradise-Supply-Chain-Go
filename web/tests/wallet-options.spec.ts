import { test, expect } from "@playwright/test";

test("wallet labels follow the entry mode and use loaded local logos", async ({
  page,
}) => {
  const duplicateKeyErrors: string[] = [];
  page.on("console", (entry) => {
    if (entry.type() === "error" && entry.text().includes("same key"))
      duplicateKeyErrors.push(entry.text());
  });
  await page.route("**/api/account", (route) =>
    route.fulfill({ status: 401, json: { message: "Unauthenticated" } }),
  );
  await page.goto("/login");
  for (const name of ["MetaMask", "WalletConnect"])
    await expect(
      page.getByRole("button", { name: `ورود با ${name}` }),
    ).toBeVisible();
  expect(
    await page
      .locator(".wallet-logo img")
      .evaluateAll((images) =>
        images.every(
          (image) =>
            (image as HTMLImageElement).complete &&
            (image as HTMLImageElement).naturalWidth > 0,
        ),
      ),
  ).toBe(true);
  await page.locator('.auth-tabs a[href="/register"]').click();
  for (const name of ["MetaMask", "WalletConnect"])
    await expect(
      page.getByRole("button", { name: `ثبت نام با ${name}` }),
    ).toBeVisible();
  const username = page.getByLabel("نام کاربری", { exact: true });
  await username.fill("1 invalid");
  await expect(page.locator("#username-error")).toBeVisible();
  await username.fill("member_name");
  await expect(page.locator("#username-error")).toHaveCount(0);
  await page.getByRole("button", { name: "ثبت نام با MetaMask" }).click();
  await expect(page.locator(".wallet-notice")).toContainText("کیف پول وب۳ پیدا نشد");
  await page.locator('.auth-tabs a[href="/login"]').click();
  await expect(page.getByRole("button", { name: "ورود با MetaMask" })).toBeVisible();
  await expect(page.locator(".auth-card form")).toHaveCount(1);
  for (let index = 0; index < 8; index++) {
    const mode = index % 2 === 0 ? "register" : "login";
    await page.locator(`.auth-tabs a[href="/${mode}"]`).click();
    await expect(page).toHaveURL(new RegExp(`/${mode}$`));
    await expect(page.locator(".wallet-options")).toHaveCount(1);
    await expect(page.locator(".wallet-button")).toHaveCount(2);
    await expect(page.locator(".auth-card form")).toHaveCount(1);
  }
  expect(duplicateKeyErrors).toEqual([]);
});
