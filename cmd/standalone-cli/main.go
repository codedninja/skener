package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/codedninja/skener/pkg/agent"
	log "github.com/sirupsen/logrus"
)

func main() {
	agentIP := flag.String("agent-ip", "", "IP address of the agent")
	agentPort := flag.String("agent-port", "6060", "Port of the agent")
	timeout := flag.Int("timeout", 120, "Timeout for the agent")

	file := flag.String("file", "", "File to upload")

	flag.Parse()

	if *agentIP == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *file == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	abs, err := filepath.Abs(*file)
	if err != nil {
		log.Fatal(err)
	}
	file = &abs

	client := agent.NewAgent(fmt.Sprintf("%s:%s", *agentIP, *agentPort))

	log.Printf("Uploading file %s to agent at %s:%s", *file, *agentIP, *agentPort)

	if err := client.UploadFile(*file); err != nil {
		log.Fatal(err)
	}

	if err := client.Execute(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Waiting for agent to finish")
	time.Sleep(time.Second * time.Duration(*timeout))

	logs, err := client.Finish()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(logs)
}
