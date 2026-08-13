import { execFileSync } from "node:child_process";
import { mkdtempSync, readdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, expect, test } from "vitest";

const temporaryDirectories: string[] = [];

afterEach(() => {
  for (const directory of temporaryDirectories.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

test("the package ships only its supported source contract", () => {
  const destination = mkdtempSync(join(tmpdir(), "roborev-ui-pack-"));
  temporaryDirectories.push(destination);
  execFileSync("bun", ["pm", "pack", "--destination", destination], {
    cwd: import.meta.dirname,
    stdio: "pipe",
  });
  const archives = readdirSync(destination).filter((name) => name.endsWith(".tgz"));
  expect(archives).toHaveLength(1);

  const entries = execFileSync("tar", ["-tf", join(destination, archives[0]!)], {
    encoding: "utf8",
  })
    .trim()
    .split("\n")
    .filter(Boolean)
    .sort();

  expect(entries).toEqual([
    "package/README.md",
    "package/package.json",
    "package/src/index.ts",
  ]);
});
