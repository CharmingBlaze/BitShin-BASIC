<script lang="ts">
  import { onMount } from 'svelte';
  import TitleBar from './lib/components/TitleBar.svelte';
  import ToolBar from './lib/components/ToolBar.svelte';
  import StatusBar from './lib/components/StatusBar.svelte';
  import FileTree from './lib/components/FileTree.svelte';
  import CommandRef from './lib/components/CommandRef.svelte';
  import SymbolOutline from './lib/components/SymbolOutline.svelte';
  import ExamplesGallery from './lib/components/ExamplesGallery.svelte';
  import OutputPanel from './lib/components/OutputPanel.svelte';
  import Editor from './lib/editor/Editor.svelte';
  import NewFileModal from './lib/components/NewFileModal.svelte';
  import BuildModal from './lib/components/BuildModal.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import { editorStore } from './lib/stores/editorState.svelte';

  let isDraggingSidebar = $state(false);
  let isDraggingOutput = $state(false);

  function startSidebarDrag(e: MouseEvent) {
    e.preventDefault();
    isDraggingSidebar = true;
    window.addEventListener('mousemove', onSidebarDrag);
    window.addEventListener('mouseup', stopSidebarDrag);
  }

  function onSidebarDrag(e: MouseEvent) {
    if (!isDraggingSidebar) return;
    const newWidth = Math.max(200, Math.min(600, e.clientX));
    editorStore.settings.sidebarWidth = newWidth;
  }

  function stopSidebarDrag() {
    isDraggingSidebar = false;
    window.removeEventListener('mousemove', onSidebarDrag);
    window.removeEventListener('mouseup', stopSidebarDrag);
    editorStore.saveSettings();
  }

  function startOutputDrag(e: MouseEvent) {
    e.preventDefault();
    isDraggingOutput = true;
    window.addEventListener('mousemove', onOutputDrag);
    window.addEventListener('mouseup', stopOutputDrag);
  }

  function onOutputDrag(e: MouseEvent) {
    if (!isDraggingOutput) return;
    const newHeight = Math.max(100, Math.min(600, window.innerHeight - e.clientY - 24));
    editorStore.settings.outputHeight = newHeight;
  }

  function stopOutputDrag() {
    isDraggingOutput = false;
    window.removeEventListener('mousemove', onOutputDrag);
    window.removeEventListener('mouseup', stopOutputDrag);
    editorStore.saveSettings();
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'F5' && !e.shiftKey) {
      e.preventDefault();
      editorStore.runActiveProgram();
    } else if (e.key === 'F5' && e.shiftKey) {
      e.preventDefault();
      editorStore.stopProgram();
    } else if (e.key === 'F7') {
      e.preventDefault();
      editorStore.showBuildModal = true;
    } else if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      editorStore.saveCurrentFile();
    } else if ((e.ctrlKey || e.metaKey) && e.key === 'o') {
      e.preventDefault();
      editorStore.openFileFromDisk();
    } else if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
      e.preventDefault();
      editorStore.showNewModal = true;
    }
  }

  onMount(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  });
</script>

<div class="app-layout flex flex-col h-screen w-screen bg-slate-950 overflow-hidden select-none">
  <!-- Top Navigation & Controls -->
  <TitleBar />
  <ToolBar />

  <!-- Main IDE Workspace -->
  <div class="flex-1 flex min-h-0 relative">
    <!-- Collapsible Sidebar -->
    <div
      style="width: {editorStore.settings.sidebarWidth}px;"
      class="h-full flex flex-col shrink-0 min-w-0"
    >
      {#if editorStore.activeSidebarTab === 'files'}
        <FileTree />
      {:else if editorStore.activeSidebarTab === 'commands'}
        <CommandRef />
      {:else if editorStore.activeSidebarTab === 'outline'}
        <SymbolOutline />
      {:else if editorStore.activeSidebarTab === 'examples'}
        <ExamplesGallery />
      {/if}
    </div>

    <!-- Horizontal Resize Handle -->
    <div
      onmousedown={startSidebarDrag}
      class="split-handle-horizontal {isDraggingSidebar ? 'active' : ''}"
    ></div>

    <!-- Center Workspace (Editor + Output Console) -->
    <div class="flex-1 flex flex-col min-w-0 h-full">
      <!-- Editor View -->
      <div class="flex-1 min-h-0">
        <Editor />
      </div>

      <!-- Vertical Resize Handle -->
      <div
        onmousedown={startOutputDrag}
        class="split-handle-vertical {isDraggingOutput ? 'active' : ''}"
      ></div>

      <!-- Output Console Panel -->
      <div
        style="height: {editorStore.settings.outputHeight}px;"
        class="shrink-0 min-h-0"
      >
        <OutputPanel />
      </div>
    </div>
  </div>

  <!-- Bottom Status Bar -->
  <StatusBar />

  <!-- Modals -->
  {#if editorStore.showNewModal}
    <NewFileModal />
  {/if}

  {#if editorStore.showBuildModal}
    <BuildModal />
  {/if}

  {#if editorStore.showSettingsModal}
    <SettingsModal />
  {/if}
</div>
