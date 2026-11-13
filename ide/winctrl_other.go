//go:build !windows

package main

import "os/exec"

func (a *App) WindowMinimise()          {}
func (a *App) WindowToggleMaximise()    {}
func (a *App) WindowStartResize(string) {}
func (a *App) WindowStartMove()         {}
func (a *App) WindowClose()             {}

func raiseProcessWindows(int) {}

func killProcessTree(pid int) {}

func noConsole(*exec.Cmd) {}
