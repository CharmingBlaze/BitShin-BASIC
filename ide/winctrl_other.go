//go:build !windows

package main

func (a *App) WindowMinimise()          {}
func (a *App) WindowToggleMaximise()    {}
func (a *App) WindowStartResize(string) {}
func (a *App) WindowStartMove()         {}
func (a *App) WindowClose()             {}

func raiseProcessWindows(int) {}

func killProcessTree(pid int) {}
