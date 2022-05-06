package main

import (
	"github.com/codedninja/skener/internal/worker"
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := worker.Run(); err != nil {
		log.Fatal(err)
	}
}
