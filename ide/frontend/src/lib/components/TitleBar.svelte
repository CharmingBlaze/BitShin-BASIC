<script lang="ts">
  import { Play, Square, Hammer, Minus, SquareCheck, X, Sparkles, Settings } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  function minimize() {
    window.runtime?.WindowMinimise?.();
  }

  function toggleMaximize() {
    window.runtime?.WindowToggleMaximise?.();
  }

  function closeWindow() {
    window.runtime?.Quit?.();
  }
</script>

<header class="titlebar flex items-center justify-between pl-3 pr-0 h-9 bg-slate-950 border-b border-slate-800/80 text-slate-300 text-xs select-none z-50" style="--wails-draggable: drag; -webkit-app-region: drag;">
  <!-- Left: Brand Logo & Title -->
  <div class="flex items-center gap-2.5 w-64 shrink-0" style="-webkit-app-region: no-drag;">
    <div class="flex items-center gap-2">
      <div class="w-5 h-5 rounded-md bg-gradient-to-tr from-sky-500 to-indigo-500 flex items-center justify-center text-white shadow-sm shadow-sky-500/20">
        <Sparkles size={11} />
      </div>
      <span class="font-semibold tracking-wide text-xs text-slate-100">BitShin BASIC</span>
      <span class="text-[9px] px-1.5 py-0.5 rounded bg-sky-500/10 border border-sky-500/30 text-sky-400 font-mono font-medium">IDE</span>
    </div>
  </div>

  <!-- Center: Active Document Breadcrumb -->
  <div class="flex-1 flex items-center justify-center gap-2 text-center text-slate-400 font-mono text-[11px] truncate px-4">
    {#if editorStore.activeTab}
      <span class="text-slate-200 font-medium">{editorStore.activeTab.name}</span>
      {#if editorStore.activeTab.isDirty}
        <span class="w-2 h-2 rounded-full bg-amber-400 inline-block animate-pulse" title="Unsaved changes"></span>
      {/if}
      <span class="text-slate-600 text-[10px] truncate max-w-sm">({editorStore.activeTab.path})</span>
    {:else}
      <span class="text-slate-500">BitShin BASIC Workspace</span>
    {/if}
  </div>

  <!-- Right: Quick Actions & Frameless Window Controls -->
  <div class="flex items-center gap-2 h-full" style="-webkit-app-region: no-drag;">
    <div class="flex items-center gap-1.5 mr-1">
      {#if editorStore.isRunning}
        <button
          onclick={() => editorStore.stopProgram()}
          class="flex items-center gap-1 px-2.5 py-1 rounded bg-rose-600 hover:bg-rose-500 text-white font-medium text-[11px] transition shadow-sm"
          title="Stop Running Program (Shift+F5)"
        >
          <Square size={11} fill="currentColor" />
          <span>Stop</span>
        </button>
      {:else}
        <button
          onclick={() => editorStore.runActiveProgram()}
          class="flex items-center gap-1.5 px-3 py-1 rounded bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-[11px] transition shadow-sm"
          title="Run Current Program (F5)"
        >
          <Play size={11} fill="currentColor" />
          <span>Run</span>
          <span class="text-[9px] px-1 py-0.2 rounded bg-emerald-700/60 font-mono text-emerald-100">F5</span>
        </button>
      {/if}

      <button
        onclick={() => editorStore.showBuildModal = true}
        class="flex items-center gap-1 px-2.5 py-1 rounded bg-slate-900 hover:bg-slate-800 text-slate-300 hover:text-white border border-slate-800 text-[11px] font-medium transition"
        title="Package Standalone (F7)"
      >
        <Hammer size={12} class="text-indigo-400" />
        <span>Build</span>
      </button>

      <button
        onclick={() => editorStore.showSettingsModal = true}
        class="p-1.5 rounded hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
        title="Settings"
      >
        <Settings size={13} />
      </button>
    </div>

    <!-- Windows native-styled controls -->
    <div class="window-controls flex items-center h-full border-l border-slate-800/80">
      <button
        onclick={minimize}
        class="h-full px-3.5 hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
        title="Minimize"
      >
        <Minus size={12} />
      </button>
      <button
        onclick={toggleMaximize}
        class="h-full px-3.5 hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
        title="Maximize"
      >
        <SquareCheck size={12} />
      </button>
      <button
        onclick={closeWindow}
        class="h-full px-3.5 hover:bg-rose-600 text-slate-400 hover:text-white transition"
        title="Close"
      >
        <X size={13} />
      </button>
    </div>
  </div>
</header>
