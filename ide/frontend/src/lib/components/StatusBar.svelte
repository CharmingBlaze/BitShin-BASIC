<script lang="ts">
  import { Cpu, Sparkles, Check, Terminal } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';
</script>

<footer class="statusbar flex items-center justify-between px-3 h-6 bg-slate-950 border-t border-slate-800 text-[11px] font-mono text-slate-400 select-none z-30">
  <!-- Left info -->
  <div class="flex items-center gap-3">
    <div class="flex items-center gap-1.5 text-sky-400 font-semibold">
      <Sparkles size={11} />
      <span>BitShin</span>
    </div>

    <div class="w-px h-3 bg-slate-800"></div>

    <div class="flex items-center gap-1 text-slate-400">
      <span>Ln {editorStore.cursorPos.line}, Col {editorStore.cursorPos.col}</span>
    </div>

    {#if editorStore.activeTab}
      <div class="text-slate-500">
        {editorStore.activeTab.content.length} chars
      </div>
    {/if}

    <div class="w-px h-3 bg-slate-800"></div>

    <button
      onclick={() => editorStore.toggleOutputPanel()}
      class="flex items-center gap-1 hover:text-slate-200 transition cursor-pointer {editorStore.isOutputCollapsed ? 'text-slate-500' : 'text-sky-400'}"
      title="Toggle Console (Ctrl+`)"
    >
      <Terminal size={11} />
      <span>Console</span>
    </button>
  </div>

  <!-- Right status -->
  <div class="flex items-center gap-3">
    {#if editorStore.isRunning}
      <div class="flex items-center gap-1 text-amber-400 animate-pulse font-medium">
        <Cpu size={11} />
        <span>Executing (OpenGL 3.3 / Ebiten)</span>
      </div>
    {:else if editorStore.exitCode === 0}
      <div class="flex items-center gap-1 text-emerald-400">
        <Check size={11} />
        <span>Ready</span>
      </div>
    {/if}

    <div class="w-px h-3 bg-slate-800"></div>

    <span class="text-slate-400">UTF-8</span>
    <span class="text-slate-400">Spaces: 4</span>
    <span class="text-slate-500">BitShin BASIC</span>
  </div>
</footer>
