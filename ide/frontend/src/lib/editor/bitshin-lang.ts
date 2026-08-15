import * as monaco from 'monaco-editor';

export const LANGUAGE_ID = 'bitshinbasic';

export const monarchLanguage: monaco.languages.IMonarchLanguage = {
  defaultToken: '',
  tokenPostfix: '.bb',
  ignoreCase: true,

  keywords: [
    'if', 'then', 'else', 'elseif', 'endif',
    'while', 'wend',
    'for', 'to', 'step', 'next',
    'repeat', 'until', 'forever',
    'select', 'case', 'default', 'end select',
    'function', 'end function', 'return',
    'type', 'field', 'end type',
    'const', 'dim', 'data', 'read', 'restore',
    'include', 'import', 'as',
    'and', 'or', 'not', 'xor', 'mod', 'shl', 'shr', 'sar',
    'true', 'false', 'yes', 'no', 'null',
    'end', 'exit', 'goto', 'gosub',
    'new', 'delete', 'first', 'last', 'after', 'before', 'insert', 'each'
  ],

  typeKeywords: [
    'int', 'float', 'string', 'vec3', 'handle', 'list', 'map', 'entity'
  ],

  operators: [
    '=', '<>', '<', '<=', '>', '>=', '=<', '=>', '><',
    '+', '-', '*', '/', '^', ':',
    'and', 'or', 'not', 'xor', 'mod'
  ],

  symbols: /[=><!~?:&|+\-*\/\^%]+/,

  tokenizer: {
    root: [
      // Comments
      [/;.*$/, 'comment'],
      [/\/\/.*$/, 'comment'],
      [/'.*$/, 'comment'],

      // Hex literals $FF0000
      [/\$[0-9a-fA-F]+/, 'number.hex'],

      // Numbers
      [/\d*\.\d+([eE][\-+]?\d+)?/, 'number.float'],
      [/\d+/, 'number'],

      // Strings
      [/"([^"\\]|\\.)*"/, 'string'],

      // Identifiers and keywords
      [/[a-zA-Z_]\w*[\$#%]?/, {
        cases: {
          '@keywords': 'keyword',
          '@typeKeywords': 'type',
          '@default': 'identifier'
        }
      }],

      // Delimiters and operators
      [/[{}()\[\]]/, '@brackets'],
      [/@symbols/, {
        cases: {
          '@operators': 'operator',
          '@default': ''
        }
      }],

      // Whitespace
      [/\s+/, 'white'],
    ]
  }
};

export const languageConfiguration: monaco.languages.LanguageConfiguration = {
  comments: {
    lineComment: ';',
  },
  brackets: [
    ['(', ')'],
    ['[', ']'],
    ['{', '}']
  ],
  autoClosingPairs: [
    { open: '(', close: ')' },
    { open: '[', close: ']' },
    { open: '{', close: '}' },
    { open: '"', close: '"', notIn: ['string'] }
  ],
  surroundingPairs: [
    { open: '(', close: ')' },
    { open: '[', close: ']' },
    { open: '{', close: '}' },
    { open: '"', close: '"' }
  ],
  folding: {
    markers: {
      start: new RegExp('^\\s*(Function|Type|While|For|Repeat|Select)\\b', 'i'),
      end: new RegExp('^\\s*(End\\s+Function|End\\s+Type|Wend|Next|Until|Forever|End\\s+Select|EndIf)\\b', 'i')
    }
  }
};

export function registerBitShinLanguage() {
  if (monaco.languages.getLanguages().some(l => l.id === LANGUAGE_ID)) {
    return;
  }

  monaco.languages.register({
    id: LANGUAGE_ID,
    extensions: ['.bb', '.basic', '.b3d'],
    aliases: ['BitShin BASIC', 'Blitz3D', 'bb']
  });

  monaco.languages.setMonarchTokensProvider(LANGUAGE_ID, monarchLanguage);
  monaco.languages.setLanguageConfiguration(LANGUAGE_ID, languageConfiguration);

  // Define Custom Themes
  monaco.editor.defineTheme('bitshin-dark', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'keyword', foreground: '569cd6', fontStyle: 'bold' },
      { token: 'type', foreground: '4ec9b0' },
      { token: 'comment', foreground: '6a9955', fontStyle: 'italic' },
      { token: 'string', foreground: 'ce9178' },
      { token: 'number', foreground: 'b5cea8' },
      { token: 'number.hex', foreground: 'dcdcaa' },
      { token: 'number.float', foreground: 'b5cea8' },
      { token: 'operator', foreground: 'd4d4d4' },
      { token: 'identifier', foreground: '9cdcfe' }
    ],
    colors: {
      'editor.background': '#0f131a',
      'editor.foreground': '#e2e8f0',
      'editor.lineHighlightBackground': '#18202f',
      'editorCursor.foreground': '#38bdf8',
      'editorLineNumber.foreground': '#475569',
      'editorLineNumber.activeForeground': '#94a3b8',
      'editor.selectionBackground': '#1e3a8a88',
      'editor.inactiveSelectionBackground': '#1e293b66'
    }
  });

  monaco.editor.defineTheme('blitz-classic', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'keyword', foreground: 'ffff00', fontStyle: 'bold' },
      { token: 'type', foreground: '00ffff' },
      { token: 'comment', foreground: '00ff00', fontStyle: 'italic' },
      { token: 'string', foreground: '00ffff' },
      { token: 'number', foreground: 'ff8800' },
      { token: 'number.hex', foreground: 'ffaa00' },
      { token: 'identifier', foreground: 'ffffff' }
    ],
    colors: {
      'editor.background': '#000044',
      'editor.foreground': '#ffffff',
      'editor.lineHighlightBackground': '#001166',
      'editorCursor.foreground': '#ffff00',
      'editorLineNumber.foreground': '#4466aa',
      'editorLineNumber.activeForeground': '#88aaff',
      'editor.selectionBackground': '#0033aa'
    }
  });

  monaco.editor.defineTheme('cyberpunk', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'keyword', foreground: 'ff007f', fontStyle: 'bold' },
      { token: 'type', foreground: '00f0ff' },
      { token: 'comment', foreground: '71717a', fontStyle: 'italic' },
      { token: 'string', foreground: 'ffe600' },
      { token: 'number', foreground: '00ff9f' },
      { token: 'number.hex', foreground: '00ff9f' },
      { token: 'identifier', foreground: 'e4e4e7' }
    ],
    colors: {
      'editor.background': '#080811',
      'editor.foreground': '#f4f4f5',
      'editor.lineHighlightBackground': '#181829',
      'editorCursor.foreground': '#ff007f',
      'editorLineNumber.foreground': '#3f3f46',
      'editorLineNumber.activeForeground': '#a1a1aa',
      'editor.selectionBackground': '#ff007f44'
    }
  });
}
