package worker

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Address string `default:"0.0.0.0:5050"`

	LogPath string `default:"./log/"`

	Xen struct {
		Address      string
		Username     string
		Password     string
		Insecure     bool `default:"true"`
		Tag          string
		AutoRollback bool `default:"true"`
	}

	Mitm struct {
		Path  string
		Ports struct {
			Min int
			Max int
		}
	}

	DNS struct {
		Path  string
		Ports struct {
			Min int
			Max int
		}
	}
}

func (w *Worker) loadConfig() error {
	f, err := os.Open("config.yaml")
	if err != nil {
		return err
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(w.config)
	if err != nil {
		return err
	}

	log.Printf("%+v", w.config)

	return nil
}
