<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import * as monaco from 'monaco-editor';
  import { registerBitShinLanguage, LANGUAGE_ID } from './bitshin-lang';
  import { registerCompletions } from './completions';
  import { registerHover } from './hover';
  import { editorStore } from '../stores/editorState.svelte';
  import { AppAPI } from '../wailsBridge';
  import {
    startLspClient,
    lspDidOpen,
    lspDidChange,
    lspDidClose,
    tabToUri,
    onLspDiagnostics,
    lspRequest
  } from './lspClient';
  import { X, Plus, FileCode } from 'lucide-svelte';

  let editorContainer: HTMLDivElement | null = $state(null);
  let editor: monaco.editor.IStandaloneCodeEditor | null = null;
  const models = new Map<string, monaco.editor.ITextModel>();
  const modelUris = new Map<string, string>();
  let repoRoot = '';
  let unsubDiagnostics: (() => void) | null = null;

  onMount(async () => {
    registerBitShinLanguage();
    registerCompletions();
    registerHover();
    registerDefinition();

    try {
      repoRoot = await AppAPI.getRepoRoot();
    } catch {
      repoRoot = '';
    }
    await startLspClient();
    unsubDiagnostics = onLspDiagnostics(applyDiagnostics);

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
      fontFamily: "'JetBrains Mono', 'Fira Code', Consolas, 'Courier New', monospace",
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
        const uri = modelUris.get(editorStore.activeTab.id);
        if (uri) {
          lspDidChange(uri, val);
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

    // Responsive layout tracking
    const resizeObserver = new ResizeObserver(() => {
      editor?.layout();
    });
    resizeObserver.observe(editorContainer);

    // Initial Model
    updateEditorModel();
  });

  onDestroy(() => {
    unsubDiagnostics?.();
    for (const uri of modelUris.values()) {
      lspDidClose(uri);
    }
    models.forEach(m => m.dispose());
    models.clear();
    modelUris.clear();
    if (editor) {
      editor.dispose();
    }
  });

  function registerDefinition() {
    monaco.languages.registerDefinitionProvider(LANGUAGE_ID, {
      provideDefinition: async (model, position) => {
        const result = await lspRequest('textDocument/definition', {
          textDocument: { uri: model.uri.toString() },
          position: { line: position.lineNumber - 1, character: position.column - 1 }
        });
        const locs = Array.isArray(result) ? result : result ? [result] : [];
        return locs.map((l: any) => ({
          uri: monaco.Uri.parse(l.uri),
          range: new monaco.Range(
            (l.range?.start?.line ?? 0) + 1,
            (l.range?.start?.character ?? 0) + 1,
            (l.range?.end?.line ?? 0) + 1,
            (l.range?.end?.character ?? 0) + 1
          )
        }));
      }
    });
  }

  function applyDiagnostics(uri: string, diagnostics: any[]) {
    const model = [...models.values()].find(m => m.uri.toString() === uri);
    if (!model) return;
    const markers = diagnostics.map((d) => ({
      severity: d.severity === 1 ? monaco.MarkerSeverity.Error : monaco.MarkerSeverity.Warning,
      message: d.message || '',
      startLineNumber: (d.range?.start?.line ?? 0) + 1,
      startColumn: (d.range?.start?.character ?? 0) + 1,
      endLineNumber: (d.range?.end?.line ?? 0) + 1,
      endColumn: (d.range?.end?.character ?? 0) + 1
    }));
    monaco.editor.setModelMarkers(model, 'bsls', markers);
  }

  function getOrCreateModel(tab: typeof editorStore.activeTab) {
    if (!tab) return null;
    let model = models.get(tab.id);
    if (!model || model.isDisposed()) {
      const uriStr = tabToUri(tab.path, tab.name, repoRoot);
      model = monaco.editor.createModel(tab.content, LANGUAGE_ID, monaco.Uri.parse(uriStr));
      models.set(tab.id, model);
      modelUris.set(tab.id, model.uri.toString());
      lspDidOpen(model.uri.toString(), tab.content);
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
  <div class="tabs-bar flex items-center bg-slate-950 border-b border-slate-800 px-1 overflow-x-auto scrollbar-none h-8 shrink-0">
    {#each editorStore.tabs as tab}
      <div
        onclick={() => editorStore.selectTab(tab.id)}
        class="flex items-center gap-1.5 px-3 py-1 text-xs cursor-pointer border-t-2 border-r border-slate-800/60 transition group {tab.id === editorStore.activeTabId ? 'bg-slate-900 text-sky-400 border-t-sky-400 font-medium' : 'bg-slate-950 text-slate-400 border-t-transparent hover:bg-slate-900/50 hover:text-slate-200'}"
      >
        <FileCode size={12} class="{tab.id === editorStore.activeTabId ? 'text-sky-400' : 'text-slate-500'}" />
        <span class="truncate max-w-[130px] font-mono text-[11px]">{tab.name}</span>

        {#if tab.isDirty}
          <span class="w-1.5 h-1.5 rounded-full bg-amber-400 shrink-0" title="Unsaved changes"></span>
        {/if}

        <button
          onclick={(e) => {
            e.stopPropagation();
            editorStore.closeTab(tab.id);
          }}
          class="p-0.5 rounded hover:bg-slate-800 text-slate-500 hover:text-slate-200 transition opacity-0 group-hover:opacity-100 ml-1"
          title="Close Tab (Ctrl+W)"
        >
          <X size={11} />
        </button>
      </div>
    {/each}

    <!-- New Tab Button -->
    <button
      onclick={() => editorStore.newTab()}
      class="p-1 rounded hover:bg-slate-800 text-slate-500 hover:text-slate-300 transition ml-1"
      title="New File Tab"
    >
      <Plus size={13} />
    </button>
  </div>

  <!-- Monaco Editor Mount Container -->
  <div class="flex-1 min-h-0 relative">
    <div bind:this={editorContainer} class="w-full h-full"></div>
  </div>
</div>
