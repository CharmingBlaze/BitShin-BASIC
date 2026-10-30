const vscode = require("vscode");
const { spawn } = require("child_process");
const fs = require("fs");
const path = require("path");

let child = null;
let buf = Buffer.alloc(0);
let nextId = 1;
const pending = new Map();
let diagCol = null;

function activate(context) {
  diagCol = vscode.languages.createDiagnosticCollection("moonbasic");
  context.subscriptions.push(diagCol);
  startServer(context);

  context.subscriptions.push(
    vscode.workspace.onDidOpenTextDocument((d) => {
      if (d.languageId === "moonbasic") didOpen(d);
    }),
    vscode.workspace.onDidChangeTextDocument((e) => {
      if (e.document.languageId === "moonbasic") didChange(e.document);
    }),
    vscode.workspace.onDidCloseTextDocument((d) => {
      if (d.languageId === "moonbasic") {
        notify("textDocument/didClose", { textDocument: { uri: d.uri.toString() } });
        diagCol.delete(d.uri);
      }
    }),
    vscode.languages.registerHoverProvider("moonbasic", {
      provideHover: async (doc, pos) => {
        const r = await request("textDocument/hover", docPos(doc, pos));
        if (!r || !r.contents) return null;
        const v = r.contents.value || r.contents;
        return new vscode.Hover(new vscode.MarkdownString(String(v)));
      },
    }),
    vscode.languages.registerCompletionItemProvider(
      "moonbasic",
      {
        provideCompletionItems: async (doc, pos) => {
          const r = await request("textDocument/completion", docPos(doc, pos));
          const items = (r && r.items) || r || [];
          return items.map((it) => {
            const c = new vscode.CompletionItem(it.label, it.kind || vscode.CompletionItemKind.Function);
            c.detail = it.detail;
            c.documentation = it.documentation;
            return c;
          });
        },
      },
      "."
    ),
    vscode.languages.registerDefinitionProvider("moonbasic", {
      provideDefinition: async (doc, pos) => {
        const r = await request("textDocument/definition", docPos(doc, pos));
        const locs = Array.isArray(r) ? r : r ? [r] : [];
        return locs.map((l) => new vscode.Location(vscode.Uri.parse(l.uri), toRange(l.range)));
      },
    }),
    vscode.languages.registerDocumentSymbolProvider("moonbasic", {
      provideDocumentSymbols: async (doc) => {
        const r = await request("textDocument/documentSymbol", { textDocument: { uri: doc.uri.toString() } });
        const list = r || [];
        return list.map(toSymbol);
      },
    })
  );

  for (const d of vscode.workspace.textDocuments) {
    if (d.languageId === "moonbasic") didOpen(d);
  }
}

function deactivate() {
  if (child) {
    notify("exit", {});
    child.kill();
    child = null;
  }
}

function startServer(context) {
  const cfg = vscode.workspace.getConfiguration("moonbasic");
  const folders = vscode.workspace.workspaceFolders || [];
  const root = folders.length ? folders[0].uri.fsPath : "";
  const exe = resolveServer(cfg.get("serverPath") || "", root);
  if (!exe) {
    vscode.window.showWarningMessage("moonBASIC: could not find moonsls.exe. Build with go build -o moonsls.exe ./cmd/bsls");
    return;
  }
  const args = path.basename(exe).toLowerCase().startsWith("moon") && !path.basename(exe).toLowerCase().startsWith("moonsls")
    ? ["lsp"]
    : [];
  child = spawn(exe, args, { stdio: ["pipe", "pipe", "pipe"], cwd: root || undefined });
  child.stderr.on("data", (d) => console.error(String(d)));
  child.stdout.on("data", onData);
  child.on("exit", () => {
    child = null;
  });
  request("initialize", {
    processId: process.pid,
    rootUri: root ? vscode.Uri.file(root).toString() : null,
    capabilities: {},
  }).then(() => {
    notify("initialized", {});
    for (const d of vscode.workspace.textDocuments) {
      if (d.languageId === "moonbasic") didOpen(d);
    }
  });
  context.subscriptions.push({ dispose: deactivate });
}

