import { fireEvent, render, screen } from "@testing-library/svelte";
import { beforeEach, describe, expect, test, vi } from "vitest";

import App from "./App.svelte";

const credentials = {
  session: "tab-session",
  csrf: "csrf-value",
  expires_at: "2026-08-13T20:00:00Z",
};

function response(status: number, body?: unknown): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

const status = {
  active_workers: 1,
  canceled_jobs: 0,
  completed_jobs: 1,
  failed_jobs: 0,
  max_workers: 2,
  queued_jobs: 0,
  running_jobs: 0,
  version: "test",
};

function applicationResponse(input: RequestInfo | URL): Response {
  const url = new URL(
    input instanceof Request ? input.url : input,
    location.origin,
  );
  if (url.pathname === "/api/status") return response(200, status);
  if (url.pathname === "/api/jobs") {
    return response(200, {
      jobs: [],
      has_more: false,
      stats: { done: 0, closed: 0, open: 0 },
    });
  }
  if (url.pathname === "/api/stream/events") {
    return new Response(new ReadableStream({ start() {} }), { status: 200 });
  }
  return response(404);
}

describe("App", () => {
  beforeEach(() => {
    sessionStorage.clear();
    history.replaceState(null, "", "/reviews");
    vi.restoreAllMocks();
  });

  test("checks the ambient browser session on mount", () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() => new Promise<Response>(() => {})),
    );
    render(App);

    expect(screen.getByText("Checking browser session…")).toBeInTheDocument();
  });

  test("renders token login when remote authentication is required", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => response(401)),
    );
    render(App);

    expect(
      await screen.findByRole("heading", { name: "Connect to Roborev" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Daemon token")).toHaveAttribute(
      "autocomplete",
      "current-password",
    );
  });

  test("clears the token input after login and renders the review workspace", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(response(401))
      .mockResolvedValueOnce(response(200, credentials))
      .mockImplementation(applicationResponse);
    vi.stubGlobal("fetch", fetchMock);
    render(App);

    const token = await screen.findByLabelText("Daemon token");
    await fireEvent.input(token, { target: { value: "one-time-secret" } });
    await fireEvent.click(screen.getByRole("button", { name: "Connect" }));

    expect(
      await screen.findByRole("region", { name: "Review jobs" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Reviews" })).toHaveAttribute(
      "href",
      "/reviews",
    );
    expect(screen.getByRole("link", { name: "Analytics" })).toHaveAttribute(
      "href",
      "/analytics",
    );
    await fireEvent.click(screen.getByRole("link", { name: "Analytics" }));
    expect(
      screen.getByRole("heading", { name: "Project analytics" }),
    ).toBeInTheDocument();
    expect(location.pathname).toBe("/analytics");
    expect(token).toHaveValue("");
    expect(JSON.stringify(fetchMock.mock.calls)).toContain("one-time-secret");
  });

  test("offers a retry after a bootstrap error", async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(response(200, credentials))
      .mockImplementation(applicationResponse);
    vi.stubGlobal("fetch", fetchMock);
    render(App);

    expect(await screen.findByText("offline")).toBeInTheDocument();
    await fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(
      await screen.findByRole("region", { name: "Review jobs" }),
    ).toBeInTheDocument();
  });

  test("re-bootstraps a local session after disconnect", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(
        input instanceof Request ? input.url : input,
        location.origin,
      );
      if (url.pathname === "/api/ui/session/bootstrap") {
        return response(200, credentials);
      }
      if (url.pathname === "/api/ui/session") return response(204);
      return applicationResponse(input);
    });
    vi.stubGlobal("fetch", fetchMock);
    render(App);

    await screen.findByRole("region", { name: "Review jobs" });
    await fireEvent.click(screen.getByRole("button", { name: "Disconnect" }));

    expect(
      await screen.findByRole("region", { name: "Review jobs" }),
    ).toBeInTheDocument();
    const paths = fetchMock.mock.calls.map(
      ([input]) =>
        new URL(input instanceof Request ? input.url : input, location.origin)
          .pathname,
    );
    expect(paths).toContain("/api/ui/session");
    expect(
      paths.filter((path) => path === "/api/ui/session/bootstrap"),
    ).toHaveLength(2);
  });
});
