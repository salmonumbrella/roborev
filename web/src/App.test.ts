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

describe("App", () => {
  beforeEach(() => {
    sessionStorage.clear();
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

  test("clears the token input after login and renders the foundation", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(response(401))
      .mockResolvedValueOnce(response(200, credentials));
    vi.stubGlobal("fetch", fetchMock);
    render(App);

    const token = await screen.findByLabelText("Daemon token");
    await fireEvent.input(token, { target: { value: "one-time-secret" } });
    await fireEvent.click(screen.getByRole("button", { name: "Connect" }));

    expect(
      await screen.findByRole("heading", { name: "Roborev" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Reviews" })).toHaveAttribute(
      "href",
      "/reviews",
    );
    expect(screen.getByRole("link", { name: "Analytics" })).toHaveAttribute(
      "href",
      "/analytics",
    );
    expect(token).toHaveValue("");
    expect(JSON.stringify(fetchMock.mock.calls)).toContain("one-time-secret");
  });

  test("offers a retry after a bootstrap error", async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(response(200, credentials));
    vi.stubGlobal("fetch", fetchMock);
    render(App);

    expect(await screen.findByText("offline")).toBeInTheDocument();
    await fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(
      await screen.findByRole("heading", { name: "Roborev" }),
    ).toBeInTheDocument();
  });
});
