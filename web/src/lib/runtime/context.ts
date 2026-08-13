import { getContext } from "svelte";

import type { RoborevClient } from "../api/client";
import type { OwnedAppRuntime } from "./runtime";

export const APP_RUNTIME_KEY = Symbol("roborev-app-runtime");
export const ROBOREV_CLIENT_KEY = Symbol("roborev-client");

export function getAppRuntime(): OwnedAppRuntime {
  return getContext(APP_RUNTIME_KEY);
}

export function getRoborevClient(): RoborevClient {
  return getContext(ROBOREV_CLIENT_KEY);
}
