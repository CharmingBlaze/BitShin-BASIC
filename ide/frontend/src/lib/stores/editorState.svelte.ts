import type { EditorTab, FileNode, ConsoleMessage, UserSettings, SymbolItem, DebugState, WatchItem } from '../types';
import { AppAPI } from '../wailsBridge';
import examplesData from '../data/examples.json';

const defaultCode = `; moonBASIC — Modern 3D/2D Game Engine
; Press F5 to Run, F7 to Build Standalone

Graphics3D(1280, 720, 0, 2)
SetWindowTitle("moonBASIC — 3D Showcase")

camera = CreateCamera()
PositionEntity(camera, 0, 3.2, -8)
RotateEntity(camera, 18, 0, 0)
CameraRange(camera, 0.1, 4000)
CameraClsColor(22, 28, 42)
AmbientLight(72, 82, 102)

light = CreateLight(1)
SetLightDirection(light, 50, 35, 0)
SetLightColor(light, 255, 240, 220)

; Create central rotating mesh
cube = CreateCube()
EntityColor(cube, 56, 189, 248)
PointEntity(camera, cube)

; Ground plane
plane = CreatePlane(40, 40)
PositionEntity(plane, 0, -1, 0)
EntityColor(plane, 30, 41, 59)

Print("Game loop started. Use ESC to quit.")

While Not KeyDown(1)
    dt# = DeltaTime() * 60
    TurnEntity(cube, 0.4 * dt, 0.7 * dt, 0)

    RenderWorld
    Color(230, 236, 245)
    Text(16, 16, "moonBASIC — 3D Showcase")
    Text(16, 40, "ESC to quit")
    Flip
Wend
End
`;

class EditorStore {
  tabs = $state<EditorTab[]>([
    {
      id: 'tab_welcome',
      name: 'main.mb',
      path: 'temp://main.mb',
      content: defaultCode,
      isDirty: false,
      isTemporary: true
    }
  ]);

  activeTabId = $state<string>('tab_welcome');
  projectTree = $state<FileNode | null>(null);
  consoleLogs = $state<ConsoleMessage[]>([]);
  isRunning = $state<boolean>(false);
  executionTime = $state<number>(0);
  exitCode = $state<number | null>(null);
  activeSidebarTab = $state<'files' | 'commands' | 'outline' | 'examples' | 'assets' | 'scene' | 'debug'>('files');
  breakpoints = $state<number[]>([]);
  debugPaused = $state(false);
  debugSession = $state(false);
  debugState = $state<DebugState | null>(null);
  watches = $state<WatchItem[]>([]);
  pausedLine = $derived(this.debugPaused && this.debugState ? this.debugState.line : 0);
  sidebarCollapsed = $state<boolean>(false);
  cursorPos = $state<{ line: number; col: number }>({ line: 1, col: 1 });
  lspConnected = $state<boolean>(false);
  panelTab = $state<'output' | 'problems'>('output');
  
  isOutputCollapsed = $state<boolean>(false);
  
  settings = $state<UserSettings>({
    theme: 'moonbasic-dark',
    fontSize: 14,
    tabSize: 4,
    minimap: true,
    wordWrap: 'off',
    autoSave: true,
    targetOS: 'windows',
    outputHeight: 200,
    sidebarWidth: 300
  });

  showNewModal = $state<boolean>(false);
  showBuildModal = $state<boolean>(false);
  showSettingsModal = $state<boolean>(false);
  showHelpModal = $state<boolean>(false);
  filterSearch = $state<string>('');

  toggleOutputPanel(show?: boolean) {
    if (show !== undefined) {
      this.isOutputCollapsed = !show;
    } else {
      this.isOutputCollapsed = !this.isOutputCollapsed;
    }
  }

  // Svelte 5 derived state
  activeTab = $derived.by(() => {
    return this.tabs.find(t => t.id === this.activeTabId) || null;
  });

