// Safe wrapper for Wails Go Backend bindings & events

declare global {
  interface Window {
    go?: {
      main?: {
        App?: any;
      };
    };
    runtime?: {
      EventsOn?: (eventName: string, callback: (...args: any[]) => void) => () => void;
      EventsOff?: (eventName: string, ...additionalEvents: string[]) => void;
      EventsEmit?: (eventName: string, ...optionalData: any[]) => void;
      WindowMinimise?: () => void;
      WindowToggleMaximise?: () => void;
      WindowStartResize?: (edge: string) => void;
      WindowStartMove?: () => void;
      Quit?: () => void;
    };
  }
}

import type { Project, ProjectInfo, ProjectTemplate, RecentProject } from './types';

// backend returns a desktop-backend method, or one that throws in a plain browser.
function backend(name: string): (...args: any[]) => Promise<any> {
  const fn = window.go?.main?.App?.[name];
  if (fn) return fn;
  return async () => {
    throw new Error('Projects require the moonBASIC IDE desktop backend');
  };
}

export const isWails = typeof window !== 'undefined' && !!window.go?.main?.App;

export const AppAPI = {
  async openFile(path: string) {
    if (window.go?.main?.App?.OpenFile) {
      return await window.go.main.App.OpenFile(path);
    }
    throw new Error('Running in web standalone mode');
  },

  async saveFile(path: string, content: string) {
    if (window.go?.main?.App?.SaveFile) {
      return await window.go.main.App.SaveFile(path, content);
    }
    localStorage.setItem(`file:${path}`, content);
    return null;
  },

  async selectOpenFile() {
    if (window.go?.main?.App?.SelectOpenFile) {
      return await window.go.main.App.SelectOpenFile();
    }
    throw new Error('Not running in Wails');
  },

  async selectSaveFile(defaultName: string, content: string) {
    if (window.go?.main?.App?.SelectSaveFile) {
      return await window.go.main.App.SelectSaveFile(defaultName, content);
    }
    throw new Error('Not running in Wails');
  },

  async selectProjectDirectory() {
    if (window.go?.main?.App?.SelectProjectDirectory) {
      return await window.go.main.App.SelectProjectDirectory();
    }
    return '';
  },

  async getProjectTree(rootPath: string) {
    if (window.go?.main?.App?.GetProjectTree) {
      return await window.go.main.App.GetProjectTree(rootPath);
    }
    return null;
  },

  async setBreakpoints(lines: string) {
    if (window.go?.main?.App?.SetBreakpoints) {
      return await window.go.main.App.SetBreakpoints(lines);
    }
    return null;
  },

  async debugCommand(line: string) {
    if (window.go?.main?.App?.DebugCommand) {
      return await window.go.main.App.DebugCommand(line);
    }
    return null;
  },

  async runProgram(code: string, filePath: string, debug: boolean) {
    if (window.go?.main?.App?.RunProgram) {
      return await window.go.main.App.RunProgram(code, filePath, debug);
    }
    throw new Error('Runner requires moonBASIC IDE desktop backend');
  },

  async stopProgram() {
    if (window.go?.main?.App?.StopProgram) {
      return await window.go.main.App.StopProgram();
    }
    return null;
  },

  async buildExecutable(filePath: string, outputDir: string, targetOS: string) {
    if (window.go?.main?.App?.BuildExecutable) {
      return await window.go.main.App.BuildExecutable(filePath, outputDir, targetOS);
    }
    throw new Error('Builder requires moonBASIC IDE desktop backend');
  },

  async getSettings() {
    if (window.go?.main?.App?.GetSettings) {
      return await window.go.main.App.GetSettings();
    }
    const saved = localStorage.getItem('moonbasic_settings');
    if (saved) {
      try { return JSON.parse(saved); } catch {}
    }
    return {
      theme: 'moonbasic-dark',
      fontSize: 14,
      tabSize: 4,
      minimap: true,
      wordWrap: 'off',
      autoSave: true,
      targetOS: 'windows',
      outputHeight: 220,
      sidebarWidth: 320
    };
  },

  async saveSettings(settings: any) {
    if (window.go?.main?.App?.SaveSettings) {
      return await window.go.main.App.SaveSettings(settings);
    }
    localStorage.setItem('moonbasic_settings', JSON.stringify(settings));
  },

  async getRepoRoot() {
    if (window.go?.main?.App?.GetRepoRoot) {
      return await window.go.main.App.GetRepoRoot();
    }
    return '';
  },

  async sendLsp(body: string) {
    if (window.go?.main?.App?.SendLSP) {
      return await window.go.main.App.SendLSP(body);
    }
    throw new Error('Language server requires moonBASIC IDE desktop backend');
  },

  async isLspRunning() {
    if (window.go?.main?.App?.IsLspRunning) {
      return await window.go.main.App.IsLspRunning();
    }
    return false;
  },

  async loadProject(path: string): Promise<ProjectInfo> {
    return await backend('LoadProject')(path);
  },

  async saveProject(root: string, project: Project): Promise<ProjectInfo> {
    return await backend('SaveProject')(root, project);
  },

  async initProject(root: string): Promise<ProjectInfo> {
    return await backend('InitProject')(root);
  },

  async createProject(parent: string, name: string, template: string): Promise<ProjectInfo> {
    return await backend('CreateProject')(parent, name, template);
  },

  async listProjectTemplates(): Promise<ProjectTemplate[]> {
    return (await backend('ListProjectTemplates')()) || [];
  },

  async defaultProjectsDir(): Promise<string> {
    return (await backend('DefaultProjectsDir')()) || '';
  },

  async getRecentProjects(): Promise<RecentProject[]> {
    return (await backend('GetRecentProjects')()) || [];
  },

  async removeRecentProject(path: string) {
    return await backend('RemoveRecentProject')(path);
  },

  async createScene(path: string): Promise<string> {
    return await backend('CreateScene')(path);
  },

  async openSceneEditor(file: string, root: string): Promise<{ file: string; reused: boolean; command: string }> {
    return await backend('OpenSceneEditor')(file, root);
  }
};
