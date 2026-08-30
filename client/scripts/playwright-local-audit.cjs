const assert = require("node:assert/strict");
const { chromium } = require("@playwright/test");

const clientURL = "http://localhost:3020";
const adminURL = "http://localhost:3010";

async function login(page, baseURL, phone, password) {
  await page.goto(`${baseURL}/login`);
  await page.getByLabel("شماره موبایل").fill(phone);
  await page.getByLabel("رمز عبور").fill(password);
  await page.getByRole("button", { name: "ورود" }).click();
}

async function main() {
  const browser = await chromium.launch({ headless: true });
  const client = await browser.newPage();
  const admin = await browser.newPage();
  const browserErrors = [];
  for (const page of [client, admin]) {
    page.on("pageerror", (error) => browserErrors.push(error.message));
  }

  await client.goto(`${clientURL}/login`);
  await client.getByLabel("شماره موبایل").fill("123");
  await client.getByLabel("رمز عبور").fill("anything");
  await client.getByRole("button", { name: "ورود" }).click();
  await assert.doesNotReject(() => client.getByText("شماره موبایل معتبر نیست").waitFor());

  await login(client, clientURL, "9120000000", "Client@Pass1999");
  await client.waitForURL(/\/(select-business|businesses\/\d+\/dashboard)/);
  await client.goto(`${clientURL}/businesses/new`);
  const name = `QA Playwright ${Date.now()}`;
  await client.getByLabel("نام کسب‌وکار").fill(name);
  await client.getByRole("button", { name: "ساخت کسب‌وکار", exact: true }).click();
  await client.waitForURL(/\/businesses\/(\d+)\/dashboard/);
  const businessID = client.url().match(/\/businesses\/(\d+)\//)?.[1];
  assert.ok(businessID, "creating a client business must navigate to its dashboard");

  await client.goto(`${clientURL}/businesses/${businessID}/settings`);
  const renamed = `${name} renamed`;
  await client.getByLabel("نام کسب‌وکار").fill(renamed);
  await client.getByRole("button", { name: "ذخیره", exact: true }).click();
  await client.getByText("تنظیمات ذخیره شد").waitFor();
  await client.goto(`${clientURL}/businesses/${businessID}/members`);
  await client.getByRole("heading", { name: "اعضا" }).waitFor();

  await login(admin, adminURL, "9370843199", "Amir@Pass1999");
  await admin.waitForURL(`${adminURL}/`);
  await admin.goto(`${adminURL}/users`);
  await admin.getByRole("heading", { name: "کاربران" }).waitFor();
  await admin.goto(`${adminURL}/businesses`);
  await admin.getByRole("heading", { name: "کسب‌وکارها" }).waitFor();
  await admin.getByText(renamed).waitFor();
  await admin.goto(`${adminURL}/businesses/${businessID}`);
  await admin.getByText(renamed).first().waitFor();

  await client.goto(`${clientURL}/businesses/${businessID}/settings`);
  client.once("dialog", (dialog) => dialog.accept());
  await client.getByRole("button", { name: "حذف کسب‌وکار" }).click();
  await client.waitForURL(`${clientURL}/select-business`);

  assert.deepEqual(browserErrors, [], `unexpected browser errors:\n${browserErrors.join("\n")}`);
  await browser.close();
  console.log("Playwright local audit passed: client auth, business CRUD/members, static deep links, and admin users/businesses.");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
