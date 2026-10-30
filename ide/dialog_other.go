//go:build !windows

package main

import "fmt"

func browseOpenFile() (string, error) {
	return "", fmt.Errorf("file dialogs are only available on Windows in this build")
}

func browseSaveFile(string) (string, error) {
	return "", fmt.Errorf("file dialogs are only available on Windows in this build")
}

func browseDirectory() (string, error) {
	return "", fmt.Errorf("file dialogs are only available on Windows in this build")
}
