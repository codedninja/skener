package route

import (
	"github.com/coreos/go-iptables/iptables"
	log "github.com/sirupsen/logrus"
)

type Route struct {
	Rules           []string
	DestinationPort string
	SourceHost      string
	ToPort          string
	tables          *iptables.IPTables
}

func NewRoute(protocol string, toPort string, sourceHost string, destionationPort string) (*Route, error) {
	tables, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return &Route{
		Rules: []string{
			"-s", sourceHost,
			"-p", protocol,
			"--destination-port", destionationPort,
			"-j", "REDIRECT",
			"--to-port", toPort,
		},
		ToPort:          toPort,
		SourceHost:      sourceHost,
		DestinationPort: destionationPort,
		tables:          tables,
	}, nil
}

func (r *Route) Apply() error {
	return r.tables.Append("nat", "PREROUTING", r.Rules...)
}

func (r *Route) Delete() error {
	if err := r.tables.Delete("nat", "PREROUTING", r.Rules...); err != nil {
		log.Error(err)
		return err
	}

	return nil
}
