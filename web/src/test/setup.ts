import "@testing-library/jest-dom/vitest";

import { cleanup } from "@testing-library/svelte";
import { afterEach } from "vitest";

// Bun exposes an optional process-global localStorage whose undefined value can
// shadow jsdom's origin-scoped implementation. Supply a standards-shaped test
// store when that happens.
if (typeof window !== "undefined") {
  const localStorage = window.localStorage ?? createMemoryStorage();
  Object.defineProperty(window, "localStorage", {
    configurable: true,
    value: localStorage,
  });
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    value: localStorage,
  });
  Object.defineProperty(globalThis, "sessionStorage", {
    configurable: true,
    value: window.sessionStorage,
  });
}

afterEach(() => cleanup());

function createMemoryStorage(): Storage {
  const values = new Map<string, string>();
  return {
    get length() {
      return values.size;
    },
    clear() {
      values.clear();
    },
    getItem(key) {
      return values.get(key) ?? null;
    },
    key(index) {
      return [...values.keys()][index] ?? null;
    },
    removeItem(key) {
      values.delete(key);
    },
    setItem(key, value) {
      values.set(key, value);
    },
  };
}
