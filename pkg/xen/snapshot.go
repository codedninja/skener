package xen

import (
	"log"

	xenapi "github.com/terra-farm/go-xen-api-client"
)

type Snapshot struct {
	client   *Client
	vm       *VM
	snapshot xenapi.VMRef
}

func (x *VM) GetSnapshots() ([]*Snapshot, error) {
	ss, err := x.client.xapi.VM.GetSnapshots(x.client.session, x.vm)
	if err != nil {
		return nil, err
	}

	var snapshots []*Snapshot
	for _, snapshot := range ss {
		snapshots = append(snapshots, &Snapshot{
			client:   x.client,
			vm:       x,
			snapshot: snapshot,
		})

		log.Printf("snapshot: %s", snapshot)
	}

	log.Println(snapshots[0])

	return snapshots, nil
}

func (s *Snapshot) Revert() error {
	return s.client.xapi.VM.Revert(s.client.session, s.snapshot)
}
