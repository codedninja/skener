package route

import (
	"github.com/coreos/go-iptables/iptables"
	log "github.com/sirupsen/logrus"
)

type Routes struct {
	Routes [][]string
	tables *iptables.IPTables
}

func ApplyRules(sourceHost string, mitmPort string, dnsPort string) (*Routes, error) {
	tables, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	tools := []struct {
		Protocol        string
		SourceHost      string
		ToPort          string
		DestinationPort string
	}{
		{
			Protocol:        "tcp",
			SourceHost:      sourceHost,
			ToPort:          mitmPort,
			DestinationPort: "80",
		},
		{
			Protocol:        "tcp",
			SourceHost:      sourceHost,
			ToPort:          mitmPort,
			DestinationPort: "443",
		},
		{
			Protocol:        "udp",
			SourceHost:      sourceHost,
			ToPort:          dnsPort,
			DestinationPort: "53",
		},
	}

	routes := &Routes{
		tables: tables,
	}

	for _, tool := range tools {
		rules := []string{
			"-s", tool.SourceHost,
			"-p", tool.Protocol,
			"--destination-port", tool.DestinationPort,
			"-j", "REDIRECT",
			"--to-port", tool.ToPort,
		}
		if err := routes.tables.Append("nat", "PREROUTING", rules...); err != nil {
			log.Error(err)
			return nil, err
		}

		routes.Routes = append(routes.Routes, rules)
	}

	return routes, nil
}

func (r *Routes) Delete() {
	for _, rule := range r.Routes {
		if err := r.tables.Delete("nat", "PREROUTING", rule...); err != nil {
			log.Error(err)
		}
	}
}
