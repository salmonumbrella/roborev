import { Effect, Fiber } from "effect";
import { TestClock } from "effect/testing";
import { afterEach, describe, expect, it, vi } from "@effect/vitest";

import type { OwnedAppRuntime } from "../../runtime/runtime";
import { makeTestAppRuntime } from "../../testing/runtime";
import { createDaemonStore } from "./daemon.svelte";

let runtime: OwnedAppRuntime | undefined;

afterEach(async () => {
  if (runtime !== undefined) {
    await Effect.runPromise(runtime.disposeEffect);
  }
  runtime = undefined;
});

describe("native daemon store", () => {
  it("publishes direct daemon status as one authority", async () => {
    runtime = makeTestAppRuntime();
    const get = vi.fn().mockResolvedValue({
      data: {
        active_workers: 1,
        canceled_jobs: 2,
        completed_jobs: 3,
        failed_jobs: 4,
        max_workers: 5,
        queued_jobs: 6,
        running_jobs: 7,
        version: "test-version",
      },
    });
    const store = createDaemonStore({ client: { GET: get } as never, runtime });

    store.checkHealth();

    await vi.waitFor(() => expect(store.isAvailable()).toBe(true));
    expect(store.getVersion()).toBe("test-version");
    expect(store.getEndpoint()).toBe(window.location.origin);
    expect(store.getQueuedJobs()).toBe(6);
    expect(store.getRunningJobs()).toBe(7);
    expect(store.getCompletedJobs()).toBe(3);
    expect(store.getFailedJobs()).toBe(4);
    expect(store.getCanceledJobs()).toBe(2);
    expect(store.getActiveWorkers()).toBe(1);
    expect(store.getMaxWorkers()).toBe(5);
    expect(store.getWasEverAvailable()).toBe(true);
  });

  it("clears stale counts when the daemon becomes unavailable", async () => {
    runtime = makeTestAppRuntime();
    const get = vi
      .fn()
      .mockResolvedValueOnce({
        data: {
          active_workers: 1,
          canceled_jobs: 2,
          completed_jobs: 3,
          failed_jobs: 4,
          max_workers: 5,
          queued_jobs: 6,
          running_jobs: 7,
          version: "test-version",
        },
      })
      .mockRejectedValueOnce(new TypeError("offline"));
    const store = createDaemonStore({ client: { GET: get } as never, runtime });

    store.checkHealth();
    await vi.waitFor(() => expect(store.isAvailable()).toBe(true));
    store.checkHealth();
    await vi.waitFor(() => expect(store.isAvailable()).toBe(false));

    expect(store.getQueuedJobs()).toBe(0);
    expect(store.getRunningJobs()).toBe(0);
    expect(store.getActiveWorkers()).toBe(0);
    expect(store.getWasEverAvailable()).toBe(true);
  });

  it.effect("retries after one second while unavailable", () =>
    Effect.gen(function* () {
      const get = vi.fn().mockRejectedValue(new TypeError("offline"));
      const testRuntime = makeTestAppRuntime();
      const store = createDaemonStore({
        client: { GET: get } as never,
        runtime: testRuntime,
      });
      const polling = yield* Effect.forkChild(store.pollingEffect);

      yield* Effect.yieldNow;
      expect(get).toHaveBeenCalledTimes(1);
      yield* TestClock.adjust("999 millis");
      expect(get).toHaveBeenCalledTimes(1);
      yield* TestClock.adjust("1 millis");
      expect(get).toHaveBeenCalledTimes(2);

      yield* Fiber.interrupt(polling);
      yield* testRuntime.disposeEffect;
    }),
  );
});
