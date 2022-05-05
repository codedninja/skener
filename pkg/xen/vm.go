package xen

import xenapi "github.com/terra-farm/go-xen-api-client"

type VM struct {
	client *Client
	vm     xenapi.VMRef
}

func (c *Client) GetAllVMs() ([]*VM, error) {
	vms := []*VM{}

	vmsRefs, err := c.xapi.VM.GetAll(c.session)
	if err != nil {
		return nil, err
	}

	for _, vmRef := range vmsRefs {
		template, err := c.xapi.VM.GetIsATemplate(c.session, vmRef)
		if err != nil || template {
			continue
		}

		vms = append(vms, &VM{
			client: c,
			vm:     vmRef,
		})
	}

	return vms, nil
}

func (c *Client) GetVMByUUID(uuid string) (*VM, error) {
	vm, err := c.xapi.VM.GetByUUID(c.session, uuid)
	if err != nil {
		return nil, err
	}

	return &VM{
		client: c,
		vm:     vm,
	}, nil
}

func (c *Client) GetVMsByTag(tag string) ([]*VM, error) {
	vms, err := c.GetAllVMs()
	if err != nil {
		return nil, err
	}

	outputVM := []*VM{}

	for _, vm := range vms {
		tags, err := vm.GetTags()
		if err != nil {
			return nil, err
		}

		if contains(tags, tag) {
			outputVM = append(outputVM, vm)
		}
	}

	return outputVM, nil
}

func (v *VM) GetTags() ([]string, error) {
	tags, err := v.client.xapi.VM.GetTags(v.client.session, v.vm)
	if err != nil {
		return nil, err
	}

	return tags, nil
}

func (v *VM) GetName() (string, error) {
	name, err := v.client.xapi.VM.GetNameLabel(v.client.session, v.vm)
	if err != nil {
		return "", err
	}

	return name, nil
}

func (v *VM) GetPowerState() (string, error) {
	status, err := v.client.xapi.VM.GetPowerState(v.client.session, v.vm)
	if err != nil {
		return "", err
	}

	return string(status), nil
}

func (v *VM) GetIPs() ([]string, error) {
	ips := []string{}

	// Get IPs from Guest Metrics
	guestMetrics, err := v.client.xapi.VM.GetGuestMetrics(v.client.session, v.vm)
	if err != nil {
		return nil, err
	}

	guestNetworks, err := v.client.xapi.VMGuestMetrics.GetNetworks(v.client.session, guestMetrics)
	if err != nil {
		return nil, err
	}

	for _, guestNetwork := range guestNetworks {
		if !contains(ips, guestNetwork) {
			ips = append(ips, guestNetwork)
		}
	}

	// TODO: Get ips from VIFs

	return ips, nil
}
