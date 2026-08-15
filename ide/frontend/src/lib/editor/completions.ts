import * as monaco from 'monaco-editor';
import commandsData from '../data/commands.json';
import { LANGUAGE_ID } from './bitshin-lang';
import { lspReady, lspRequest } from './lspClient';

export function registerCompletions() {
  monaco.languages.registerCompletionItemProvider(LANGUAGE_ID, {
    triggerCharacters: [' ', '.', '(', ','],
    provideCompletionItems: async (model, position) => {
      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn
      };

      const suggestions: monaco.languages.CompletionItem[] = [];

      if (lspReady) {
        const result = await lspRequest('textDocument/completion', {
          textDocument: { uri: model.uri.toString() },
          position: { line: position.lineNumber - 1, character: position.column - 1 }
        });
        const items = (result && result.items) || result || [];
        if (Array.isArray(items) && items.length > 0) {
          for (const it of items) {
            suggestions.push({
              label: it.label,
              kind: it.kind || monaco.languages.CompletionItemKind.Function,
              detail: it.detail,
              documentation: it.documentation,
              insertText: it.label,
              range
            });
          }
          return { suggestions };
        }
      }

      // 1. Builtin Commands from documentation
      for (const cmd of commandsData.commands) {
        let insertSnippet = cmd.syntax;
        // Transform `Function arg1, arg2` into snippet if applicable
        if (insertSnippet.includes('(') && insertSnippet.includes(')')) {
          const name = cmd.name;
          insertSnippet = `${name}($0)`;
        }

        suggestions.push({
          label: cmd.name,
          kind: monaco.languages.CompletionItemKind.Function,
          documentation: {
            value: `**${cmd.syntax}**\n\n*Category: ${cmd.category}*\n\n${cmd.description}`
          },
          detail: cmd.syntax,
          insertText: cmd.name,
          range: range
        });
      }

      // 2. Language Keywords
      const keywords = [
        'If', 'Then', 'Else', 'ElseIf', 'EndIf',
        'While', 'Wend', 'For', 'To', 'Step', 'Next',
        'Repeat', 'Until', 'Forever', 'Select', 'Case', 'Default',
        'Function', 'End Function', 'Return',
        'Type', 'Field', 'End Type', 'Const', 'Dim',
        'Include', 'Import', 'True', 'False', 'Null', 'End'
      ];

      for (const kw of keywords) {
        suggestions.push({
          label: kw,
          kind: monaco.languages.CompletionItemKind.Keyword,
          insertText: kw,
          range: range
        });
      }

      // 3. Common Snippets
      suggestions.push({
        label: 'Graphics3D Loop',
        kind: monaco.languages.CompletionItemKind.Snippet,
        insertText: [
          'Graphics3D(1280, 720, 0, 2)',
          'camera = CreateCamera()',
          'light = CreateLight()',
          'cube = CreateCube()',
          'PositionEntity(cube, 0, 0, 5)',
          '',
          'While Not KeyDown(1)',
          '    dt# = DeltaTime() * 60',
          '    TurnEntity(cube, 0.5 * dt, 0.8 * dt, 0)',
          '    RenderWorld',
          '    Flip',
          'Wend',
          'End'
        ].join('\n'),
        insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
        documentation: 'Standard 3D game loop template',
        detail: 'Snippet: 3D Scene Game Loop',
        range: range
      });

      suggestions.push({
        label: 'Graphics2D Loop',
        kind: monaco.languages.CompletionItemKind.Snippet,
        insertText: [
          'Graphics2D(800, 600)',
          'SetWindowTitle("2D Game")',
          '',
          'While Not KeyDown(1)',
          '    Cls',
          '    Color(255, 255, 255)',
          '    Text(10, 10, "Hello BitShin BASIC 2D")',
          '    Flip',
          'Wend',
          'End'
        ].join('\n'),
        insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
        documentation: 'Standard 2D game loop template',
        detail: 'Snippet: 2D Game Loop',
        range: range
      });

      suggestions.push({
        label: 'Function template',
        kind: monaco.languages.CompletionItemKind.Snippet,
        insertText: [
          'Function ${1:FunctionName}(${2:args})',
          '    ${0}',
          '    Return ${3:0}',
          'End Function'
        ].join('\n'),
        insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
        documentation: 'Create a new function',
        detail: 'Snippet: Function',
        range: range
      });

      suggestions.push({
        label: 'Type definition',
        kind: monaco.languages.CompletionItemKind.Snippet,
        insertText: [
          'Type ${1:TypeName}',
          '    Field ${2:x#}, ${3:y#}, ${4:z#}',
          '    Field ${5:entity}',
          'End Type'
        ].join('\n'),
        insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
        documentation: 'Create a custom Type struct',
        detail: 'Snippet: Type',
        range: range
      });

      // 4. Scan active file for user-defined functions
      const fullText = model.getValue();
      const fnRegex = /\bFunction\s+([a-zA-Z_]\w*[\$#%]?)\s*\(([^\)]*)\)/gi;
      let match;
      while ((match = fnRegex.exec(fullText)) !== null) {
        const fnName = match[1];
        const fnParams = match[2];
        suggestions.push({
          label: fnName,
          kind: monaco.languages.CompletionItemKind.Function,
          detail: `User Function: ${fnName}(${fnParams})`,
          insertText: `${fnName}($0)`,
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          range: range
        });
      }

      return { suggestions };
    }
  });
}
