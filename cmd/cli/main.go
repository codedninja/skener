package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

	log.Printf("Uploading file %s to agent at %s:%s", *file, *agentIP, *agentPort)

	if err := uploadFile(*file, *agentIP, *agentPort); err != nil {
		log.Fatal(err)
	}

	if err := executeMalware(*agentIP, *agentPort, *timeout); err != nil {
		log.Fatal(err)
	}

	log.Printf("Waiting for agent to finish")
	time.Sleep(time.Second * time.Duration(*timeout))

	logs, err := finish(*agentIP, *agentPort)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(logs)
}

func finish(agentIP string, agentPort string) (string, error) {
	resp, err := http.Post(fmt.Sprintf("http://%s:%s/finish", agentIP, agentPort), "text/plain", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func executeMalware(agentIP string, agentPort string, timeout int) error {
	resp, err := http.Post(fmt.Sprintf("http://%s:%s/execute", agentIP, agentPort), "text/plain", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("%s", resp.Status)
	}

	return nil
}

func uploadFile(path string, agentIP string, agentPort string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// File
	formFile, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return err
	}

	if _, err := io.Copy(formFile, f); err != nil {
		return err
	}

	// Filename
	formFilename, err := writer.CreateFormField("filename")
	if err != nil {
		return err
	}

	_, err = io.Copy(formFilename, bytes.NewBufferString(filepath.Base(path)))
	if err != nil {
		return err
	}

	writer.Close()

	request, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%s/malware", agentIP, agentPort), body)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 2 * time.Minute}

	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("%s", resp.Status)
	}

	return nil
}
