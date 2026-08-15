<script lang="ts">
  import {
    FilePlus, FolderOpen, Save, Play, Square, Hammer,
    FolderTree, BookOpen, Code2, Sparkles, Terminal,
    Palette, RefreshCw
  } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  const themes = [
    { id: 'bitshin-dark', name: 'BitShin Dark' },
    { id: 'blitz-classic', name: 'Blitz3D Classic Blue' },
    { id: 'cyberpunk', name: 'Cyberpunk Neon' },
    { id: 'vs-dark', name: 'VS Dark' }
  ];

  function handleThemeChange(e: Event) {
    const val = (e.target as HTMLSelectElement).value;
    editorStore.settings.theme = val;
    editorStore.saveSettings();
  }
</script>

<div class="toolbar flex items-center justify-between px-3 h-10 bg-slate-900/95 border-b border-slate-800 text-slate-300 text-xs select-none">
  <!-- Left: File Operations & Run/Build -->
  <div class="flex items-center gap-1.5">
    <button
      onclick={() => editorStore.showNewModal = true}
      class="flex items-center gap-1 px-2.5 py-1.5 rounded hover:bg-slate-800 text-slate-300 hover:text-white transition"
      title="New File / Template (Ctrl+N)"
    >
      <FilePlus size={14} class="text-sky-400" />
      <span>New</span>
    </button>

    <button
      onclick={() => editorStore.openFileFromDisk()}
      class="flex items-center gap-1 px-2.5 py-1.5 rounded hover:bg-slate-800 text-slate-300 hover:text-white transition"
      title="Open File (Ctrl+O)"
    >
      <FolderOpen size={14} class="text-amber-400" />
      <span>Open</span>
    </button>

    <button
      onclick={() => editorStore.saveCurrentFile()}
      class="flex items-center gap-1 px-2.5 py-1.5 rounded hover:bg-slate-800 text-slate-300 hover:text-white transition"
      title="Save File (Ctrl+S)"
    >
      <Save size={14} class="text-emerald-400" />
      <span>Save</span>
    </button>

    <div class="w-px h-5 bg-slate-800 mx-1"></div>

    {#if editorStore.isRunning}
      <button
        onclick={() => editorStore.stopProgram()}
        class="flex items-center gap-1.5 px-3 py-1 rounded bg-rose-600 hover:bg-rose-500 text-white font-semibold transition shadow-sm"
      >
        <Square size={13} fill="currentColor" />
        <span>Stop</span>
      </button>
    {:else}
      <button
        onclick={() => editorStore.runActiveProgram()}
        class="flex items-center gap-1.5 px-3 py-1 rounded bg-emerald-600 hover:bg-emerald-500 text-white font-semibold transition shadow-sm hover:scale-105 active:scale-95"
      >
        <Play size={13} fill="currentColor" />
        <span>Run (F5)</span>
      </button>
    {/if}

    <button
      onclick={() => editorStore.showBuildModal = true}
      class="flex items-center gap-1 px-2.5 py-1.5 rounded hover:bg-slate-800 text-slate-300 hover:text-white transition border border-transparent hover:border-slate-700"
      title="Build & Package (F7)"
    >
      <Hammer size={14} class="text-indigo-400" />
      <span>Package</span>
    </button>
  </div>

  <!-- Center: Sidebar Navigator Tabs -->
  <div class="flex items-center bg-slate-950 p-0.5 rounded-lg border border-slate-800">
    <button
      onclick={() => editorStore.activeSidebarTab = 'files'}
      class="flex items-center gap-1 px-2.5 py-1 rounded-md text-[11px] font-medium transition {editorStore.activeSidebarTab === 'files' ? 'bg-sky-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
      title="Project Files"
    >
      <FolderTree size={13} />
      <span>Explorer</span>
    </button>

    <button
      onclick={() => editorStore.activeSidebarTab = 'commands'}
      class="flex items-center gap-1 px-2.5 py-1 rounded-md text-[11px] font-medium transition {editorStore.activeSidebarTab === 'commands' ? 'bg-sky-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
      title="Command Reference & Documentation Browser"
    >
      <BookOpen size={13} />
      <span>Commands</span>
    </button>

    <button
      onclick={() => editorStore.activeSidebarTab = 'outline'}
      class="flex items-center gap-1 px-2.5 py-1 rounded-md text-[11px] font-medium transition {editorStore.activeSidebarTab === 'outline' ? 'bg-sky-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
      title="Functions & Symbols Outline"
    >
      <Code2 size={13} />
      <span>Outline</span>
    </button>

    <button
      onclick={() => editorStore.activeSidebarTab = 'examples'}
      class="flex items-center gap-1 px-2.5 py-1 rounded-md text-[11px] font-medium transition {editorStore.activeSidebarTab === 'examples' ? 'bg-sky-600 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
      title="Sample Games & Demos Gallery"
    >
      <Sparkles size={13} />
      <span>Demos (69)</span>
    </button>
  </div>

  <!-- Right: Theme Selector & Utilities -->
  <div class="flex items-center gap-2">
    <div class="flex items-center gap-1.5 text-slate-400">
      <Palette size={13} />
      <select
        value={editorStore.settings.theme}
        onchange={handleThemeChange}
        class="bg-slate-950 border border-slate-800 text-slate-300 text-[11px] rounded px-2 py-1 focus:outline-none focus:border-sky-500 cursor-pointer"
      >
        {#each themes as t}
          <option value={t.id}>{t.name}</option>
        {/each}
      </select>
    </div>

    <div class="w-px h-5 bg-slate-800 mx-1"></div>

    <button
      onclick={() => editorStore.clearLogs()}
      class="flex items-center gap-1 px-2 py-1 rounded hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
      title="Clear Console Output"
    >
      <RefreshCw size={12} />
      <span>Clear Console</span>
    </button>
  </div>
</div>
