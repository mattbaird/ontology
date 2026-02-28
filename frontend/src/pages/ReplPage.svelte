<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte';

  // ── Types ──────────────────────────────────────────────────────────────────

  interface HistoryEntry {
    type: 'input' | 'meta' | 'table' | 'error' | 'info';
    content: string;
    rows?: Record<string, any>[];
    fields?: string[];
    elapsed?: string;
    total?: number;
  }

  interface CompletionItem {
    label: string;
    kind: string;
    detail?: string;
    insert_text?: string;
  }

  // ── State ──────────────────────────────────────────────────────────────────

  let ws: WebSocket | null = null;
  let connected = false;
  let sessionId = '';
  let inputValue = '';
  let history: HistoryEntry[] = [];
  let commandHistory: string[] = [];
  let historyIndex = -1;
  let inputEl: HTMLInputElement;
  let outputEl: HTMLDivElement;
  let completions: CompletionItem[] = [];
  let selectedCompletion = 0;
  let showCompletions = false;

  // Accumulate rows for the current query
  let pendingMeta: { entity: string; fields: string[]; total: number } | null = null;
  let pendingRows: Record<string, any>[] = [];
  let currentRequestId = 0;

  // ── WebSocket ──────────────────────────────────────────────────────────────

  function connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/api/repl/ws`);

    ws.onopen = () => {
      connected = true;
      history = [...history, { type: 'info', content: 'Connected to Propeller REPL.' }];
    };

    ws.onclose = () => {
      connected = false;
      history = [...history, { type: 'info', content: 'Disconnected.' }];
    };

    ws.onerror = () => {
      connected = false;
      history = [...history, { type: 'error', content: 'WebSocket connection failed. Is the backend running?' }];
    };

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      handleServerMessage(msg);
    };
  }

  function handleServerMessage(msg: { type: string; request_id?: string; data?: any }) {
    switch (msg.type) {
      case 'session':
        sessionId = msg.data?.session_id || '';
        break;

      case 'meta':
        pendingMeta = msg.data;
        pendingRows = [];
        break;

      case 'rows':
        if (msg.data?.rows) {
          for (const row of msg.data.rows) {
            pendingRows.push(typeof row === 'string' ? JSON.parse(row) : row);
          }
        }
        break;

      case 'done': {
        const elapsed = msg.data?.elapsed || '';
        const total = msg.data?.total ?? pendingRows.length;
        if (pendingMeta || pendingRows.length > 0) {
          history = [...history, {
            type: 'table',
            content: '',
            rows: pendingRows,
            fields: pendingMeta?.fields || (pendingRows.length > 0 ? Object.keys(pendingRows[0]) : []),
            elapsed,
            total,
          }];
        } else {
          history = [...history, { type: 'info', content: `Done. ${total} result(s) in ${elapsed}` }];
        }
        pendingMeta = null;
        pendingRows = [];
        scrollToBottom();
        break;
      }

      case 'error':
        history = [...history, { type: 'error', content: msg.data?.message || 'Unknown error' }];
        pendingMeta = null;
        pendingRows = [];
        scrollToBottom();
        break;

      case 'completions':
        if (msg.data?.items?.length) {
          completions = msg.data.items;
          selectedCompletion = 0;
          showCompletions = true;
        } else {
          showCompletions = false;
        }
        break;

      case 'pong':
        break;
    }
  }

  // ── Commands ───────────────────────────────────────────────────────────────

  function execute(pql: string) {
    const trimmed = pql.trim();
    if (!trimmed) return;

    // Handle local :clear
    if (trimmed === ':clear') {
      history = [];
      return;
    }

    history = [...history, { type: 'input', content: trimmed }];
    commandHistory = [trimmed, ...commandHistory];
    historyIndex = -1;

    if (!ws || ws.readyState !== WebSocket.OPEN) {
      history = [...history, { type: 'error', content: 'Not connected.' }];
      return;
    }

    currentRequestId++;
    const id = String(currentRequestId);
    ws.send(JSON.stringify({
      type: 'execute',
      id,
      data: { pql: trimmed },
    }));

    scrollToBottom();
  }

  function requestCompletions(pql: string, cursor: number) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({
      type: 'autocomplete',
      id: 'ac',
      data: { pql, cursor },
    }));
  }

  // ── Input handling ─────────────────────────────────────────────────────────

  let completionTimeout: ReturnType<typeof setTimeout>;

  function handleKeydown(e: KeyboardEvent) {
    // Completions navigation
    if (showCompletions) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        selectedCompletion = Math.min(selectedCompletion + 1, completions.length - 1);
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        selectedCompletion = Math.max(selectedCompletion - 1, 0);
        return;
      }
      if (e.key === 'Tab') {
        if (completions[selectedCompletion]) {
          e.preventDefault();
          applyCompletion(completions[selectedCompletion]);
          return;
        }
      }
      // Enter dismisses completions and executes (handled below)
      if (e.key === 'Enter') {
        showCompletions = false;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        showCompletions = false;
        return;
      }
    }

    // Execute on Enter (when completions not showing)
    if (e.key === 'Enter') {
      e.preventDefault();
      showCompletions = false;
      execute(inputValue);
      inputValue = '';
      return;
    }

    // Command history
    if (e.key === 'ArrowUp' && !showCompletions) {
      e.preventDefault();
      if (historyIndex < commandHistory.length - 1) {
        historyIndex++;
        inputValue = commandHistory[historyIndex];
      }
      return;
    }
    if (e.key === 'ArrowDown' && !showCompletions) {
      e.preventDefault();
      if (historyIndex > 0) {
        historyIndex--;
        inputValue = commandHistory[historyIndex];
      } else {
        historyIndex = -1;
        inputValue = '';
      }
      return;
    }

    // Tab triggers completion request
    if (e.key === 'Tab' && !showCompletions) {
      e.preventDefault();
      requestCompletions(inputValue, inputValue.length);
      return;
    }
  }

  function handleInput() {
    showCompletions = false;
    clearTimeout(completionTimeout);
    if (inputValue.length > 0) {
      completionTimeout = setTimeout(() => {
        requestCompletions(inputValue, inputValue.length);
      }, 150);
    }
  }

  function applyCompletion(item: CompletionItem) {
    const text = item.insert_text || item.label;
    // Replace the last partial token
    const parts = inputValue.split(/\s+/);
    parts[parts.length - 1] = text;
    inputValue = parts.join(' ') + ' ';
    showCompletions = false;
    inputEl?.focus();
  }

  // ── Utilities ──────────────────────────────────────────────────────────────

  async function scrollToBottom() {
    await tick();
    if (outputEl) {
      outputEl.scrollTop = outputEl.scrollHeight;
    }
  }

  function formatValue(val: any): string {
    if (val === null || val === undefined) return '—';
    if (typeof val === 'object') return JSON.stringify(val);
    return String(val);
  }

  function truncate(s: string, max: number): string {
    if (s.length <= max) return s;
    return s.slice(0, max - 1) + '…';
  }

  // ── Lifecycle ──────────────────────────────────────────────────────────────

  onMount(() => {
    connect();
    inputEl?.focus();
  });

  onDestroy(() => {
    clearTimeout(completionTimeout);
    if (ws) {
      ws.close();
      ws = null;
    }
  });
</script>

<div class="flex flex-col h-full">
  <!-- Header -->
  <div class="flex items-center justify-between mb-3">
    <div class="flex items-center gap-3">
      <h1 class="h2">REPL</h1>
      <span class="badge text-xs font-mono" class:variant-soft-success={connected} class:variant-soft-error={!connected}>
        {connected ? 'connected' : 'disconnected'}
      </span>
      {#if sessionId}
        <span class="text-xs text-surface-500 font-mono">{sessionId.slice(0, 8)}</span>
      {/if}
    </div>
    <div class="flex items-center gap-2">
      <button class="btn btn-sm variant-soft" on:click={() => { history = []; }}>Clear</button>
      {#if !connected}
        <button class="btn btn-sm variant-filled-primary" on:click={connect}>Reconnect</button>
      {/if}
    </div>
  </div>

  <!-- Output area -->
  <div
    bind:this={outputEl}
    class="flex-1 overflow-y-auto font-mono text-sm bg-surface-900 rounded-t-lg p-4 space-y-3 min-h-0"
    on:click={() => inputEl?.focus()}
  >
    {#if history.length === 0}
      <div class="text-surface-500">
        Type a PQL query and press Enter. Try: <span class="text-primary-400">find lease limit 5</span>
        <br/>
        Mutations: <span class="text-primary-400">create</span>, <span class="text-primary-400">update</span>, <span class="text-primary-400">delete</span>.
        Press Tab for autocomplete. Type <span class="text-primary-400">:help</span> for commands.
      </div>
    {/if}

    {#each history as entry}
      {#if entry.type === 'input'}
        <div class="flex gap-2">
          <span class="text-primary-400 select-none shrink-0">pql&gt;</span>
          <span class="text-white">{entry.content}</span>
        </div>

      {:else if entry.type === 'error'}
        <div class="text-error-400">{entry.content}</div>

      {:else if entry.type === 'info'}
        <div class="text-surface-400">{entry.content}</div>

      {:else if entry.type === 'table' && entry.rows && entry.rows.length > 0}
        <div class="overflow-x-auto">
          <table class="w-full text-xs">
            <thead>
              <tr class="text-surface-400 border-b border-surface-700">
                {#each entry.fields || [] as field}
                  <th class="text-left px-2 py-1 font-medium">{field}</th>
                {/each}
              </tr>
            </thead>
            <tbody>
              {#each entry.rows as row}
                <tr class="border-b border-surface-800 hover:bg-surface-800/50">
                  {#each entry.fields || [] as field}
                    <td class="px-2 py-1 text-surface-200 max-w-xs" title={formatValue(row[field])}>
                      {truncate(formatValue(row[field]), 60)}
                    </td>
                  {/each}
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        {#if entry.total !== undefined || entry.elapsed}
          <div class="text-xs text-surface-500">
            {entry.total} result{entry.total === 1 ? '' : 's'}
            {#if entry.elapsed} in {entry.elapsed}{/if}
          </div>
        {/if}

      {:else if entry.type === 'table'}
        <div class="text-surface-500">No results.</div>
        {#if entry.elapsed}
          <div class="text-xs text-surface-500">0 results in {entry.elapsed}</div>
        {/if}
      {/if}
    {/each}
  </div>

  <!-- Input area -->
  <div class="relative">
    {#if showCompletions && completions.length > 0}
      <div class="absolute bottom-full left-0 w-full max-w-lg bg-surface-800 border border-surface-700 rounded-t-lg shadow-lg z-10 max-h-48 overflow-y-auto">
        {#each completions as item, i}
          <button
            class="w-full text-left px-3 py-1.5 text-sm font-mono flex items-center gap-3 hover:bg-surface-700"
            class:bg-surface-700={i === selectedCompletion}
            on:mousedown|preventDefault={() => applyCompletion(item)}
          >
            <span class="text-xs text-surface-500 w-12 shrink-0">{item.kind}</span>
            <span class="text-white">{item.label}</span>
            {#if item.detail}
              <span class="text-xs text-surface-500 ml-auto">{item.detail}</span>
            {/if}
          </button>
        {/each}
      </div>
    {/if}

    <div class="flex items-center bg-surface-800 rounded-b-lg border-t border-surface-700 px-4 py-2 gap-2">
      <span class="text-primary-400 font-mono text-sm select-none shrink-0">pql&gt;</span>
      <input
        bind:this={inputEl}
        bind:value={inputValue}
        on:keydown={handleKeydown}
        on:input={handleInput}
        type="text"
        class="flex-1 bg-transparent border-none outline-none text-white font-mono text-sm placeholder-surface-600"
        placeholder="find lease where status = &quot;active&quot; limit 10"
        spellcheck="false"
        autocomplete="off"
      />
    </div>
  </div>
</div>
