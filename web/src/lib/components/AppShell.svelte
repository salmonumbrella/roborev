<script lang="ts">
  import { Effect } from "effect";
  import { onDestroy } from "svelte";

  import { createRoborevClient } from "../api/client";
  import { createRouter } from "../router/router.svelte";
  import { setAppRuntime, setRoborevClient } from "../runtime/context";
  import { makeAppRuntime } from "../runtime/runtime";
  import { createReviewStores } from "../stores/composition.svelte";
  import { provideReviewStores } from "../stores/context";
  import AnalyticsPlaceholder from "../views/AnalyticsPlaceholder.svelte";
  import ReviewsView from "../views/ReviewsView.svelte";

  interface Props {
    ondisconnect: () => void | Promise<void>;
  }

  const { ondisconnect }: Props = $props();
  const runtime = makeAppRuntime();
  const client = createRoborevClient("/");
  const router = createRouter();
  const route = $derived(router.getRoute());
  const stores = createReviewStores({
    runtime,
    client,
    navigate: router.navigateToReview,
  });
  setAppRuntime(runtime);
  setRoborevClient(client);
  provideReviewStores(stores);

  const polling = runtime.runCommand(stores.roborevDaemon.pollingEffect, {
    operation: "poll Roborev daemon status",
    safeContext: {},
    onFailure: () => {},
  });

  onDestroy(() => {
    polling.interrupt();
    stores.roborevJobs.dispose();
    router.dispose();
    void Effect.runPromise(runtime.disposeEffect);
  });

  function navigateReviews(event: MouseEvent): void {
    event.preventDefault();
    router.navigateToReview();
  }

  function navigateAnalytics(event: MouseEvent): void {
    event.preventDefault();
    router.navigateToAnalytics();
  }
</script>

<div class="app-shell">
  <header class="app-header">
    <a class="brand" href="/reviews" onclick={navigateReviews}>Roborev</a>
    <nav aria-label="Application">
      <a
        href="/reviews"
        class:active={route.page === "reviews"}
        onclick={navigateReviews}>Reviews</a
      >
      <a
        href="/analytics"
        class:active={route.page === "analytics"}
        onclick={navigateAnalytics}>Analytics</a
      >
    </nav>
    <button class="disconnect" type="button" onclick={ondisconnect}>
      Disconnect
    </button>
  </header>

  <div class="app-content">
    {#if route.page === "analytics"}
      <AnalyticsPlaceholder />
    {:else}
      <ReviewsView jobId={route.jobId} />
    {/if}
  </div>
</div>

<style>
  .app-shell {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    background: var(--bg-primary);
    color: var(--text-primary);
  }

  .app-header {
    display: flex;
    min-height: 48px;
    flex: 0 0 auto;
    align-items: center;
    gap: 20px;
    padding: 0 16px;
    border-bottom: 1px solid var(--border-default);
    background: var(--bg-surface);
  }

  .brand {
    color: var(--text-primary);
    font-weight: 700;
  }

  nav {
    display: flex;
    align-self: stretch;
    gap: 4px;
  }

  nav a {
    display: inline-flex;
    align-items: center;
    padding: 0 12px;
    border-bottom: 2px solid transparent;
    color: var(--text-secondary);
  }

  nav a.active {
    border-bottom-color: var(--accent-blue);
    color: var(--text-primary);
  }

  .disconnect {
    margin-left: auto;
    padding: 5px 10px;
    border: 1px solid var(--border-default);
    border-radius: var(--radius-sm);
  }

  .disconnect:hover {
    background: var(--bg-surface-hover);
  }

  .app-content {
    display: flex;
    min-height: 0;
    flex: 1;
  }
</style>
