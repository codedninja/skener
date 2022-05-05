package xen

import (
	"crypto/tls"
	"net/http"

	xenapi "github.com/terra-farm/go-xen-api-client"
)

type Config struct {
	Address  string
	Username string
	Password string
}

type Client struct {
	config  *Config
	xapi    *xenapi.Client
	session xenapi.SessionRef
}

func NewClient(config Config) *Client {
	return &Client{
		config: &config,
	}
}

func (c *Client) Connect() error {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	xapi, err := xenapi.NewClient("https://"+c.config.Address, transport)
	if err != nil {
		return err
	}

	session, err := xapi.Session.LoginWithPassword(c.config.Username, c.config.Password, "0.1", "skener")
	if err != nil {
		return err
	}

	c.xapi = xapi
	c.session = session

	return nil
}
