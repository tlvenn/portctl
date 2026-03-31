package client

import (
	"fmt"
	"net/http"
)

type Endpoint struct {
	ID   int    `json:"Id"`
	Name string `json:"Name"`
	Type int    `json:"Type"`
	URL  string `json:"URL"`
}

func (c *Client) ListEndpoints() ([]Endpoint, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/api/endpoints", nil)
	if err != nil {
		return nil, err
	}

	var endpoints []Endpoint
	if err := c.do(req, &endpoints); err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}
	return endpoints, nil
}

// ResolveEndpointID returns the configured endpoint ID, or auto-discovers the first local Docker endpoint.
func (c *Client) ResolveEndpointID(configuredID int) (int, error) {
	if configuredID != 0 {
		return configuredID, nil
	}

	endpoints, err := c.ListEndpoints()
	if err != nil {
		return 0, err
	}

	for _, ep := range endpoints {
		if ep.Type == 1 { // DockerEnvironment (local socket)
			return ep.ID, nil
		}
	}
	return 0, fmt.Errorf("no local Docker endpoint found; set PORTCTL_ENDPOINT_ID manually")
}
