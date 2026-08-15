<script lang="ts">
  import { Play, Square, Hammer, Minus, SquareCheck, X, Sparkles, FolderOpen, Save, Settings } from 'lucide-svelte';
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

<header class="titlebar select-none flex items-center justify-between px-3 h-10 bg-slate-950/90 border-b border-slate-800/80 text-slate-300 text-xs backdrop-blur-md z-50">
  <!-- Left: Logo & Project Info -->
  <div class="flex items-center gap-3 w-72">
    <div class="flex items-center gap-2 font-bold tracking-wide text-sky-400">
      <div class="w-5 h-5 rounded bg-gradient-to-tr from-sky-500 to-indigo-500 flex items-center justify-center text-white shadow-sm shadow-sky-500/30">
        <Sparkles size={12} />
      </div>
      <span>BitShin BASIC</span>
      <span class="text-[10px] px-1.5 py-0.5 rounded bg-sky-950 border border-sky-800/60 text-sky-400 font-mono">IDE</span>
    </div>
  </div>

  <!-- Center: Active Document Title & Status -->
  <div class="flex-1 flex items-center justify-center gap-2 text-center text-slate-400 font-mono text-[11px] truncate px-4">
    {#if editorStore.activeTab}
      <span class="text-slate-200 font-medium">{editorStore.activeTab.name}</span>
      {#if editorStore.activeTab.isDirty}
        <span class="w-2 h-2 rounded-full bg-amber-400 inline-block animate-pulse" title="Unsaved changes"></span>
      {/if}
      <span class="text-slate-600 text-[10px] truncate max-w-sm">({editorStore.activeTab.path})</span>
    {:else}
      <span>No file open</span>
    {/if}
  </div>

  <!-- Right: Run / Stop quick triggers & Window buttons -->
  <div class="flex items-center gap-2">
    <div class="flex items-center gap-1.5 mr-2">
      {#if editorStore.isRunning}
        <button
          onclick={() => editorStore.stopProgram()}
          class="flex items-center gap-1 px-2.5 py-1 rounded bg-rose-600/90 hover:bg-rose-500 text-white font-medium shadow-sm shadow-rose-950 transition-all text-[11px]"
          title="Stop Running Program (Shift+F5)"
        >
          <Square size={12} fill="currentColor" />
          <span>Stop</span>
        </button>
      {:else}
        <button
          onclick={() => editorStore.runActiveProgram()}
          class="flex items-center gap-1 px-3 py-1 rounded bg-emerald-600 hover:bg-emerald-500 text-white font-medium shadow-sm shadow-emerald-950 transition-all text-[11px] hover:scale-105 active:scale-95"
          title="Run Current Program (F5)"
        >
          <Play size={12} fill="currentColor" />
          <span>Run (F5)</span>
        </button>
      {/if}

      <button
        onclick={() => editorStore.showBuildModal = true}
        class="flex items-center gap-1 px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-all text-[11px]"
        title="Build Standalone Package (F7)"
      >
        <Hammer size={12} />
        <span>Build</span>
      </button>

      <button
        onclick={() => editorStore.showSettingsModal = true}
        class="p-1 rounded hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
        title="IDE Settings"
      >
        <Settings size={14} />
      </button>
    </div>

    <!-- Window controls -->
    <div class="flex items-center border-l border-slate-800 pl-2">
      <button
        onclick={minimize}
        class="p-1.5 hover:bg-slate-800 text-slate-400 hover:text-slate-200 rounded transition"
        title="Minimize"
      >
        <Minus size={13} />
      </button>
      <button
        onclick={toggleMaximize}
        class="p-1.5 hover:bg-slate-800 text-slate-400 hover:text-slate-200 rounded transition"
        title="Maximize"
      >
        <SquareCheck size={13} />
      </button>
      <button
        onclick={closeWindow}
        class="p-1.5 hover:bg-rose-600 text-slate-400 hover:text-white rounded transition"
        title="Close"
      >
        <X size={13} />
      </button>
    </div>
  </div>
</header>
