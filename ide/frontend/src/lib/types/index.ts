export interface EditorTab {
  id: string;
  name: string;
  path: string;
  content: string;
  isDirty: boolean;
  isTemporary?: boolean;
}

export interface FileNode {
  name: string;
  path: string;
  isDir: boolean;
  children?: FileNode[];
  size?: number;
}

export interface CommandItem {
  name: string;
  syntax: string;
  description: string;
  category: string;
}

export interface CommandCategory {
  name: string;
  description?: string;
  commands: CommandItem[];
}

export interface ExampleItem {
  filename: string;
  name: string;
  description: string;
  code: string;
}

export interface ConsoleMessage {
  id: string;
  type: 'stdout' | 'stderr' | 'system' | 'error' | 'success';
  text: string;
  time: string;
  file?: string;
  line?: number;
  col?: number;
}

export interface UserSettings {
  theme: string;
  fontSize: number;
  tabSize: number;
  minimap: boolean;
  wordWrap: 'off' | 'on' | 'wordWrapColumn' | 'bounded';
  autoSave: boolean;
  targetOS: string;
  outputHeight: number;
  sidebarWidth: number;
  recentProjects?: string[];
}

export interface DebugFrame {
  name: string;
  file: string;
  line: number;
}

export interface DebugVar {
  name: string;
  kind: string;
  value: string;
}

// One "@debug {json}" line from moon -debugjson.
export interface DebugState {
  reason: string;
  message?: string;
  file: string;
  line: number;
  depth: number;
  stack: DebugFrame[];
  locals: DebugVar[];
  globals: DebugVar[];
}

export interface WatchItem {
  expr: string;
  value?: string;
  error?: string;
}

export interface SymbolItem {
  name: string;
  type: 'function' | 'type' | 'const' | 'label';
  line: number;
  signature?: string;
}

// moon.json at a project root (see docs/STUDIO.md).
export interface Project {
  name: string;
  main: string;
  window: { width: number; height: number; title: string };
  quality: string;
  scenes: string;
  assets: string;
  build: { outputDir: string; icon: string; version: string; company: string };
}

// A loaded project with absolute paths, from LoadProject / CreateProject / SaveProject.
export interface ProjectInfo {
  root: string;
  file: string;
  mainPath: string;
  scenesPath: string;
  assetsPath: string;
  outputPath: string;
  project: Project;
  warnings?: string[];
}

export interface RecentProject {
  root: string;
  name: string;
  exists: boolean;
}

export interface ProjectTemplate {
  id: string;
  title: string;
  description: string;
  is2D: boolean;
}
