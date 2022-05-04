package agent

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Agent struct {
	Address string
}

func NewAgent(address string) *Agent {
	return &Agent{
		Address: address,
	}
}

func (a *Agent) UploadFile(path string) error {
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

	request, err := http.NewRequest("POST", fmt.Sprintf("http://%s/malware", a.Address), body)
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

func (a *Agent) Execute() error {
	resp, err := http.Post(fmt.Sprintf("http://%s/execute", a.Address), "text/plain", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("%s", resp.Status)
	}

	return nil
}

func (a *Agent) Finish() (string, error) {
	resp, err := http.Post(fmt.Sprintf("http://%s/finish", a.Address), "text/plain", nil)
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
