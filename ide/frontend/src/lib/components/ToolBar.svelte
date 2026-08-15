<script lang="ts">
  import {
    FilePlus, FolderOpen, Save, Play, Square, Hammer,
    FolderTree, BookOpen, Code2, Sparkles, Terminal,
    Palette, Trash2, ChevronDown
  } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  const themes = [
    { id: 'bitshin-dark', name: 'BitShin Dark' },
    { id: 'blitz-classic', name: 'Blitz3D Classic' },
    { id: 'cyberpunk', name: 'Cyberpunk Neon' },
    { id: 'vs-dark', name: 'VS Dark' }
  ];

  function handleThemeChange(e: Event) {
    const val = (e.target as HTMLSelectElement).value;
    editorStore.settings.theme = val;
    editorStore.saveSettings();
  }
</script>

<div class="toolbar flex items-center justify-between px-3 h-9 bg-slate-900 border-b border-slate-800 text-slate-300 text-xs select-none">
  <!-- Left: File Operations & Actions -->
  <div class="flex items-center gap-1">
    <button
      onclick={() => editorStore.showNewModal = true}
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md hover:bg-slate-800 text-slate-300 hover:text-white transition text-xs font-medium"
      title="New File / Template (Ctrl+N)"
    >
      <FilePlus size={13} class="text-sky-400" />
      <span>New</span>
    </button>

    <button
      onclick={() => editorStore.openFileFromDisk()}
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md hover:bg-slate-800 text-slate-300 hover:text-white transition text-xs font-medium"
      title="Open File (Ctrl+O)"
    >
      <FolderOpen size={13} class="text-amber-400" />
      <span>Open</span>
    </button>

    <button
      onclick={() => editorStore.saveCurrentFile()}
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md hover:bg-slate-800 text-slate-300 hover:text-white transition text-xs font-medium"
      title="Save File (Ctrl+S)"
    >
      <Save size={13} class="text-emerald-400" />
      <span>Save</span>
    </button>

    <div class="w-px h-4 bg-slate-800 mx-1"></div>

    {#if editorStore.isRunning}
      <button
        onclick={() => editorStore.stopProgram()}
        class="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-rose-600 hover:bg-rose-500 text-white font-medium text-xs transition shadow-sm"
        title="Stop Execution (Shift+F5)"
      >
        <Square size={11} fill="currentColor" />
        <span>Stop</span>
      </button>
    {:else}
      <button
        onclick={() => editorStore.runActiveProgram()}
        class="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-emerald-600 hover:bg-emerald-500 text-white font-medium text-xs transition shadow-sm"
        title="Run Current Program (F5)"
      >
        <Play size={11} fill="currentColor" />
        <span>Run</span>
      </button>
    {/if}

    <button
      onclick={() => editorStore.showBuildModal = true}
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-md hover:bg-slate-800 text-slate-300 hover:text-white transition text-xs font-medium"
      title="Build & Package Standalone (F7)"
    >
      <Hammer size={13} class="text-indigo-400" />
      <span>Package</span>
    </button>
  </div>

  <!-- Center: Sidebar Segmented Mode Switcher -->
  <div class="flex items-center bg-slate-950 p-0.5 rounded-lg border border-slate-800/90 gap-0.5">
    <button
      onclick={() => editorStore.activeSidebarTab = 'files'}
      class="flex items-center gap-1.5 px-2.5 py-1 rounded text-[11px] font-medium transition {editorStore.activeSidebarTab === 'files' ? 'bg-slate-800 text-sky-400 shadow-sm border border-slate-700/80 font-semibold' : 'text-slate-400 hover:text-slate-200'}"
      title="Project Explorer"
    >
      <FolderTree size={12} />
      <span>Explorer</span>
    </button>

    <button
      onclick={() => editorStore.activeSidebarTab = 'commands'}
      class="flex items-center gap-1.5 px-2.5 py-1 rounded text-[11px] font-medium transition {editorStore.activeSidebarTab === 'commands' ? 'bg-slate-800 text-sky-400 shadow-sm border border-slate-700/80 font-semibold' : 'text-slate-400 hover:text-slate-200'}"
      title="Command Reference & API"
    >
      <BookOpen size={12} />
      <span>Commands</span>
    </button>

    <button
      onclick={() => editorStore.activeSidebarTab = 'outline'}
      class="flex items-center gap-1.5 px-2.5 py-1 rounded text-[11px] font-medium transition {editorStore.activeSidebarTab === 'outline' ? 'bg-slate-800 text-sky-400 shadow-sm border border-slate-700/80 font-semibold' : 'text-slate-400 hover:text-slate-200'}"
      title="Symbol & Functions Outline"
    >
      <Code2 size={12} />
      <span>Outline</span>
    </button>

    <button
      onclick={() => editorStore.activeSidebarTab = 'examples'}
      class="flex items-center gap-1.5 px-2.5 py-1 rounded text-[11px] font-medium transition {editorStore.activeSidebarTab === 'examples' ? 'bg-slate-800 text-sky-400 shadow-sm border border-slate-700/80 font-semibold' : 'text-slate-400 hover:text-slate-200'}"
      title="69 Built-in Demos and Game Templates"
    >
      <Sparkles size={12} />
      <span>Demos (69)</span>
    </button>
  </div>

  <!-- Right: Theme & Utilities -->
  <div class="flex items-center gap-2">
    <div class="flex items-center gap-1 px-2 py-0.5 rounded-md bg-slate-950 border border-slate-800 text-slate-300 text-[11px]">
      <Palette size={12} class="text-slate-400" />
      <select
        value={editorStore.settings.theme}
        onchange={handleThemeChange}
        class="bg-transparent text-slate-300 text-[11px] focus:outline-none cursor-pointer pr-1"
      >
        {#each themes as t}
          <option value={t.id}>{t.name}</option>
        {/each}
      </select>
    </div>

    <div class="w-px h-4 bg-slate-800 mx-0.5"></div>

    <button
      onclick={() => editorStore.clearLogs()}
      class="flex items-center gap-1 px-2 py-1 rounded-md hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition text-[11px]"
      title="Clear Console Output"
    >
      <Trash2 size={12} />
      <span>Clear</span>
    </button>
  </div>
</div>
