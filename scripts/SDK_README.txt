moonBASIC for Windows
=====================

Keep every file in this folder together. moon.exe needs the two DLLs beside it.

  moonbasic.exe     The IDE: editor, run (F5), debugger, command reference, game packager
  moon.exe          The compiler and runtime, from a command prompt
  moonsls.exe       Language server, for VS Code and other editors (see editors\vscode)
  docs\GUIDE.md     Start here: the language, the frame loop, and each engine system
  examples\         Small programs, one engine feature each
  games\            Two complete games with source: Greenwood and The Black Veil

From a command prompt in this folder:

  moon examples\spinning_cube.mb               run a program
  cd games\hunter
  ..\..\moon main.mb                           run Greenwood from its source
  ..\..\moon build main.mb -o C:\Out\Greenwood  package it: an .exe, its scripts, a .pak

A packaged game is a folder. Zip it and send it; whoever gets it double-clicks
the .exe. No Go, compiler or install is needed, here or there.

Requires Windows 10 or 11 (64-bit) and a graphics driver with OpenGL 3.3.
