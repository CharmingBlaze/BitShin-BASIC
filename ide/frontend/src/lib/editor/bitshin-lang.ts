import * as monaco from 'monaco-editor';
import commandsData from '../data/commands.json';

export const LANGUAGE_ID = 'bitshinbasic';

const KEYWORDS = [
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
  'end', 'exit', 'goto', 'gosub',
  'new', 'delete', 'first', 'last', 'after', 'before', 'insert', 'each'
];

const CONSTANTS = [
  'true', 'false', 'yes', 'no', 'null', 'pi'
];

const TYPE_KEYWORDS = [
  'int', 'float', 'string', 'vec3', 'handle', 'list', 'map', 'entity'
];

const reserved = new Set([...KEYWORDS, ...CONSTANTS, ...TYPE_KEYWORDS]);

function commandBaseName(name: string): string {
  return name.replace(/[$#%]+$/, '').toLowerCase();
}

function isCreateCommand(name: string): boolean {
  return /^(create|load|copy|make)/.test(name);
}

const catalogNames = [...new Set(
  (commandsData.commands as { name: string }[])
    .map((c) => commandBaseName(c.name))
    .filter((n) => n.length > 0 && !reserved.has(n))
)];

const createCommands = catalogNames.filter(isCreateCommand);
const actionCommands = catalogNames.filter((n) => !isCreateCommand(n));

const identifierCases = {
  '@keywords': 'keyword',
  '@constants': 'constant',
  '@typeKeywords': 'type',
  '@createCommands': 'predefined.create',
  '@actionCommands': 'predefined',
  '@default': 'identifier'
};

const callCases = {
  ...identifierCases,
  '@default': 'predefined'
};

export const monarchLanguage: monaco.languages.IMonarchLanguage = {
  defaultToken: '',
  tokenPostfix: '.bb',
  ignoreCase: true,

  keywords: KEYWORDS,
  constants: CONSTANTS,
  typeKeywords: TYPE_KEYWORDS,
  createCommands,
  actionCommands,

  operators: [
    '=', '<>', '<', '<=', '>', '>=', '=<', '=>', '><',
    '+', '-', '*', '/', '^', ':',
    'and', 'or', 'not', 'xor', 'mod'
  ],

  symbols: /[=><!~?:&|+\-*\/\^%]+/,

  tokenizer: {
    root: [
      [/;.*$/, 'comment'],
      [/\/\/.*$/, 'comment'],
      [/'.*$/, 'comment'],

      [/\$[0-9a-fA-F]+/, 'number.hex'],

      [/\d*\.\d+([eE][\-+]?\d+)?/, 'number.float'],
      [/\d+/, 'number'],

      [/"([^"\\]|\\.)*"/, 'string'],

      [/KEY_[A-Za-z0-9_]+/, 'constant'],

      // name$ / name# / name% — suffix tinted separately
      [/([a-zA-Z_]\w*)([$#%])(?=\s*\()/, [{ cases: callCases }, 'type.identifier']],
      [/([a-zA-Z_]\w*)([$#%])/, [{ cases: identifierCases }, 'type.identifier']],

      // Commands and calls (with or without parens)
      [/[a-zA-Z_]\w*(?=\s*\()/, { cases: callCases }],
      [/[a-zA-Z_]\w*/, { cases: identifierCases }],

      [/[{}()\[\]]/, 'delimiter.bracket'],
      [/[,.]/, 'delimiter'],
      [/@symbols/, {
        cases: {
          '@operators': 'operator',
          '@default': 'operator'
        }
      }],

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

  monaco.editor.defineTheme('bitshin-dark', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'keyword', foreground: 'e8c547', fontStyle: 'bold' },
      { token: 'type', foreground: '7eb8a8' },
      { token: 'type.identifier', foreground: '8eb4c4' },
      { token: 'comment', foreground: '6d6d76', fontStyle: 'italic' },
      { token: 'string', foreground: 'e0a070' },
      { token: 'number', foreground: '8fca6e' },
      { token: 'number.hex', foreground: 'b8c85a' },
      { token: 'number.float', foreground: '8fca6e' },
      { token: 'operator', foreground: 'c08090' },
      { token: 'delimiter', foreground: 'c08090' },
      { token: 'delimiter.bracket', foreground: 'c08090' },
      { token: 'identifier', foreground: 'e2dfd8' },
      { token: 'predefined', foreground: '5eb8c8' },
      { token: 'predefined.create', foreground: '5cceae' },
      { token: 'constant', foreground: 'c4a0d0' }
    ],
    colors: {
      'editor.background': '#1a1b1f',
      'editor.foreground': '#e6e4df',
      'editor.lineHighlightBackground': '#222328',
      'editorCursor.foreground': '#e8c547',
      'editorLineNumber.foreground': '#4a4a52',
      'editorLineNumber.activeForeground': '#9b9ba3',
      'editor.selectionBackground': '#3d341888',
      'editor.inactiveSelectionBackground': '#2a2b3166',
      'editorIndentGuide.background': '#2a2b31',
      'editorIndentGuide.activeBackground': '#4a4a52',
      'editorGutter.background': '#1a1b1f',
      'minimap.background': '#141518'
    }
  });

  monaco.editor.defineTheme('blitz-classic', {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'keyword', foreground: 'ffff00', fontStyle: 'bold' },
      { token: 'type', foreground: '00ffff' },
      { token: 'type.identifier', foreground: '88ddff' },
      { token: 'comment', foreground: '00ff00', fontStyle: 'italic' },
      { token: 'string', foreground: '00ffff' },
      { token: 'number', foreground: 'ff8800' },
      { token: 'number.hex', foreground: 'ffaa00' },
      { token: 'operator', foreground: 'ff6688' },
      { token: 'delimiter', foreground: 'ff6688' },
      { token: 'identifier', foreground: 'ffffff' },
      { token: 'predefined', foreground: '00ddff' },
      { token: 'predefined.create', foreground: '00ffcc' },
      { token: 'constant', foreground: 'ff99ff' }
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
      { token: 'type.identifier', foreground: '7ad4ff' },
      { token: 'comment', foreground: '71717a', fontStyle: 'italic' },
      { token: 'string', foreground: 'ffe600' },
      { token: 'number', foreground: '00ff9f' },
      { token: 'number.hex', foreground: '00ff9f' },
      { token: 'operator', foreground: 'ff6b9d' },
      { token: 'delimiter', foreground: 'ff6b9d' },
      { token: 'identifier', foreground: 'e4e4e7' },
      { token: 'predefined', foreground: '00d4ff' },
      { token: 'predefined.create', foreground: '00ffc8' },
      { token: 'constant', foreground: 'd4a0ff' }
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
