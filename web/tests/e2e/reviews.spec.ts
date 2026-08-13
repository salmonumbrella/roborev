import { expect, test } from "@playwright/test";

import { jobRow, openReview, openReviews, selectStatus } from "./support";

test.describe.serial("native review workspace", () => {
  test("loads the seeded daemon and paginates real jobs", async ({ page }) => {
    await openReviews(page);

    await expect(page.locator(".job-row")).toHaveCount(50);
    await expect(page.getByText("project-alpha").first()).toBeVisible();
    await expect(page.getByText("project-beta").first()).toBeVisible();
    await expect(page.locator(".load-more-btn")).toBeVisible();

    await page.locator(".load-more-btn").click();
    await expect(page.locator(".job-row")).toHaveCount(54);
  });

  test("filters and sorts the live listing", async ({ page }) => {
    await openReviews(page);
    await selectStatus(page, "Failed");

    await expect(
      page.locator(".status-badge.status-failed").first(),
    ).toBeVisible();
    await expect(page.locator(".status-badge:not(.status-failed)")).toHaveCount(
      0,
    );

    await selectStatus(page, "All statuses");
    const firstBefore = Number(
      (await page.locator(".col-id .mono").first().textContent())?.trim(),
    );
    await page.locator("th.sortable", { hasText: "ID" }).click();
    const firstAfter = Number(
      (await page.locator(".col-id .mono").first().textContent())?.trim(),
    );
    expect(firstAfter).toBeLessThan(firstBefore);
  });

  test("opens a deep-linked rich review with comments", async ({ page }) => {
    await openReview(page, 52);

    await expect(page.locator(".review-dock-header .job-id")).toContainText(
      "52",
    );
    await expect(
      page.getByRole("heading", { name: "Streaming errors are discarded" }),
    ).toBeVisible();
    await expect(page.locator(".review-content pre.shiki")).toBeVisible();
    await expect(page.locator(".review-content pre.mermaid")).toBeVisible();
    await expect(page.locator(".response-item")).toHaveCount(2);
  });

  test("posts a comment through the daemon mutation contract", async ({
    page,
  }) => {
    await openReview(page, 52);
    const comment = "Browser parity comment";

    await page.locator(".comment-textarea").fill(comment);
    await page.locator(".submit-btn").click();

    await expect(
      page.locator(".response-item", { hasText: comment }),
    ).toHaveCount(1);
    const response = await page.evaluate(async () => {
      const session = sessionStorage.getItem("roborev.web.session");
      const result = await fetch("/api/comments?job_id=52", {
        headers: { "X-Roborev-Web-Session": session ?? "" },
      });
      return { status: result.status, body: await result.text() };
    });
    expect(response.status).toBe(200);
    expect(response.body).toContain(comment);
  });

  test("closes a review and reflects the authoritative result", async ({
    page,
  }) => {
    await openReview(page, 51);
    const actions = page.getByRole("group", { name: "Review actions" });

    await actions.getByRole("button", { name: "Close Review" }).click();
    await expect(actions.getByRole("button", { name: "Reopen" })).toBeVisible();
  });

  test("expands panel members fetched from the daemon", async ({ page }) => {
    await openReviews(page);
    const parent = jobRow(page, 56);
    await expect(parent).toBeVisible();

    await parent.getByRole("button", { name: "Expand panel" }).click();

    await expect(page.locator(".job-row.member")).toHaveCount(2);
    await expect(
      page.locator(".member-name", { hasText: "correctness" }),
    ).toBeVisible();
    await expect(
      page.locator(".member-name", { hasText: "security" }),
    ).toBeVisible();
  });

  test("preserves transplanted keyboard navigation and help", async ({
    page,
  }) => {
    await openReviews(page);

    await page.keyboard.press("?");
    await expect(
      page.getByText("Keyboard Shortcuts", { exact: true }),
    ).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(
      page.getByText("Keyboard Shortcuts", { exact: true }),
    ).toHaveCount(0);

    await page.keyboard.press("j");
    const highlighted = page.locator(".job-row.highlighted");
    await expect(highlighted).toHaveCount(1);
    const id = (
      await highlighted.locator(".col-id .mono").textContent()
    )?.trim();
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(new RegExp(`/reviews/${id}$`));
    await page.keyboard.press("Escape");
    await expect(page).toHaveURL(/\/reviews$/);
  });
});
