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
      Quit?: () => void;
    };
  }
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

  async runProgram(code: string, filePath: string, debug: boolean) {
    if (window.go?.main?.App?.RunProgram) {
      return await window.go.main.App.RunProgram(code, filePath, debug);
    }
    throw new Error('Runner requires BitShin IDE desktop backend');
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
    throw new Error('Builder requires BitShin IDE desktop backend');
  },

  async getSettings() {
    if (window.go?.main?.App?.GetSettings) {
      return await window.go.main.App.GetSettings();
    }
    const saved = localStorage.getItem('bitshin_settings');
    if (saved) {
      try { return JSON.parse(saved); } catch {}
    }
    return {
      theme: 'bitshin-dark',
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
    localStorage.setItem('bitshin_settings', JSON.stringify(settings));
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
    throw new Error('Language server requires BitShin IDE desktop backend');
  },

  async isLspRunning() {
    if (window.go?.main?.App?.IsLspRunning) {
      return await window.go.main.App.IsLspRunning();
    }
    return false;
  }
};