  symbols = $derived.by<SymbolItem[]>(() => {
    if (!this.activeTab) return [];
    const lines = this.activeTab.content.split('\n');
    const list: SymbolItem[] = [];

    const fnRegex = /^\s*Function\s+([a-zA-Z_]\w*[\$#%]?)\s*(\([^\)]*\))?/i;
    const typeRegex = /^\s*Type\s+([a-zA-Z_]\w*)/i;
    const constRegex = /^\s*Const\s+([a-zA-Z_]\w*)/i;
    const labelRegex = /^\s*\.([a-zA-Z_]\w*)/i;

    lines.forEach((line, idx) => {
      const lineNum = idx + 1;
      let m = fnRegex.exec(line);
      if (m) {
        list.push({
          name: m[1],
          type: 'function',
          line: lineNum,
          signature: m[2] ? `${m[1]}${m[2]}` : `${m[1]}()`
        });
        return;
      }
      m = typeRegex.exec(line);
      if (m) {
        list.push({
          name: m[1],
          type: 'type',
          line: lineNum
        });
        return;
      }
      m = constRegex.exec(line);
      if (m) {
        list.push({
          name: m[1],
          type: 'const',
          line: lineNum
        });
        return;
      }
      m = labelRegex.exec(line);
      if (m) {
        list.push({
          name: m[1],
          type: 'label',
          line: lineNum
        });
      }
    });

    return list;
  });

  constructor() {
    this.initEventListeners();
    this.loadInitialSettings();
    this.refreshProjectTree();
  }

  async loadInitialSettings() {
    try {
      const s = await AppAPI.getSettings();
      if (s) {
        if (s.outputHeight) {
          s.outputHeight = Math.max(80, Math.min(260, s.outputHeight));
        }
        if (s.sidebarWidth) {
          s.sidebarWidth = Math.max(180, Math.min(380, s.sidebarWidth));
        }
        this.settings = { ...this.settings, ...s };
      }
    } catch {}
  }

  async saveSettings() {
    this.settings.outputHeight = Math.max(80, Math.min(260, this.settings.outputHeight));
    this.settings.sidebarWidth = Math.max(180, Math.min(380, this.settings.sidebarWidth));
    await AppAPI.saveSettings(this.settings);
  }

  async refreshProjectTree(root = '') {
    try {
      const tree = await AppAPI.getProjectTree(root);
      if (tree) {
        this.projectTree = tree;
      }
    } catch {}
  }

  initEventListeners() {
    if (typeof window !== 'undefined' && window.runtime?.EventsOn) {
      window.runtime.EventsOn('run:start', (data: any) => {
        this.isRunning = true;
        this.exitCode = null;
        this.debugPaused = false;
        this.debugState = null;
        this.addLog({
          id: Math.random().toString(36).substring(7),
          type: 'system',
          text: `[${data.time}] Started: ${data.file}`,
          time: data.time
        });
      });

      window.runtime.EventsOn('run:stdout', (line: string) => {
        // DebugLog prints "[debug] file.mb:12: message"
        const dbg = line.match(/^\[debug[^\]]*\] .*?:(\d+): /);
        this.addLog({
          id: Math.random().toString(36).substring(7),
          type: 'stdout',
          text: line,
          time: new Date().toLocaleTimeString(),
          line: dbg ? parseInt(dbg[1], 10) : undefined
        });
      });

      window.runtime.EventsOn('run:stderr', (line: string) => {
        if (this.handleDebugLine(line)) return;
        // Try parsing file and line info from error
        const lineMatch = line.match(/(?:at\s+|line\s+|:)(\d+)(?::(\d+))?/i);
        const lineNum = lineMatch ? parseInt(lineMatch[1]) : undefined;
        const colNum = lineMatch && lineMatch[2] ? parseInt(lineMatch[2]) : undefined;

        this.addLog({
          id: Math.random().toString(36).substring(7),
          type: 'stderr',
          text: line,
          time: new Date().toLocaleTimeString(),
          line: lineNum,
          col: colNum
        });
      });

      window.runtime.EventsOn('run:exit', (data: any) => {
        this.isRunning = false;
        this.endDebugSession();
        this.exitCode = data.code;
        this.executionTime = data.elapsed;
        this.addLog({
          id: Math.random().toString(36).substring(7),
          type: data.code === 0 ? 'success' : 'error',
          text: `[${data.time}] Process finished with exit code ${data.code} (${data.elapsed.toFixed(2)}s)`,
          time: data.time
        });
      });

      window.runtime.EventsOn('run:stopped', (msg: string) => {
        this.isRunning = false;
        this.endDebugSession();
        this.addLog({
          id: Math.random().toString(36).substring(7),
          type: 'system',
          text: `Program execution terminated by user.`,
          time: new Date().toLocaleTimeString()
        });
      });
    }
  }

  openTab(file: { path: string; name: string; content: string }) {
    const existing = this.tabs.find(t => t.path === file.path);
    if (existing) {
      existing.content = file.content;
      existing.name = file.name;
      existing.isDirty = false;
      this.activeTabId = existing.id;
      return;
    }
    const newId = 'tab_' + Math.random().toString(36).substring(2, 9);
    this.tabs.push({
      id: newId,
      name: file.name,
      path: file.path,
      content: file.content,
      isDirty: false
    });
    this.activeTabId = newId;
  }

  newTab(templateName = 'New Script', initialCode = defaultCode) {
    const newId = 'tab_' + Math.random().toString(36).substring(2, 9);
    const count = this.tabs.filter(t => t.isTemporary).length + 1;
    const name = `untitled_${count}.mb`;
    this.tabs.push({
      id: newId,
      name,
      path: `temp://${name}`,
      content: initialCode,
      isDirty: false,
      isTemporary: true
    });
    this.activeTabId = newId;
  }

  closeTab(id: string) {
    const idx = this.tabs.findIndex(t => t.id === id);
    if (idx === -1) return;
    
    this.tabs.splice(idx, 1);
    if (this.tabs.length === 0) {
      this.activeTabId = '';
    } else if (this.activeTabId === id) {
      const nextIdx = Math.max(0, idx - 1);
      this.activeTabId = this.tabs[nextIdx].id;
    }
  }

  selectTab(id: string) {
    this.activeTabId = id;
  }

  updateActiveContent(content: string) {
    if (this.activeTab && this.activeTab.content !== content) {
      this.activeTab.content = content;
      this.activeTab.isDirty = true;
    }
  }

  async saveCurrentFile() {
    if (!this.activeTab) return;
    if (this.activeTab.isTemporary) {
      try {
        const chosen = await AppAPI.selectSaveFile(this.activeTab.name, this.activeTab.content);
        if (chosen) {
          this.activeTab.path = chosen;
          this.activeTab.name = chosen.split(/[\\/]/).pop() || this.activeTab.name;
          this.activeTab.isTemporary = false;
          this.activeTab.isDirty = false;
          this.refreshProjectTree();
        }
      } catch (err) {
        console.error('Save failed:', err);
      }
    } else {
      try {
        await AppAPI.saveFile(this.activeTab.path, this.activeTab.content);
        this.activeTab.isDirty = false;
      } catch (err) {
        console.error('Save failed:', err);
      }
    }
  }

  async openFileFromDisk() {
    try {
      const res = await AppAPI.selectOpenFile();
      if (res && res.path) {
        this.openTab({
          path: res.path,
          name: res.name,
          content: res.content
        });
      }
    } catch (err) {
      console.error('Open file error:', err);
    }
  }

  async loadExample(filename: string) {
    try {
      const root = await AppAPI.getRepoRoot();
      if (root) {
        const sep = root.includes('\\') ? '\\' : '/';
        const res = await AppAPI.openFile(`${root}${sep}examples${sep}${filename}`);
        if (res?.content) {
          this.openTab({
            path: res.path,
            name: res.name || filename,
            content: res.content
          });
          return;
        }
      }
    } catch (err) {
      console.error('Open example from disk failed:', err);
    }
    const ex = examplesData.find(e => e.filename === filename);
    if (ex) {
      this.openTab({
        path: `example://${ex.filename}`,
        name: ex.filename,
        content: ex.code
      });
    }
  }

  toggleBreakpoint(line: number) {
    if (line < 1) return;
    const on = !this.breakpoints.includes(line);
    if (on) {
      this.breakpoints = [...this.breakpoints, line].sort((a, b) => a - b);
    } else {
      this.breakpoints = this.breakpoints.filter((n) => n !== line);
    }
    // The debugger reads commands only while paused; new breakpoints apply from there on.
    if (this.debugPaused) {
      void this.sendDebug(`${on ? 'b' : 'd'} ${line}`);
    }
  }

  // handleDebugLine consumes the "@debug" / "@eval" lines of moon -debugjson.
  handleDebugLine(line: string): boolean {
    const at = line.indexOf('@');
    if (at < 0) return false;
    const body = line.slice(at);
    if (body.startsWith('@debug ')) {
      let st: DebugState;
      try {
        st = JSON.parse(body.slice(7));
      } catch {
        return false;
      }
      this.debugState = st;
      this.debugPaused = true;
      this.debugSession = true;
      if (this.activeSidebarTab !== 'debug' || this.sidebarCollapsed) {
        this.activeSidebarTab = 'debug';
        this.sidebarCollapsed = false;
      }
      this.revealLine(st.line, false);
      const why = st.reason === 'assert' ? 'Assert failed' : st.reason === 'debugbreak' ? 'DebugBreak' : st.reason === 'step' ? 'Step' : 'Breakpoint';
      this.addLog({
        id: Math.random().toString(36).substring(7),
        type: st.reason === 'assert' ? 'error' : 'system',
        text: `${why}: paused at ${st.file}:${st.line}${st.stack?.[0] ? ` in ${st.stack[0].name}` : ''}${st.message ? ` — ${st.message}` : ''}`,
        time: new Date().toLocaleTimeString(),
        line: st.line
      });
      this.refreshWatches();
      return true;
    }
    if (body.startsWith('@eval ')) {
      try {
        const res = JSON.parse(body.slice(6));
        for (const w of this.watches) {
          if (w.expr === res.expr) {
            w.value = res.value;
            w.error = res.error;
          }
        }
      } catch {}
      return true;
    }
    return body.startsWith('@stack ');
  }

  revealLine(line: number, focus = true) {
    if (!line || line < 1) return;
    const editor = (window as any).__monacoEditor;
    if (editor) {
      editor.revealLineInCenter(line);
      editor.setPosition({ lineNumber: line, column: 1 });
      if (focus) editor.focus();
    }
  }

  endDebugSession() {
    this.debugPaused = false;
    this.debugSession = false;
    this.debugState = null;
  }

  // Continue, step, and quit resume the program, so the panes go stale until the next pause.
  debugContinue() { return this.sendDebug('c'); }
  debugStepOver() { return this.sendDebug('n'); }
  debugStepInto() { return this.sendDebug('s'); }
  debugStepOut() { return this.sendDebug('o'); }
  debugQuit() { return this.sendDebug('q'); }

  refreshWatches() {
    if (!this.debugPaused) return;
    for (const w of this.watches) {
      void AppAPI.debugCommand(`p ${w.expr}`).catch(() => {});
    }
  }

  addWatch(expr: string) {
    expr = expr.trim().replace(/[\r\n]+/g, ' ');
    if (!expr || this.watches.some((w) => w.expr === expr)) return;
    this.watches = [...this.watches, { expr }];
    if (this.debugPaused) {
      void AppAPI.debugCommand(`p ${expr}`).catch(() => {});
    }
  }

  removeWatch(expr: string) {
    this.watches = this.watches.filter((w) => w.expr !== expr);
  }

  async sendDebug(cmd: string) {
    // Resume before sending: the next "@debug" pause can arrive before the request returns.
    if (['c', 's', 'n', 'o', 'q'].includes(cmd)) {
      this.debugPaused = false;
    }
    try {
      await AppAPI.debugCommand(cmd);
    } catch (err: any) {
      this.addLog({
        id: Math.random().toString(36).substring(7),
        type: 'error',
        text: `Debug: ${err.message || err}`,
        time: new Date().toLocaleTimeString()
      });
    }
  }

  async runActiveProgram(debug = false) {
    if (!this.activeTab) return;
    this.isOutputCollapsed = false;
    this.addLog({
      id: Math.random().toString(36).substring(7),
      type: 'system',
      text: `Preparing to run ${this.activeTab.name}...`,
      time: new Date().toLocaleTimeString()
    });

    try {
      const path = this.activeTab.path;
      if (
        path &&
        !this.activeTab.isDirty &&
        !path.startsWith('temp://') &&
        !path.startsWith('example://')
      ) {
        try {
          const res = await AppAPI.openFile(path);
          if (res?.content) {
            this.activeTab.content = res.content;
          }
        } catch {
          // run the buffer we already have
        }
      }
      if (debug) {
        await AppAPI.setBreakpoints(this.breakpoints.join(','));
      }
      this.debugSession = debug;
      await AppAPI.runProgram(this.activeTab.content, this.activeTab.path, debug);
    } catch (err: any) {
      this.addLog({
        id: Math.random().toString(36).substring(7),
        type: 'error',
        text: `Execution error: ${err.message || err}`,
        time: new Date().toLocaleTimeString()
      });
    }
  }

  async stopProgram() {
    this.isRunning = false;
    this.endDebugSession();
    try {
      await AppAPI.stopProgram();
    } catch (err) {
      console.error('Stop error:', err);
    }
  }

  addLog(msg: ConsoleMessage) {
    this.consoleLogs.push(msg);
    if (this.consoleLogs.length > 1000) {
      this.consoleLogs.shift();
    }
  }

  clearLogs() {
    this.consoleLogs = [];
  }
}

export const editorStore = new EditorStore();
