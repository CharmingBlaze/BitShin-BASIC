<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import * as monaco from 'monaco-editor';
  import { registerBitShinLanguage, LANGUAGE_ID } from './bitshin-lang';
  import { registerCompletions } from './completions';
  import { registerHover } from './hover';
  import { editorStore } from '../stores/editorState.svelte';
  import { X, Plus } from 'lucide-svelte';

  let editorContainer: HTMLDivElement | null = $state(null);
  let editor: monaco.editor.IStandaloneCodeEditor | null = null;
  const models = new Map<string, monaco.editor.ITextModel>();

  onMount(() => {
    registerBitShinLanguage();
    registerCompletions();
    registerHover();

    if (!editorContainer) return;

    editor = monaco.editor.create(editorContainer, {
      theme: editorStore.settings.theme,
      fontSize: editorStore.settings.fontSize,
      tabSize: editorStore.settings.tabSize,
      minimap: { enabled: editorStore.settings.minimap },
      wordWrap: editorStore.settings.wordWrap,
      automaticLayout: true,
      scrollBeyondLastLine: false,
      cursorBlinking: 'smooth',
      renderLineHighlight: 'all',
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', 'Courier New', monospace",
      fontLigatures: true,
      bracketPairColorization: { enabled: true }
    });

    (window as any).__monacoEditor = editor;

    // Track Cursor Position
    editor.onDidChangeCursorPosition((e) => {
      editorStore.cursorPos = {
        line: e.position.lineNumber,
        col: e.position.column
      };
    });

    // Track Content Changes
    editor.onDidChangeModelContent(() => {
      if (editor && editorStore.activeTab) {
        const val = editor.getValue();
        if (val !== editorStore.activeTab.content) {
          editorStore.activeTab.content = val;
          editorStore.activeTab.isDirty = true;
        }
      }
    });

    // Keyboard Shortcuts
    editor.addCommand(monaco.KeyCode.F5, () => {
      editorStore.runActiveProgram();
    });

    editor.addCommand(monaco.KeyMod.Shift | monaco.KeyCode.F5, () => {
      editorStore.stopProgram();
    });

    editor.addCommand(monaco.KeyCode.F7, () => {
      editorStore.showBuildModal = true;
    });

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      editorStore.saveCurrentFile();
    });

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyO, () => {
      editorStore.openFileFromDisk();
    });

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyN, () => {
      editorStore.showNewModal = true;
    });

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyW, () => {
      if (editorStore.activeTab) {
        editorStore.closeTab(editorStore.activeTab.id);
      }
    });

    // Initial Model
    updateEditorModel();
  });

  onDestroy(() => {
    models.forEach(m => m.dispose());
    models.clear();
    if (editor) {
      editor.dispose();
    }
  });

  function getOrCreateModel(tab: typeof editorStore.activeTab) {
    if (!tab) return null;
    let model = models.get(tab.id);
    if (!model || model.isDisposed()) {
      model = monaco.editor.createModel(tab.content, LANGUAGE_ID);
      models.set(tab.id, model);
    }
    return model;
  }

  function updateEditorModel() {
    if (!editor || !editorStore.activeTab) return;
    const model = getOrCreateModel(editorStore.activeTab);
    if (model && editor.getModel() !== model) {
      editor.setModel(model);
      editor.focus();
    }
  }

  // Effect to switch model when activeTabId changes
  $effect(() => {
    const _ = editorStore.activeTabId;
    if (editor && editorStore.activeTab) {
      updateEditorModel();
    }
  });

  // Effect to update theme or font settings
  $effect(() => {
    if (editor) {
      monaco.editor.setTheme(editorStore.settings.theme);
      editor.updateOptions({
        fontSize: editorStore.settings.fontSize,
        tabSize: editorStore.settings.tabSize,
        minimap: { enabled: editorStore.settings.minimap },
        wordWrap: editorStore.settings.wordWrap
      });
    }
  });
</script>

<div class="editor-view flex flex-col h-full bg-slate-950 select-none">
  <!-- Document Tabs Bar -->
  <div class="tabs-bar flex items-center bg-slate-950 border-b border-slate-800/80 px-2 overflow-x-auto scrollbar-none h-9 shrink-0 gap-1">
    {#each editorStore.tabs as tab}
      <div
        onclick={() => editorStore.selectTab(tab.id)}
        class="flex items-center gap-2 px-3 py-1.5 rounded-t-md text-xs cursor-pointer border-t-2 transition group {tab.id === editorStore.activeTabId ? 'bg-slate-900 text-sky-400 border-sky-400 font-medium' : 'bg-slate-950 text-slate-400 border-transparent hover:bg-slate-900/60 hover:text-slate-200'}"
      >
        <span class="truncate max-w-[140px] font-mono text-[11px]">{tab.name}</span>

        {#if tab.isDirty}
          <span class="w-1.5 h-1.5 rounded-full bg-amber-400 shrink-0"></span>
        {/if}

        <button
          onclick={(e) => {
            e.stopPropagation();
            editorStore.closeTab(tab.id);
          }}
          class="p-0.5 rounded hover:bg-slate-800 text-slate-500 hover:text-slate-200 transition opacity-0 group-hover:opacity-100"
          title="Close tab (Ctrl+W)"
        >
          <X size={11} />
        </button>
      </div>
    {/each}

    <!-- New Tab Button -->
    <button
      onclick={() => editorStore.newTab()}
      class="p-1.5 rounded hover:bg-slate-800 text-slate-500 hover:text-slate-300 transition"
      title="New Tab"
    >
      <Plus size={13} />
    </button>
  </div>

  <!-- Monaco Editor Mount Container -->
  <div class="flex-1 min-h-0 relative">
    <div bind:this={editorContainer} class="w-full h-full"></div>
  </div>
</div>