function resolveServer(configured, root) {
  const candidates = [];
  if (configured) candidates.push(configured.replace(/\$\{workspaceFolder\}/g, root));
  if (root) {
    candidates.push(path.join(root, "moonsls.exe"), path.join(root, "moonsls"));
    candidates.push(path.join(root, "moon.exe"), path.join(root, "moon"));
  }
  for (const p of candidates) {
    if (p && fs.existsSync(p)) return p;
  }
  return configured || (process.platform === "win32" ? "moonsls.exe" : "moonsls");
}

function didOpen(doc) {
  notify("textDocument/didOpen", {
    textDocument: {
      uri: doc.uri.toString(),
      languageId: "moonbasic",
      version: doc.version,
      text: doc.getText(),
    },
  });
}

function didChange(doc) {
  notify("textDocument/didChange", {
    textDocument: { uri: doc.uri.toString(), version: doc.version },
    contentChanges: [{ text: doc.getText() }],
  });
}

function docPos(doc, pos) {
  return {
    textDocument: { uri: doc.uri.toString() },
    position: { line: pos.line, character: pos.character },
  };
}

function toRange(r) {
  if (!r) return new vscode.Range(0, 0, 0, 0);
  return new vscode.Range(r.start.line, r.start.character, r.end.line, r.end.character);
}

function toSymbol(s) {
  const kind = s.kind || vscode.SymbolKind.Function;
  const sel = toRange(s.selectionRange || s.range);
  const rng = toRange(s.range);
  const out = new vscode.DocumentSymbol(s.name, "", kind, rng, sel);
  if (s.children) out.children = s.children.map(toSymbol);
  return out;
}

function notify(method, params) {
  send({ jsonrpc: "2.0", method, params });
}

function request(method, params) {
  const id = nextId++;
  return new Promise((resolve) => {
    pending.set(id, resolve);
    send({ jsonrpc: "2.0", id, method, params });
    setTimeout(() => {
      if (pending.has(id)) {
        pending.delete(id);
        resolve(null);
      }
    }, 8000);
  });
}

function send(obj) {
  if (!child || !child.stdin.writable) return;
  const json = JSON.stringify(obj);
  child.stdin.write("Content-Length: " + Buffer.byteLength(json) + "\r\n\r\n" + json);
}

function onData(chunk) {
  buf = Buffer.concat([buf, chunk]);
  while (true) {
    const headerEnd = buf.indexOf("\r\n\r\n");
    if (headerEnd < 0) return;
    const header = buf.slice(0, headerEnd).toString("utf8");
    const m = /Content-Length:\s*(\d+)/i.exec(header);
    if (!m) {
      buf = buf.slice(headerEnd + 4);
      continue;
    }
    const len = parseInt(m[1], 10);
    const start = headerEnd + 4;
    if (buf.length < start + len) return;
    const body = buf.slice(start, start + len).toString("utf8");
    buf = buf.slice(start + len);
    handleMessage(JSON.parse(body));
  }
}

function handleMessage(msg) {
  if (msg.method === "textDocument/publishDiagnostics" && msg.params) {
    const uri = vscode.Uri.parse(msg.params.uri);
    const list = (msg.params.diagnostics || []).map((d) => {
      const r = toRange(d.range);
      const sev = d.severity === 1 ? vscode.DiagnosticSeverity.Error : vscode.DiagnosticSeverity.Warning;
      return new vscode.Diagnostic(r, d.message, sev);
    });
    diagCol.set(uri, list);
    return;
  }
  if (msg.id != null && pending.has(msg.id)) {
    const fn = pending.get(msg.id);
    pending.delete(msg.id);
    fn(msg.result);
  }
}

module.exports = { activate, deactivate };
