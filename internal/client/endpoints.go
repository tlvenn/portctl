package client

import (
	"fmt"
	"net/http"
	"strconv"
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

// ResolveEndpointID picks the endpoint to operate on. Precedence:
//  1. override (from --endpoint) — numeric ID or endpoint name
//  2. configuredID (from PORTCTL_ENDPOINT_ID)
//  3. Auto-discovery of the first local Docker endpoint
func (c *Client) ResolveEndpointID(configuredID int, override string) (int, error) {
	if override != "" {
		if id, err := strconv.Atoi(override); err == nil {
			return id, nil
		}
		endpoints, err := c.ListEndpoints()
		if err != nil {
			return 0, err
		}
		for _, ep := range endpoints {
			if ep.Name == override {
				return ep.ID, nil
			}
		}
		names := make([]string, 0, len(endpoints))
		for _, ep := range endpoints {
			names = append(names, ep.Name)
		}
		return 0, fmt.Errorf("endpoint %q not found; available: %v", override, names)
	}

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
	return 0, fmt.Errorf("no local Docker endpoint found; set --endpoint or PORTCTL_ENDPOINT_ID")
}

// ResolveEndpointFilter returns the endpoint ID to scope stack lookups to.
// Returns 0 ("any endpoint") when no override or configured ID is set —
// preserves the cross-endpoint scan that commands like redeploy/delete/env
// have historically relied on.
func (c *Client) ResolveEndpointFilter(configuredID int, override string) (int, error) {
	if override == "" && configuredID == 0 {
		return 0, nil
	}
	return c.ResolveEndpointID(configuredID, override)
}
