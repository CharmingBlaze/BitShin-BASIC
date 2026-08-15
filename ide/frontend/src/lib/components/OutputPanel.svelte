<script lang="ts">
  import {
    Terminal, Trash2, ArrowDownCircle, CheckCircle2,
    XCircle, Loader2, ChevronDown, ChevronUp, X
  } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  let autoScroll = $state(true);
  let logContainer: HTMLElement | null = $state(null);

  $effect(() => {
    if (autoScroll && logContainer && editorStore.consoleLogs.length) {
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

<div class="output-panel flex flex-col h-full bg-slate-950 border-t border-slate-800 text-xs font-mono select-text">
  <!-- Console Header Bar -->
  <div
    ondblclick={() => editorStore.toggleOutputPanel()}
    class="flex items-center justify-between px-3 h-8 bg-slate-900 border-b border-slate-800 text-slate-400 select-none cursor-pointer"
    title="Double-click to expand / collapse console"
  >
    <div class="flex items-center gap-2.5">
      <div class="flex items-center gap-1.5 text-slate-200 font-semibold text-[11px]">
        <Terminal size={13} class="text-sky-400" />
        <span>Execution Console</span>
      </div>

      {#if editorStore.isRunning}
        <div class="flex items-center gap-1 px-2 py-0.5 rounded bg-amber-500/10 border border-amber-500/30 text-amber-400 text-[10px]">
          <Loader2 size={10} class="animate-spin" />
          <span>Running...</span>
        </div>
      {:else if editorStore.exitCode !== null}
        {#if editorStore.exitCode === 0}
          <div class="flex items-center gap-1 px-2 py-0.5 rounded bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 text-[10px]">
            <CheckCircle2 size={10} />
            <span>Finished (0)</span>
          </div>
        {:else}
          <div class="flex items-center gap-1 px-2 py-0.5 rounded bg-rose-500/10 border border-rose-500/30 text-rose-400 text-[10px]">
            <XCircle size={10} />
            <span>Exited ({editorStore.exitCode})</span>
          </div>
        {/if}
      {/if}
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-1" onclick={(e) => e.stopPropagation()}>
      <button
        onclick={() => autoScroll = !autoScroll}
        class="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] transition {autoScroll ? 'bg-sky-500/10 text-sky-400 border border-sky-500/30' : 'text-slate-500 hover:text-slate-300'}"
        title="Toggle Auto-Scroll"
      >
        <ArrowDownCircle size={11} />
        <span>Auto-scroll</span>
      </button>

      <button
        onclick={() => editorStore.clearLogs()}
        class="p-1 rounded hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
        title="Clear Console Output"
      >
        <Trash2 size={12} />
      </button>

      <div class="w-px h-3.5 bg-slate-800 mx-0.5"></div>

      <!-- Collapse / Expand Button -->
      <button
        onclick={() => editorStore.toggleOutputPanel()}
        class="p-1 rounded hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
        title="{editorStore.isOutputCollapsed ? 'Expand Console' : 'Collapse Console'}"
      >
        {#if editorStore.isOutputCollapsed}
          <ChevronUp size={13} />
        {:else}
          <ChevronDown size={13} />
        {/if}
      </button>
    </div>
  </div>

  <!-- Logs Stream -->
  {#if !editorStore.isOutputCollapsed}
    <div
      bind:this={logContainer}
      class="flex-1 overflow-y-auto p-3 space-y-1 scrollbar-thin font-mono text-[11px] leading-relaxed select-text"
    >
      {#if editorStore.consoleLogs.length === 0}
        <div class="text-slate-600 italic select-none py-1">
          Ready. Press F5 or click Run to execute the current script.
        </div>
      {:else}
        {#each editorStore.consoleLogs as log}
          <div class="flex items-start gap-2 group">
            <span class="text-slate-600 text-[10px] select-none shrink-0 pt-0.5">[{log.time}]</span>

            {#if log.type === 'stderr'}
              <div class="flex-1 text-rose-400 break-all">
                <span>{log.text}</span>
                {#if log.line}
                  <button
                    onclick={() => jumpToLocation(log.line)}
                    class="ml-2 px-1.5 py-0.5 rounded bg-rose-950 border border-rose-800 text-rose-300 text-[10px] hover:underline cursor-pointer select-none"
                  >
                    Jump to line {log.line}
                  </button>
                {/if}
              </div>
            {:else if log.type === 'system'}
              <div class="flex-1 text-sky-400 font-medium">{log.text}</div>
            {:else if log.type === 'success'}
              <div class="flex-1 text-emerald-400 font-medium">{log.text}</div>
            {:else}
              <div class="flex-1 text-slate-300 break-all">{log.text}</div>
            {/if}
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>
