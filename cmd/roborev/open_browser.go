package main

import (
	"os/exec"
	"runtime"
)

var (
	browserGOOS         = runtime.GOOS
	startBrowserCommand = func(name string, args ...string) error {
		return exec.Command(name, args...).Start()
	}
	openBrowserURL = platformOpenBrowserURL
)

func platformOpenBrowserURL(target string) error {
	switch browserGOOS {
	case "darwin":
		return startBrowserCommand("open", target)
	case "windows":
		return startBrowserCommand("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		return startBrowserCommand("xdg-open", target)
	}
}
