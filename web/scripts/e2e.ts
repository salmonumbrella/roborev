import { spawn, type ChildProcess } from "node:child_process";
import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { compilationStub } from "./embed-assets";

interface RuntimeRecord {
  metadata?: Record<string, string>;
}

interface SignalEmitter {
  exitCode?: string | number | null;
  once(signal: "SIGINT" | "SIGTERM", listener: () => void): unknown;
  removeListener(signal: "SIGINT" | "SIGTERM", listener: () => void): unknown;
}

const browserToken = "roborev-e2e-browser-token";

export async function runBrowserTests(): Promise<number> {
  const webRoot = resolve(import.meta.dirname, "..");
  const repoRoot = dirname(webRoot);
  const scratch = await mkdtemp(join(tmpdir(), "roborev-web-e2e-"));
  const dataDir = join(scratch, "data");
  const homeDir = join(scratch, "home");
  const database = join(scratch, "reviews.db");
  const jobLogDir = join(dataDir, "logs", "jobs");
  const config = join(scratch, "config.toml");
  const binary = join(
    scratch,
    process.platform === "win32" ? "roborev.exe" : "roborev",
  );
  let assetsEmbedded = false;
  let daemon: ChildProcess | undefined;
  const commands = new Set<ChildProcess>();
  let cleanupPromise: Promise<void> | undefined;
  const cleanup = (): Promise<void> => {
    cleanupPromise ??= (async () => {
      await Promise.all(Array.from(commands, stop));
      if (daemon) {
        await stop(daemon);
      }
      if (assetsEmbedded) {
        await run("bun", ["run", "assets:restore"], webRoot);
        assetsEmbedded = false;
      }
      await rm(scratch, { recursive: true, force: true });
    })();
    return cleanupPromise;
  };
  const removeSignalHandlers = installCleanupSignalHandlers(cleanup);

  try {
    const sourceIndex = await readFile(
      join(repoRoot, "internal", "web", "dist", "index.html"),
      "utf8",
    );
    if (sourceIndex !== compilationStub) {
      throw new Error("embedded web assets must start at the compilation stub");
    }

    await Promise.all([
      mkdir(dataDir, { recursive: true, mode: 0o700 }),
      mkdir(homeDir, { recursive: true, mode: 0o700 }),
    ]);
    await writeFile(
      config,
      `max_workers = 0\n\n[web]\nenabled = true\nlisten = "127.0.0.1:0"\nauth_token = "${browserToken}"\n`,
      { mode: 0o600 },
    );

    await run(
      "go",
      ["run", "./internal/testutil/cmd/seed-web", "-out", database],
      repoRoot,
      {},
      true,
      commands,
    );
    await mkdir(jobLogDir, { recursive: true, mode: 0o700 });
    await writeFile(
      join(jobLogDir, "52.log"),
      "fixture review started\nstreamed analysis complete\n",
      { mode: 0o600 },
    );
    await run("bun", ["run", "build"], webRoot, {}, true, commands);
    assetsEmbedded = true;
    await run("bun", ["run", "assets:embed"], webRoot, {}, true, commands);
    await run(
      "go",
      ["build", "-o", binary, "./cmd/roborev"],
      repoRoot,
      {},
      true,
      commands,
    );
    await run("bun", ["run", "assets:restore"], webRoot, {}, true, commands);
    assetsEmbedded = false;

    daemon = spawn(
      binary,
      [
        "daemon",
        "run",
        "--db",
        database,
        "--config",
        config,
        "--addr",
        "127.0.0.1:0",
      ],
      {
        cwd: repoRoot,
        env: isolatedDaemonEnvironment(process.env, dataDir, homeDir),
        stdio: "inherit",
      },
    );
    const origin = await waitForBrowserOrigin(dataDir, daemon);
    return await run(
      "bunx",
      ["playwright", "test", "--config", "playwright.config.ts"],
      webRoot,
      { ROBOREV_E2E_ORIGIN: origin, ROBOREV_E2E_TOKEN: browserToken },
      false,
      commands,
    );
  } finally {
    removeSignalHandlers();
    await cleanup();
  }
}

