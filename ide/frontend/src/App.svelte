<script lang="ts">
  import { onMount } from 'svelte';
  import TitleBar from './lib/components/TitleBar.svelte';
  import ActivityBar from './lib/components/ActivityBar.svelte';
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
    const newWidth = Math.max(180, Math.min(420, e.clientX - 44));
    editorStore.settings.sidebarWidth = newWidth;
    if (newWidth > 180) editorStore.sidebarCollapsed = false;
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
    editorStore.isOutputCollapsed = false;
    window.addEventListener('mousemove', onOutputDrag);
    window.addEventListener('mouseup', stopOutputDrag);
  }

  function onOutputDrag(e: MouseEvent) {
    if (!isDraggingOutput) return;
    const maxOutputHeight = Math.min(320, Math.floor(window.innerHeight * 0.4));
    const newHeight = Math.max(80, Math.min(maxOutputHeight, window.innerHeight - e.clientY - 22));
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
    } else if ((e.ctrlKey || e.metaKey) && e.key === '`') {
      e.preventDefault();
      editorStore.toggleOutputPanel();
    }
  }

  onMount(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  });
</script>

<div class="ide-app">
  <TitleBar />

  <div class="ide-body">
    <ActivityBar />

    {#if !editorStore.sidebarCollapsed}
      <div
        style="width: {editorStore.settings.sidebarWidth}px;"
        class="h-full shrink-0 min-w-0"
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
    {/if}

    <div
      role="separator"
      aria-orientation="vertical"
      onmousedown={startSidebarDrag}
      class="split-h {isDraggingSidebar ? 'active' : ''}"
      title="Resize sidebar"
    ></div>

    <div class="flex-1 flex flex-col min-w-0 h-full">
      <div class="flex-1 min-h-0">
        <Editor />
      </div>

      <div
        role="separator"
        aria-orientation="horizontal"
        onmousedown={startOutputDrag}
        ondblclick={() => editorStore.toggleOutputPanel()}
        class="split-v {isDraggingOutput ? 'active' : ''}"
        title="Resize output · double-click to toggle"
      ></div>

      <div
        style="height: {editorStore.isOutputCollapsed ? '28px' : `${editorStore.settings.outputHeight}px`};"
        class="shrink-0 min-h-0"
      >
        <OutputPanel />
      </div>
    </div>
  </div>

  <StatusBar />

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
