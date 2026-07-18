import { expect, test } from "@playwright/test";

test("landing renders hero, hotkey table, download", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toContainText("Tiles Spliter");
  await expect(page.getByRole("link", { name: /download/i }).first()).toHaveAttribute(
    "href",
    /github\.com\/.+\/releases/,
  );
  await expect(page.getByText("⌥⌘C")).toBeVisible(); // hotkey table driven by shared defaults
  await expect(page.getByText("Half Left")).toBeVisible();
});
