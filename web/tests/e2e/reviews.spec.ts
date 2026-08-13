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

  test("filters by project, ref, and closed state", async ({ page }) => {
    await openReviews(page);

    await page.locator(".picker-button").click();
    await page.locator(".repo-item", { hasText: "project-beta" }).click();
    await expect(page.locator(".job-row").first()).toBeVisible();
    await expect(
      page.locator(".repo-name", { hasText: "project-alpha" }),
    ).toHaveCount(0);

    await page.locator(".picker-button").click();
    await page.locator(".dropdown-item", { hasText: /^All Repos$/ }).click();
    const search = page.getByRole("searchbox", { name: "Search by ref" });
    await search.fill("0000000000000000000000000000000000000034");
    await expect(page.locator(".job-row")).toHaveCount(1);
    await expect(jobRow(page, 52)).toBeVisible();

    await search.fill("");
    await expect(jobRow(page, 42)).toBeVisible();
    await page.getByRole("checkbox", { name: "Hide closed" }).check();
    await expect(jobRow(page, 42)).toHaveCount(0);
  });

  test("restores review preferences after a reload", async ({ page }) => {
    await openReviews(page);
    const hideClosed = page.getByRole("checkbox", { name: "Hide closed" });
    await hideClosed.check();
    await expect(jobRow(page, 42)).toHaveCount(0);

    await page.reload();
    await expect(hideClosed).toBeChecked();
    await expect(jobRow(page, 42)).toHaveCount(0);
  });

  test("shows an empty result without losing the filter controls", async ({
    page,
  }) => {
    await openReviews(page);
    await page
      .getByRole("searchbox", { name: "Search by ref" })
      .fill("no-such-fixture-ref");

    await expect(
      page.getByText("No jobs found", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("searchbox", { name: "Search by ref" }),
    ).toBeVisible();
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

  test("renders compact output, persisted logs, and the review prompt", async ({
    page,
  }) => {
    await openReview(page, 53);
    await expect(
      page.getByText("No issues found after consolidated review."),
    ).toBeVisible();
    await expect(
      page.locator(".col-type", { hasText: "compact" }),
    ).toBeVisible();

    await openReview(page, 52);
    await page.getByRole("button", { name: "Log", exact: true }).click();
    await expect(page.getByText("fixture review started")).toBeVisible();
    await expect(page.getByText("streamed analysis complete")).toBeVisible();
    await page.getByRole("button", { name: "Prompt", exact: true }).click();
    await expect(page.locator(".prompt-text")).toHaveText(
      "Review the fixture change",
    );
  });

  test("preserves deep links through reload and browser history", async ({
    page,
  }) => {
    await openReviews(page);
    await jobRow(page, 52).click();
    await expect(page).toHaveURL(/\/reviews\/52$/);

    await page.reload();
    await expect(page.locator(".review-dock-header .job-id")).toContainText(
      "52",
    );
    await page.goBack();
    await expect(page).toHaveURL(/\/reviews$/);
    await expect(
      page.getByRole("region", { name: "Review details" }),
    ).not.toBeVisible();
    await page.goForward();
    await expect(page).toHaveURL(/\/reviews\/52$/);
    await expect(
      page.getByRole("region", { name: "Review details" }),
    ).toBeVisible();
  });

  test("authenticates the fetch-based event stream", async ({ page }) => {
    const streamRequest = page.waitForRequest((request) =>
      request.url().includes("/api/stream/events"),
    );
    await openReviews(page);

    const request = await streamRequest;
    expect(request.headers()["x-roborev-web-session"]).toBeTruthy();
  });

  test("keeps the review table horizontally reachable in a narrow viewport", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 640, height: 720 });
    await openReviews(page);

    const overflow = await page
      .locator(".table-wrapper")
      .evaluate((element) => ({
        clientWidth: element.clientWidth,
        scrollWidth: element.scrollWidth,
        overflowX: getComputedStyle(element).overflowX,
      }));
    expect(overflow.scrollWidth).toBeGreaterThan(overflow.clientWidth);
    expect(overflow.overflowX).toBe("auto");
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
    await actions.getByRole("button", { name: "Reopen" }).click();
    await expect(
      actions.getByRole("button", { name: "Close Review" }),
    ).toBeVisible();
  });

  test("cancels and reruns jobs through authoritative daemon state", async ({
    page,
  }) => {
    await openReview(page, 50);
    const queuedActions = page.getByRole("group", { name: "Review actions" });
    await queuedActions.getByRole("button", { name: "Cancel" }).click();
    await expect(
      page.locator(".review-dock-header .status-badge"),
    ).toContainText("canceled");

    await openReview(page, 48);
    const failedActions = page.getByRole("group", { name: "Review actions" });
    await failedActions.getByRole("button", { name: "Rerun" }).click();
    await expect(
      page.locator(".review-dock-header .status-badge"),
    ).toContainText("queued");
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
