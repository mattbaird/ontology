<script lang="ts">
  let healthStatus = 'checking...';

  async function checkHealth() {
    try {
      const res = await fetch('/healthz');
      const data = await res.json();
      healthStatus = data.status === 'ok' ? 'Connected' : 'Error';
    } catch {
      healthStatus = 'Backend not reachable';
    }
  }

  checkHealth();
</script>

<div class="max-w-2xl mx-auto space-y-6">
  <h1 class="h1">Propeller POC</h1>
  <p class="text-surface-400">
    This proof-of-concept demonstrates the full ontology pipeline:
  </p>
  <div class="card p-4 space-y-2">
    <div class="flex items-center gap-3">
      <span class="font-mono text-sm bg-surface-200 dark:bg-surface-700 px-2 py-0.5 rounded">CUE Ontology</span>
      <span class="text-surface-400">&rarr;</span>
      <span class="font-mono text-sm bg-surface-200 dark:bg-surface-700 px-2 py-0.5 rounded">Ent Schema + DB</span>
      <span class="text-surface-400">&rarr;</span>
      <span class="font-mono text-sm bg-surface-200 dark:bg-surface-700 px-2 py-0.5 rounded">REST API</span>
      <span class="text-surface-400">&rarr;</span>
      <span class="font-mono text-sm bg-primary-500 text-white px-2 py-0.5 rounded">Svelte UI</span>
    </div>
  </div>

  <div class="card p-4">
    <h3 class="h4 mb-2">Backend Status</h3>
    <p class="text-sm">
      API Server: <span class="badge" class:variant-soft-success={healthStatus === 'Connected'} class:variant-soft-error={healthStatus !== 'Connected'}>
        {healthStatus}
      </span>
    </p>
    <p class="text-xs text-surface-400 mt-1">Proxied via Vite to localhost:8080</p>
  </div>

  <div class="card p-4">
    <h3 class="h4 mb-2">Entities</h3>
    <p class="text-sm text-surface-400 mb-3">Browse generated entity views:</p>
    <div class="grid grid-cols-2 gap-2">
      <a href="#/properties" class="btn variant-soft">Properties</a>
      <a href="#/spaces" class="btn variant-soft">Spaces</a>
      <a href="#/leases" class="btn variant-soft">Leases</a>
      <a href="#/persons" class="btn variant-soft">Persons</a>
      <a href="#/accounts" class="btn variant-soft">Accounts</a>
    </div>
  </div>

  <div class="card p-4">
    <h3 class="h4 mb-2">How to Demo</h3>
    <ol class="list-decimal list-inside text-sm space-y-1 text-surface-300">
      <li>Start the backend: <code class="code">go run ./cmd/server</code></li>
      <li>Start the frontend: <code class="code">cd frontend && npm run dev</code></li>
      <li>Create a property, then a lease against it</li>
      <li>Modify <code class="code">ontology/*.cue</code>, run <code class="code">make generate</code></li>
      <li>Watch changes propagate: DB schema &rarr; API &rarr; UI components</li>
    </ol>
  </div>
</div>
