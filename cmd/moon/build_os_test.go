package main

import "testing"

func TestNormalizeOS(t *testing.T) {
	if normalizeOS("WIN") != "windows" {
		t.Fatal("windows")
	}
	if normalizeOS("macos") != "darwin" {
		t.Fatal("darwin")
	}
	if normalizeOS("linux") != "linux" {
		t.Fatal("linux")
	}
}
