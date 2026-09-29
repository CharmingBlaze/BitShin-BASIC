<script lang="ts">
  import { FilePlus, FolderOpen, Save, Play, Square, Hammer, Trash2 } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  const themes = [
    { id: 'bitshin-dark', name: 'BitShin Dark' },
    { id: 'blitz-classic', name: 'Blitz3D Classic' },
    { id: 'cyberpunk', name: 'Cyberpunk' },
    { id: 'vs-dark', name: 'VS Dark' }
  ];

  function handleThemeChange(e: Event) {
    const val = (e.target as HTMLSelectElement).value;
    editorStore.settings.theme = val;
    editorStore.saveSettings();
  }
</script>

<div class="ide-titlebar" style="justify-content: space-between; padding-right: 8px;">
  <div class="flex items-center gap-1 ide-nodrag">
    <button class="ide-btn" onclick={() => editorStore.showNewModal = true} title="New file (Ctrl+N)">
      <FilePlus size={13} /> New
    </button>
    <button class="ide-btn" onclick={() => editorStore.openFileFromDisk()} title="Open file (Ctrl+O)">
      <FolderOpen size={13} /> Open
    </button>
    <button class="ide-btn" onclick={() => editorStore.saveCurrentFile()} title="Save (Ctrl+S)">
      <Save size={13} /> Save
    </button>
    {#if editorStore.isRunning}
      <button class="ide-run stop" onclick={() => editorStore.stopProgram()} title="Stop (Shift+F5)">
        <Square size={11} fill="currentColor" /> Stop
      </button>
    {:else}
      <button class="ide-run" onclick={() => editorStore.runActiveProgram()} title="Run (F5)">
        <Play size={11} fill="currentColor" /> Run
      </button>
    {/if}
    <button class="ide-btn" onclick={() => editorStore.showBuildModal = true} title="Build (F7)">
      <Hammer size={13} /> Package
    </button>
  </div>

  <div class="ide-panel-tabs ide-nodrag">
    <button class="ide-panel-tab {editorStore.activeSidebarTab === 'files' ? 'active' : ''}" onclick={() => editorStore.activeSidebarTab = 'files'}>Explorer</button>
    <button class="ide-panel-tab {editorStore.activeSidebarTab === 'commands' ? 'active' : ''}" onclick={() => editorStore.activeSidebarTab = 'commands'}>Commands</button>
    <button class="ide-panel-tab {editorStore.activeSidebarTab === 'outline' ? 'active' : ''}" onclick={() => editorStore.activeSidebarTab = 'outline'}>Outline</button>
    <button class="ide-panel-tab {editorStore.activeSidebarTab === 'examples' ? 'active' : ''}" onclick={() => editorStore.activeSidebarTab = 'examples'}>Examples</button>
  </div>

  <div class="flex items-center gap-2 ide-nodrag">
    <select class="ide-field" style="width: auto; height: 22px;" value={editorStore.settings.theme} onchange={handleThemeChange}>
      {#each themes as t}
        <option value={t.id}>{t.name}</option>
      {/each}
    </select>
    <button class="ide-btn" onclick={() => editorStore.clearLogs()} title="Clear output">
      <Trash2 size={12} /> Clear
    </button>
  </div>
</div>
