import * as monaco from 'monaco-editor';
import commandsData from '../data/commands.json';
import { LANGUAGE_ID } from './bitshin-lang';

export function registerHover() {
  monaco.languages.registerHoverProvider(LANGUAGE_ID, {
    provideHover: (model, position) => {
      const word = model.getWordAtPosition(position);
      if (!word) return null;

      const lookup = word.word.toLowerCase();

      // Find in commands catalog
      const cmd = commandsData.commands.find(c => c.name.toLowerCase() === lookup);
      if (cmd) {
        return {
          range: new monaco.Range(position.lineNumber, word.startColumn, position.lineNumber, word.endColumn),
          contents: [
            { value: `### \`${cmd.syntax}\`` },
            { value: `**Category:** *${cmd.category}*` },
            { value: cmd.description }
          ]
        };
      }

      // Check for user-defined function in file
      const fullText = model.getValue();
      const fnRegex = new RegExp(`\\bFunction\\s+(${word.word}[\\$#%]?)\\s*\\(([^\\)]*)\\)`, 'i');
      const match = fnRegex.exec(fullText);
      if (match) {
        return {
          range: new monaco.Range(position.lineNumber, word.startColumn, position.lineNumber, word.endColumn),
          contents: [
            { value: `### Function \`${match[1]}(${match[2]})\`` },
            { value: `*User-defined function in current document*` }
          ]
        };
      }

      return null;
    }
  });
}
