type LspMessage = {
  jsonrpc?: string;
  id?: number;
  method?: string;
  params?: any;
  result?: any;
  error?: { code: number; message: string };
};

type Pending = {
  resolve: (value: any) => void;
  reject: (err: Error) => void;
};

let nextId = 1;
const pending = new Map<number, Pending>();
let started = false;
export let lspReady = false;
const openVersions = new Map<string, number>();
const diagnosticListeners = new Set<(uri: string, diagnostics: any[]) => void>();

export function onLspDiagnostics(fn: (uri: string, diagnostics: any[]) => void) {
  diagnosticListeners.add(fn);
  return () => diagnosticListeners.delete(fn);
}

export async function startLspClient(): Promise<boolean> {
  if (started) return lspReady;
  started = true;
  if (!window.runtime?.EventsOn || !window.go?.main?.App?.SendLSP) {
    return false;
  }
  const running = await window.go.main.App.IsLspRunning();
  if (!running) {
    return false;
  }
  window.runtime.EventsOn('lsp:message', handleMessage);
  let root = '';
  try {
    root = await window.go.main.App.GetRepoRoot();
  } catch {
    root = '';
  }
  const result = await request('initialize', {
    processId: null,
    rootUri: root ? pathToUri(root) : null,
    capabilities: {}
  });
  if (!result) {
    return false;
  }
  notify('initialized', {});
  lspReady = true;
  return true;
}

export function pathToUri(filePath: string): string {
  const norm = filePath.replace(/\\/g, '/');
  if (/^[A-Za-z]:/.test(norm)) {
    return 'file:///' + encodeURI(norm);
  }
  if (norm.startsWith('/')) {
    return 'file://' + encodeURI(norm);
  }
  return 'file:///' + encodeURI(norm);
}

export function tabToUri(tabPath: string, tabName: string, repoRoot: string): string {
  if (!tabPath || tabPath.startsWith('temp://') || tabPath.startsWith('example://')) {
    const scratch = repoRoot ? `${repoRoot.replace(/\\/g, '/')}/scratch/${tabName}` : tabName;
    return pathToUri(scratch);
  }
  return pathToUri(tabPath);
}

export function lspDidOpen(uri: string, text: string) {
  if (!lspReady) return;
  const version = (openVersions.get(uri) ?? 0) + 1;
  openVersions.set(uri, version);
  notify('textDocument/didOpen', {
    textDocument: {
      uri,
      languageId: 'bitshinbasic',
      version,
      text
    }
  });
}

export function lspDidChange(uri: string, text: string) {
  if (!lspReady) return;
  const version = (openVersions.get(uri) ?? 0) + 1;
  openVersions.set(uri, version);
  notify('textDocument/didChange', {
    textDocument: { uri, version },
    contentChanges: [{ text }]
  });
}

export function lspDidClose(uri: string) {
  if (!lspReady) return;
  openVersions.delete(uri);
  notify('textDocument/didClose', {
    textDocument: { uri }
  });
}

export function lspRequest(method: string, params: any): Promise<any> {
  return request(method, params);
}

function notify(method: string, params: any) {
  void send({ jsonrpc: '2.0', method, params });
}

function request(method: string, params: any): Promise<any> {
  const id = nextId++;
  return new Promise((resolve, reject) => {
    pending.set(id, { resolve, reject });
    void send({ jsonrpc: '2.0', id, method, params });
    setTimeout(() => {
      if (pending.has(id)) {
        pending.delete(id);
        resolve(null);
      }
    }, 8000);
  });
}

async function send(obj: LspMessage) {
  if (!window.go?.main?.App?.SendLSP) return;
  try {
    await window.go.main.App.SendLSP(JSON.stringify(obj));
  } catch (err) {
    console.error('LSP send failed:', err);
  }
}

function handleMessage(raw: string) {
  let msg: LspMessage;
  try {
    msg = JSON.parse(raw);
  } catch {
    return;
  }
  if (msg.method === 'textDocument/publishDiagnostics' && msg.params) {
    for (const fn of diagnosticListeners) {
      fn(msg.params.uri, msg.params.diagnostics || []);
    }
    return;
  }
  if (msg.id != null && pending.has(msg.id)) {
    const p = pending.get(msg.id)!;
    pending.delete(msg.id);
    if (msg.error) {
      p.resolve(null);
      return;
    }
    p.resolve(msg.result);
  }
}
