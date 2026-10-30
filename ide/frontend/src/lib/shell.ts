// Desktop shell. The window is Neutralino. The Go process serves this page
// and the /api and /events endpoints. Wails is not part of the IDE.

type Listener = (data: any) => void

const listeners = new Map<string, Set<Listener>>()
const eventNames = ['run:start', 'run:stdout', 'run:stderr', 'run:exit', 'run:stopped', 'lsp:message']
let source: EventSource | null = null

function ensureEvents() {
  if (source || typeof EventSource === 'undefined') return
  source = new EventSource('/events')
  for (const name of eventNames) {
    source.addEventListener(name, (ev) => {
      const raw = (ev as MessageEvent).data as string
      let data: any = raw
      try {
        data = JSON.parse(raw)
      } catch {
        data = raw
      }
      listeners.get(name)?.forEach((fn) => fn(data))
    })
  }
}

async function call(method: string, body?: object) {
  const res = await fetch('/api/' + method, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body ?? {}),
  })
  const data = await res.json()
  if (!data.ok) {
    throw new Error(data.error || method + ' failed')
  }
  return data.result
}

function installBackend() {
  window.runtime = {
    EventsOn(eventName, callback) {
      ensureEvents()
      if (!listeners.has(eventName)) listeners.set(eventName, new Set())
      listeners.get(eventName)!.add(callback)
      return () => listeners.get(eventName)?.delete(callback)
    },
    WindowMinimise() {
      void call('WindowMinimise')
    },
    WindowToggleMaximise() {
      void call('WindowToggleMaximise')
    },
    WindowStartResize(edge: string) {
      void call('WindowStartResize', { edge })
    },
    WindowStartMove() {
      void call('WindowStartMove')
    },
    Quit() {
      void call('Quit')
    },
  }
  window.go = {
    main: {
      App: {
        OpenFile: (path: string) => call('OpenFile', { path }),
        SaveFile: (path: string, content: string) => call('SaveFile', { path, content }),
        SelectOpenFile: () => call('SelectOpenFile'),
        SelectSaveFile: (defaultName: string, content: string) => call('SelectSaveFile', { defaultName, content }),
        SelectProjectDirectory: () => call('SelectProjectDirectory'),
        GetProjectTree: (rootPath: string) => call('GetProjectTree', { rootPath }),
        SetBreakpoints: (lines: string) => call('SetBreakpoints', { lines }),
        DebugCommand: (line: string) => call('DebugCommand', { line }),
        RunProgram: (code: string, filePath: string, debug: boolean) => call('RunProgram', { code, filePath, debug }),
        StopProgram: () => call('StopProgram'),
        BuildExecutable: (filePath: string, outputDir: string, targetOS: string) => call('BuildExecutable', { filePath, outputDir, targetOS }),
        GetSettings: () => call('GetSettings'),
        SaveSettings: (settings: any) => call('SaveSettings', { settings }),
        GetRepoRoot: () => call('GetRepoRoot'),
        SendLSP: (body: string) => call('SendLSP', { body }),
        IsLspRunning: () => call('IsLspRunning'),
      },
    },
  }
}

installBackend()

const neu = (window as any).Neutralino
if (neu?.init) {
  neu.init()
}
