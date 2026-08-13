package main

import (
	"os/exec"
	"runtime"
)

var (
	browserGOOS       = runtime.GOOS
	runBrowserCommand = func(name string, args ...string) error {
		return exec.Command(name, args...).Run()
	}
	openBrowserURL = platformOpenBrowserURL
)

func platformOpenBrowserURL(target string) error {
	switch browserGOOS {
	case "darwin":
		return runBrowserCommand("open", target)
	case "windows":
		return runBrowserCommand("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		return runBrowserCommand("xdg-open", target)
	}
}
