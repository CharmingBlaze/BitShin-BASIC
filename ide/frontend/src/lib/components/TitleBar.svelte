<script lang="ts">
  import { Play, Square, Minus, Square as Maximize, X } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';
  import examplesData from '../data/examples.json';

  type MenuId = 'file' | 'run' | 'view' | null;
  let openMenu = $state<MenuId>(null);

  const featuredExamples = examplesData.slice(0, 8);

  function toggleMenu(id: Exclude<MenuId, null>) {
    openMenu = openMenu === id ? null : id;
  }

  function closeMenus() {
    openMenu = null;
  }

  function minimize() {
    window.runtime?.WindowMinimise?.();
  }

  function toggleMaximize() {
    window.runtime?.WindowToggleMaximise?.();
  }

  function closeWindow() {
    window.runtime?.Quit?.();
  }

  function fileNew() {
    closeMenus();
    editorStore.showNewModal = true;
  }

  function fileOpen() {
    closeMenus();
    editorStore.openFileFromDisk();
  }

  function fileSave() {
    closeMenus();
    editorStore.saveCurrentFile();
  }

  function showExamples() {
    closeMenus();
    editorStore.activeSidebarTab = 'examples';
    editorStore.sidebarCollapsed = false;
  }

  function openExample(filename: string) {
    closeMenus();
    editorStore.loadExample(filename);
  }
</script>

<svelte:window onclick={() => closeMenus()} />

<header class="ide-titlebar">
  <div class="ide-brand ide-nodrag">
    <div class="ide-mark" aria-hidden="true">B</div>
    <span class="ide-brand-name">BitShin BASIC</span>
  </div>

  <nav class="ide-menu ide-nodrag" aria-label="Application">
    <div class="ide-menu-wrap">
      <button
        class="ide-menu-btn {openMenu === 'file' ? 'open' : ''}"
        onclick={(e) => { e.stopPropagation(); toggleMenu('file'); }}
      >File</button>
      {#if openMenu === 'file'}
        <div class="ide-menu-drop" onclick={(e) => e.stopPropagation()}>
          <button class="ide-menu-item" onclick={fileNew}>New file <kbd>Ctrl+N</kbd></button>
          <button class="ide-menu-item" onclick={fileOpen}>Open… <kbd>Ctrl+O</kbd></button>
          <button class="ide-menu-item" onclick={fileSave}>Save <kbd>Ctrl+S</kbd></button>
          <div class="ide-menu-sep"></div>
          {#each featuredExamples as ex}
            <button class="ide-menu-item" onclick={() => openExample(ex.filename)}>{ex.name}</button>
          {/each}
          <button class="ide-menu-item" onclick={showExamples}>All examples…</button>
          <div class="ide-menu-sep"></div>
          <button class="ide-menu-item" onclick={() => { closeMenus(); closeWindow(); }}>Exit</button>
        </div>
      {/if}
    </div>

    <div class="ide-menu-wrap">
      <button
        class="ide-menu-btn {openMenu === 'run' ? 'open' : ''}"
        onclick={(e) => { e.stopPropagation(); toggleMenu('run'); }}
      >Run</button>
      {#if openMenu === 'run'}
        <div class="ide-menu-drop" onclick={(e) => e.stopPropagation()}>
          <button class="ide-menu-item" onclick={() => { closeMenus(); editorStore.runActiveProgram(); }}>Run <kbd>F5</kbd></button>
          <button class="ide-menu-item" onclick={() => { closeMenus(); editorStore.stopProgram(); }}>Stop <kbd>Shift+F5</kbd></button>
          <button class="ide-menu-item" onclick={() => { closeMenus(); editorStore.showBuildModal = true; }}>Build package… <kbd>F7</kbd></button>
          <div class="ide-menu-sep"></div>
          <button class="ide-menu-item" onclick={() => { closeMenus(); editorStore.toggleOutputPanel(); }}>Toggle output <kbd>Ctrl+`</kbd></button>
        </div>
      {/if}
    </div>

    <div class="ide-menu-wrap">
      <button
        class="ide-menu-btn {openMenu === 'view' ? 'open' : ''}"
        onclick={(e) => { e.stopPropagation(); toggleMenu('view'); }}
      >View</button>
      {#if openMenu === 'view'}
        <div class="ide-menu-drop" onclick={(e) => e.stopPropagation()}>
          <button class="ide-menu-item" onclick={() => { closeMenus(); editorStore.activeSidebarTab = 'files'; editorStore.sidebarCollapsed = false; }}>Explorer</button>
          <button class="ide-menu-item" onclick={() => { closeMenus(); editorStore.activeSidebarTab = 'commands'; editorStore.sidebarCollapsed = false; }}>Commands</button>
          <button class="ide-menu-item" onclick={() => { closeMenus(); editorStore.activeSidebarTab = 'outline'; editorStore.sidebarCollapsed = false; }}>Outline</button>
          <button class="ide-menu-item" onclick={showExamples}>Examples</button>
          <div class="ide-menu-sep"></div>
          <button class="ide-menu-item" onclick={() => { closeMenus(); editorStore.showSettingsModal = true; }}>Settings…</button>
        </div>
      {/if}
    </div>
  </nav>

  <div class="ide-title-center">
    {#if editorStore.activeTab}
      <span class="name">{editorStore.activeTab.name}</span>
      {#if editorStore.activeTab.isDirty}<span class="ide-dirty" title="Unsaved changes"></span>{/if}
    {:else}
      BitShin BASIC
    {/if}
  </div>

  <div class="flex items-center h-full ide-nodrag">
    {#if editorStore.isRunning}
      <button class="ide-run stop" onclick={() => editorStore.stopProgram()} title="Stop (Shift+F5)">
        <Square size={10} fill="currentColor" />
        Stop
      </button>
    {:else}
      <button class="ide-run" onclick={() => editorStore.runActiveProgram()} title="Run (F5)">
        <Play size={10} fill="currentColor" />
        Run
      </button>
    {/if}

    <button class="ide-winbtn" onclick={minimize} title="Minimize"><Minus size={12} /></button>
    <button class="ide-winbtn" onclick={toggleMaximize} title="Maximize"><Maximize size={11} /></button>
    <button class="ide-winbtn close" onclick={closeWindow} title="Close"><X size={13} /></button>
  </div>
</header>
