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
}

export interface SymbolItem {
  name: string;
  type: 'function' | 'type' | 'const' | 'label';
  line: number;
  signature?: string;
}
