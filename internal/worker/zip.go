package worker

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

var logFiles = []string{
	"agent.log",
	"mitmproxy-stream.log", // Stream file from mitmproxy
	"mitmproxy.log",        // Log from stdout & stderr from mitmproxy
	"dnschef.log",          // Log from stdout & stderr from dnschef
}

func (w *Worker) zipResults(path string) error {
	archive, err := os.Create(filepath.Join(path, "results.zip"))
	if err != nil {
		return err
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)

	for _, file := range logFiles {
		f, err := os.Open(filepath.Join(path, file))
		if err != nil {
			return err
		}
		defer f.Close()

		w, err := zipWriter.Create(file)
		if err != nil {
			return err
		}

		if _, err = io.Copy(w, f); err != nil {
			return err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return err
	}

	return nil
}
