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
  import { X, Plus, FileCode, FilePlus, FolderOpen, Library } from 'lucide-svelte';

  let editorContainer: HTMLDivElement | null = $state(null);
  let editor = $state<monaco.editor.IStandaloneCodeEditor | null>(null);
  let breakDecos: monaco.editor.IEditorDecorationsCollection | null = null;

  $effect(() => {
    const lines = editorStore.breakpoints;
    const ed = editor;
    if (!ed) return;
    const marks = lines.map((line) => ({
      range: new monaco.Range(line, 1, line, 1),
      options: {
        isWholeLine: true,
        glyphMarginClassName: 'bitshin-bp',
        overviewRuler: {
          color: 'rgba(196, 72, 64, 0.95)',
          position: monaco.editor.OverviewRulerLane.Left
        }
      }
    }));
    if (!breakDecos) breakDecos = ed.createDecorationsCollection(marks);
    else breakDecos.set(marks);
  });
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
    editorStore.lspConnected = await startLspClient();
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
      lineNumbers: 'on',
      glyphMargin: true,
      folding: true,
      padding: { top: 8 },
      fontFamily: "'Geist Mono Variable', 'Cascadia Code', ui-monospace, Consolas, monospace",
      fontLigatures: true,
      smoothScrolling: true,
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
      editorStore.runActiveProgram(false);
    });

    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.F5, () => {
      editorStore.runActiveProgram(true);
    });

    editor.addCommand(monaco.KeyCode.F9, () => {
      const line = editor?.getPosition()?.lineNumber ?? 0;
      editorStore.toggleBreakpoint(line);
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
    if (!editor) return;
    if (!editorStore.activeTab) {
      editor.setModel(null);
      return;
    }
    const model = getOrCreateModel(editorStore.activeTab);
    if (model && editor.getModel() !== model) {
      editor.setModel(model);
      editor.focus();
    }
  }

  $effect(() => {
    const _ = editorStore.activeTabId;
    if (editor) {
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

<div class="ide-editor">
  <div class="ide-tabs">
    {#each editorStore.tabs as tab}
      <div
        onclick={() => editorStore.selectTab(tab.id)}
        class="ide-tab {tab.id === editorStore.activeTabId ? 'active' : ''}"
      >
        <FileCode size={12} class="tab-icon" />
        <span class="truncate font-mono" style="font-size:11px;">{tab.name}</span>
        {#if tab.isDirty}<span class="ide-dirty" title="Unsaved"></span>{/if}
        <button
          class="ide-tab-close"
          onclick={(e) => { e.stopPropagation(); editorStore.closeTab(tab.id); }}
          title="Close (Ctrl+W)"
        >
          <X size={11} />
        </button>
      </div>
    {/each}
    <button class="ide-iconbtn" style="margin-left:4px;" onclick={() => editorStore.newTab()} title="New file">
      <Plus size={13} />
    </button>
  </div>

  <div class="flex-1 min-h-0 relative">
    {#if !editorStore.activeTab}
      <div class="ide-empty">
        <div class="ide-mark" style="width:36px;height:36px;font-size:18px;">B</div>
        <h1>BitShin BASIC</h1>
        <p>Open a .bb file, start from a template, or browse the built-in examples.</p>
        <div class="ide-empty-actions">
          <button class="ide-btn primary" onclick={() => editorStore.showNewModal = true}>
            <FilePlus size={13} /> New file
          </button>
          <button class="ide-btn" onclick={() => editorStore.openFileFromDisk()}>
            <FolderOpen size={13} /> Open…
          </button>
          <button class="ide-btn" onclick={() => { editorStore.activeSidebarTab = 'examples'; editorStore.sidebarCollapsed = false; }}>
            <Library size={13} /> Examples
          </button>
        </div>
        <div class="ide-keys">
          <span><kbd>Ctrl+N</kbd> New</span>
          <span><kbd>Ctrl+O</kbd> Open</span>
          <span><kbd>F5</kbd> Run</span>
          <span><kbd>Ctrl+F5</kbd> Debug</span>
          <span><kbd>F9</kbd> Breakpoint</span>
        </div>
      </div>
    {/if}
    <div bind:this={editorContainer} class="w-full h-full" style={editorStore.activeTab ? '' : 'visibility:hidden;position:absolute;inset:0;'}></div>
  </div>
</div>