export function installCleanupSignalHandlers(
  cleanup: () => Promise<void>,
  target: SignalEmitter = process,
): () => void {
  let handlingSignal = false;
  const handle = (exitCode: number) => () => {
    if (handlingSignal) return;
    handlingSignal = true;
    target.exitCode = exitCode;
    void cleanup();
  };
  const interrupt = handle(130);
  const terminate = handle(143);
  target.once("SIGINT", interrupt);
  target.once("SIGTERM", terminate);
  return () => {
    target.removeListener("SIGINT", interrupt);
    target.removeListener("SIGTERM", terminate);
  };
}

export function isolatedDaemonEnvironment(
  source: NodeJS.ProcessEnv,
  dataDir: string,
  homeDir: string,
): NodeJS.ProcessEnv {
  const environment: NodeJS.ProcessEnv = {
    ...source,
    HOME: homeDir,
    USERPROFILE: homeDir,
    XDG_CACHE_HOME: join(homeDir, ".cache"),
    XDG_CONFIG_HOME: join(homeDir, ".config"),
    ROBOREV_DATA_DIR: dataDir,
    ROBOREV_TELEMETRY_ENABLED: "false",
  };
  delete environment.ROBOREV_WEB_DEV_BACKEND;
  delete environment.ROBOREV_E2E_ORIGIN;
  delete environment.ROBOREV_E2E_TOKEN;
  return environment;
}

async function run(
  command: string,
  args: string[],
  cwd: string,
  extraEnvironment: Record<string, string> = {},
  rejectOnFailure = true,
  children?: Set<ChildProcess>,
): Promise<number> {
  const child = spawn(command, args, {
    cwd,
    env: { ...process.env, ...extraEnvironment },
    stdio: "inherit",
  });
  children?.add(child);
  const code = await childExit(child).finally(() => children?.delete(child));
  if (rejectOnFailure && code !== 0) {
    throw new Error(`${command} exited with status ${code}`);
  }
  return code;
}

async function waitForBrowserOrigin(
  dataDir: string,
  daemon: ChildProcess,
): Promise<string> {
  if (!daemon.pid) {
    throw new Error("browser test daemon did not start");
  }
  const runtime = join(dataDir, "runtime", `daemon.${daemon.pid}.json`);
  const deadline = Date.now() + 20_000;
  while (Date.now() < deadline) {
    if (daemon.exitCode !== null || daemon.signalCode !== null) {
      throw new Error(
        "browser test daemon exited before publishing runtime metadata",
      );
    }
    try {
      const record = JSON.parse(
        await readFile(runtime, "utf8"),
      ) as RuntimeRecord;
      const origin = record.metadata?.web_origin;
      if (origin) {
        const ping = await fetch(`${origin}/api/ping`);
        if (ping.ok) return origin;
      }
    } catch {
      // Runtime publication and listener startup are both asynchronous.
    }
    await delay(25);
  }
  throw new Error("timed out waiting for the browser test daemon");
}

async function stop(child: ChildProcess): Promise<void> {
  if (child.exitCode !== null || child.signalCode !== null) return;
  child.kill("SIGTERM");
  if (await exitsWithin(child, 3_000)) return;
  child.kill("SIGKILL");
  await exitsWithin(child, 1_000);
}

function exitsWithin(
  child: ChildProcess,
  milliseconds: number,
): Promise<boolean> {
  return Promise.race([
    childExit(child).then(() => true),
    delay(milliseconds).then(() => false),
  ]);
}

function childExit(child: ChildProcess): Promise<number> {
  return new Promise((resolveExit) => {
    if (child.exitCode !== null) {
      resolveExit(child.exitCode);
      return;
    }
    child.once("error", () => resolveExit(1));
    child.once("exit", (code) => resolveExit(code ?? 1));
  });
}

function delay(milliseconds: number): Promise<void> {
  return new Promise((resolveDelay) => setTimeout(resolveDelay, milliseconds));
}

const entrypoint = process.argv[1];
if (entrypoint && resolve(entrypoint) === fileURLToPath(import.meta.url)) {
  process.exitCode = await runBrowserTests();
}
