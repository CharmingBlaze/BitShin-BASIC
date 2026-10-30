//go:build windows

package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

func runDialog(script string, env []string) (string, error) {
	cmd := exec.Command("powershell", "-STA", "-NoProfile", "-Command", "-")
	cmd.Stdin = strings.NewReader(script)
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func browseOpenFile() (string, error) {
	return runDialog(`
Add-Type -AssemblyName System.Windows.Forms
$d = New-Object System.Windows.Forms.OpenFileDialog
$d.Title = "Open moonBASIC file"
$d.Filter = "moonBASIC (*.mb)|*.mb;*.basic;*.txt|Shaders (*.glsl)|*.glsl;*.vert;*.frag|All files (*.*)|*.*"
if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $d.FileName }
`, nil)
}

func browseSaveFile(defaultName string) (string, error) {
	return runDialog(`
Add-Type -AssemblyName System.Windows.Forms
$d = New-Object System.Windows.Forms.SaveFileDialog
$d.Title = "Save moonBASIC file"
$d.Filter = "moonBASIC (*.mb)|*.mb|All files (*.*)|*.*"
$d.FileName = $env:MB_FILE
if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $d.FileName }
`, []string{"MB_FILE=" + defaultName})
}

func browseDirectory() (string, error) {
	return runDialog(`
Add-Type -AssemblyName System.Windows.Forms
$d = New-Object System.Windows.Forms.FolderBrowserDialog
$d.Description = "Select project directory"
if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $d.SelectedPath }
`, nil)
}
