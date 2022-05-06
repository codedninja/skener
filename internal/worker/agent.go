package worker

import (
	"fmt"
	"net"

	"github.com/codedninja/skener/internal/queue"
	log "github.com/sirupsen/logrus"
)

func (w *Worker) findPossibleAgents() error {
	vms, err := w.xen.GetVMsByTag(w.config.Xen.Tag)
	if err != nil {
		return nil
	}

	mitmPort := w.config.Mitm.Ports.Min
	dnsPort := w.config.DNS.Ports.Min

	for i := 0; i < len(vms); i++ {
		agent := &queue.Agent{
			VM:       vms[i],
			MITMPort: fmt.Sprintf("%d", mitmPort),
			DNSPort:  fmt.Sprintf("%d", dnsPort),
		}

		ips, err := agent.VM.GetIPs()
		if err != nil {
			log.Error(err)
			continue
		}

		for _, ip := range ips {
			if ip4 := net.ParseIP(ip).To4(); ip4 != nil {
				agent.IP = ip4.String()
				break
			}
		}

		w.agents = append(w.agents, agent)

		// Check if any more available ports
		mitmPort++
		if mitmPort > w.config.Mitm.Ports.Max {
			break
		}

		dnsPort++
		if dnsPort > w.config.DNS.Ports.Max {
			break
		}
	}

	return nil
}
