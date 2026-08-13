<script lang="ts">
  import { onMount, tick } from "svelte";

  import { bootstrapSession, login, logout } from "./lib/api/session";

  type ViewState = "checking" | "login" | "authenticated" | "error";

  let view: ViewState = "checking";
  let token = "";
  let errorMessage = "";
  let connecting = false;

  onMount(() => {
    void checkSession();
  });

  async function checkSession(): Promise<void> {
    view = "checking";
    const result = await bootstrapSession();
    applySessionResult(result);
  }

  async function connect(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    connecting = true;
    let result: Awaited<ReturnType<typeof login>>;
    try {
      result = await login(token);
    } finally {
      token = "";
      connecting = false;
      await tick();
    }
    applySessionResult(result);
  }

  async function disconnect(): Promise<void> {
    try {
      await logout();
      await checkSession();
    } catch (error) {
      errorMessage =
        error instanceof Error ? error.message : "Could not end the session";
      view = "error";
    }
  }

  function applySessionResult(
    result: Awaited<ReturnType<typeof bootstrapSession>>,
  ): void {
    switch (result.state) {
      case "authenticated":
        view = "authenticated";
        break;
      case "login-required":
        view = "login";
        break;
      case "error":
        errorMessage = result.message;
        view = "error";
        break;
    }
  }
</script>

<main>
  {#if view === "checking"}
    <section class="card status" aria-live="polite">
      <p class="eyebrow">Roborev</p>
      <p>Checking browser session…</p>
    </section>
  {:else if view === "login"}
    <section class="card login-card">
      <p class="eyebrow">Roborev web</p>
      <h1>Connect to Roborev</h1>
      <p class="muted">
        Enter the browser access token configured for this daemon. It is
        exchanged once and is not retained by the application.
      </p>
      <form onsubmit={connect}>
        <label for="daemon-token">Daemon token</label>
        <input
          id="daemon-token"
          type="password"
          autocomplete="current-password"
          bind:value={token}
          required
        />
        <button type="submit" disabled={connecting}>
          {connecting ? "Connecting…" : "Connect"}
        </button>
      </form>
    </section>
  {:else if view === "authenticated"}
    <section class="card foundation">
      <header>
        <div>
          <p class="eyebrow">Browser foundation</p>
          <h1>Roborev</h1>
        </div>
        <button class="secondary" type="button" onclick={disconnect}>
          Disconnect
        </button>
      </header>
      <p class="muted">
        The native review workspace and project analytics will land in the next
        migration stages.
      </p>
      <nav aria-label="Application">
        <a href="/reviews">Reviews</a>
        <a href="/analytics">Analytics</a>
      </nav>
    </section>
  {:else}
    <section class="card status" role="alert">
      <p class="eyebrow">Connection problem</p>
      <h1>Roborev is unavailable</h1>
      <p>{errorMessage}</p>
      <button type="button" onclick={checkSession}>Retry</button>
    </section>
  {/if}
</main>
