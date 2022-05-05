package tools

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/codedninja/skener/pkg/tee"
)

type Tool struct {
	Tools map[string]*tee.Tee
}

func StartTools(interfaceHost string, mitmPort string, logPath string) (*Tool, error) {
	tools := map[string][]string{
		"mitmproxy": {
			"/opt/mitmproxy/mitmdump", // Location of mitmdump
			"--mode", "transparent",   // Transparent mode
			"--showhost",                   // Show hostname instead of IP
			"--ssl-insecure",               // Allow self-signed certificates
			"--listen-host", interfaceHost, // Listen on interface
			"--listen-port", mitmPort, // Listen on port
			"--save-stream-file", filepath.Join(logPath, "mitmproxy-stream.log"), // Save stream to file
			"--flow-detail", "3", // Show flow details 3
			"--set", "confdir=/root/.mitmproxy", // Set config directory
		},
	}

	t := &Tool{
		Tools: make(map[string]*tee.Tee, len(tools)),
	}

	var err error
	for tool, command := range tools {
		log.Println("Starting tool:", tool)
		log.Println(strings.Join(command, " "))
		t.Tools[tool], err = tee.Run(filepath.Join(logPath, tool+".log"), command[0], command[1:]...)
		if err != nil {
			log.Println("Error running tool:", err)
			return nil, err
		}
		log.Println(t.Tools)
	}

	return t, nil
}

func (t *Tool) Stop() {
	log.Println(t.Tools)
	for a, tool := range t.Tools {
		log.Println("Stopping tool:", a)
		if err := tool.Kill(); err != nil {
			log.Println("Error closing tool:", err)
		}
	}
}
