<script lang="ts">
  import { Trash2, ArrowDownCircle, ChevronDown, ChevronUp } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  let autoScroll = $state(true);
  let logContainer: HTMLElement | null = $state(null);

  let problems = $derived(editorStore.consoleLogs.filter(l => l.type === 'stderr' || l.type === 'error'));
  let visibleLogs = $derived(editorStore.panelTab === 'problems' ? problems : editorStore.consoleLogs);

  $effect(() => {
    if (autoScroll && logContainer && visibleLogs.length) {
      logContainer.scrollTop = logContainer.scrollHeight;
    }
  });

  function jumpToLocation(line?: number) {
    if (!line) return;
    const editor = (window as any).__monacoEditor;
    if (editor) {
      editor.revealLineInCenter(line);
      editor.setPosition({ lineNumber: line, column: 1 });
      editor.focus();
    }
  }
</script>

<div class="ide-panel">
  <div
    class="ide-panel-bar"
    ondblclick={() => editorStore.toggleOutputPanel()}
    title="Double-click to expand or collapse"
  >
    <div class="flex items-center gap-2">
      <div class="ide-panel-tabs">
        <button
          class="ide-panel-tab {editorStore.panelTab === 'output' ? 'active' : ''}"
          onclick={() => { editorStore.panelTab = 'output'; editorStore.isOutputCollapsed = false; }}
        >Output</button>
        <button
          class="ide-panel-tab {editorStore.panelTab === 'problems' ? 'active' : ''}"
          onclick={() => { editorStore.panelTab = 'problems'; editorStore.isOutputCollapsed = false; }}
        >Problems{problems.length ? ` (${problems.length})` : ''}</button>
      </div>

      {#if editorStore.isRunning}
        <span class="ide-badge run">Running</span>
      {:else if editorStore.exitCode !== null}
        {#if editorStore.exitCode === 0}
          <span class="ide-badge ok">Finished 0</span>
        {:else}
          <span class="ide-badge err">Exited {editorStore.exitCode}</span>
        {/if}
      {/if}
    </div>

    <div class="flex items-center gap-1" onclick={(e) => e.stopPropagation()}>
      <button
        class="ide-iconbtn"
        style={autoScroll ? 'color: var(--accent);' : ''}
        onclick={() => autoScroll = !autoScroll}
        title="Auto-scroll"
      >
        <ArrowDownCircle size={13} />
      </button>
      <button class="ide-iconbtn" onclick={() => editorStore.clearLogs()} title="Clear">
        <Trash2 size={13} />
      </button>
      <button class="ide-iconbtn" onclick={() => editorStore.toggleOutputPanel()} title={editorStore.isOutputCollapsed ? 'Expand' : 'Collapse'}>
        {#if editorStore.isOutputCollapsed}
          <ChevronUp size={13} />
        {:else}
          <ChevronDown size={13} />
        {/if}
      </button>
    </div>
  </div>

  {#if !editorStore.isOutputCollapsed}
    <div bind:this={logContainer} class="ide-log">
      {#if visibleLogs.length === 0}
        <div style="color: var(--text-dim);">
          {#if editorStore.panelTab === 'problems'}
            No problems reported.
          {:else}
            Ready. Press F5 or Run to execute with bs.exe.
          {/if}
        </div>
      {:else}
        {#each visibleLogs as log}
          <div class="ide-log-row">
            <span class="ide-log-time">[{log.time}]</span>
            <div class="ide-log-{log.type}" style="flex:1;word-break:break-all;">
              {log.text}
              {#if log.line}
                <button class="ide-jump" onclick={() => jumpToLocation(log.line)}>line {log.line}</button>
              {/if}
            </div>
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>
