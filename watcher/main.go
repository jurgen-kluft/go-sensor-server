package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

const (
	endpointToMonitor = "http://192.168.8.88:8080/health"
	checkInterval     = 5 * time.Minute
)

func triggerLocalMacNotification(message string) {
	// Uses AppleScript to fire a native macOS notification banner
	script := fmt.Sprintf(`display notification "%s" with title "⚠️ Endpoint Alert" sound name "Frog"`, message)
	cmd := exec.Command("osascript", "-e", script)

	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to execute AppleScript notification: %v\n", err)
	}
}

func checkEndpoint() {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(endpointToMonitor)
	if err != nil {
		triggerLocalMacNotification("Endpoint is DOWN or unreachable!")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		triggerLocalMacNotification(fmt.Sprintf("Endpoint error status: %d", resp.StatusCode))
	} else {
		triggerLocalMacNotification("Endpoint is UP and running smoothly!")
	}
}

func main() {
	checkEndpoint()
	ticker := time.NewTicker(checkInterval)
	for range ticker.C {
		checkEndpoint()
	}
}
