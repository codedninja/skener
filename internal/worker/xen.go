package worker

import (
	"github.com/codedninja/skener/pkg/xen"
)

func (w *Worker) connectXenServer() error {
	w.xen = xen.NewClient(xen.Config{
		Address:  w.config.Xen.Address,
		Username: w.config.Xen.Username,
		Password: w.config.Xen.Password,
	})

	if err := w.xen.Connect(); err != nil {
		return err
	}

	return nil
}
