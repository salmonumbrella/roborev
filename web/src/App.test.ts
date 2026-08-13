import { render, screen } from "@testing-library/svelte";
import { describe, expect, test } from "vitest";

import App from "./App.svelte";

describe("App", () => {
  test("renders the browser foundation", () => {
    render(App);

    expect(
      screen.getByRole("heading", { name: "Roborev" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Browser foundation ready")).toBeInTheDocument();
  });
});
