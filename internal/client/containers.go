package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Container struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	State   string            `json:"State"`
	Labels  map[string]string `json:"Labels"`
	Image   string            `json:"Image"`
	ImageID string            `json:"ImageID"`
}

type Image struct {
	ID          string   `json:"Id"`
	RepoTags    []string `json:"RepoTags"`
	RepoDigests []string `json:"RepoDigests"`
}

type DistributionInfo struct {
	Descriptor struct {
		Digest string `json:"digest"`
	} `json:"Descriptor"`
}

func (c *Client) ListImages(endpointID int) ([]Image, error) {
	reqURL := fmt.Sprintf("%s/api/endpoints/%d/docker/images/json", c.BaseURL, endpointID)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var images []Image
	if err := c.do(req, &images); err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	return images, nil
}

// GetRemoteDigest checks the remote registry for the current digest of an image reference.
// Uses Docker's distribution API via Portainer proxy.
func (c *Client) GetRemoteDigest(endpointID int, imageRef string) (string, error) {
	// URL-encode is not needed for the image ref in the path, Docker API handles it
	reqURL := fmt.Sprintf("%s/api/endpoints/%d/docker/distribution/%s/json", c.BaseURL, endpointID, imageRef)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	var info DistributionInfo
	if err := c.do(req, &info); err != nil {
		return "", err
	}
	return info.Descriptor.Digest, nil
}

// ImageStatus represents whether a container's image is up to date.
type ImageStatus int

const (
	ImageUnknown  ImageStatus = iota
	ImageUpToDate
	ImageOutdated
)

func (s ImageStatus) String() string {
	switch s {
	case ImageUpToDate:
		return "up to date"
	case ImageOutdated:
		return "outdated"
	default:
		return "unknown"
	}
}

// CheckImageStatus compares the local image digest against the remote registry.
func (c *Client) CheckImageStatus(endpointID int, container Container, images []Image) ImageStatus {
	// Find the local image by ID
	var localImage *Image
	for i, img := range images {
		if img.ID == container.ImageID {
			localImage = &images[i]
			break
		}
	}
	if localImage == nil {
		return ImageUnknown
	}

	// Get the image reference (e.g., "lscr.io/linuxserver/prowlarr:latest")
	imageRef := container.Image
	if imageRef == "" {
		return ImageUnknown
	}

	// Get the remote digest
	remoteDigest, err := c.GetRemoteDigest(endpointID, imageRef)
	if err != nil {
		return ImageUnknown
	}

	// Compare against local RepoDigests
	for _, localDigest := range localImage.RepoDigests {
		// RepoDigests format: "repo@sha256:abc..."
		if strings.Contains(localDigest, remoteDigest) {
			return ImageUpToDate
		}
	}

	return ImageOutdated
}

func (ct Container) ServiceName() string {
	if svc, ok := ct.Labels["com.docker.compose.service"]; ok {
		return svc
	}
	if len(ct.Names) > 0 {
		return strings.TrimPrefix(ct.Names[0], "/")
	}
	return ct.ID[:12]
}

func (c *Client) ListContainers(endpointID int, stackName string) ([]Container, error) {
	filters := fmt.Sprintf(`{"label":["com.docker.compose.project=%s"]}`, stackName)
	reqURL := fmt.Sprintf("%s/api/endpoints/%d/docker/containers/json?all=true&filters=%s",
		c.BaseURL, endpointID, url.QueryEscape(filters))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var containers []Container
	if err := c.do(req, &containers); err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	return containers, nil
}

func (c *Client) GetContainerLogs(endpointID int, containerID string, tail int, follow bool) (io.ReadCloser, error) {
	reqURL := fmt.Sprintf("%s/api/endpoints/%d/docker/containers/%s/logs?stdout=true&stderr=true&timestamps=true&tail=%d",
		c.BaseURL, endpointID, containerID, tail)

	if follow {
		reqURL += "&follow=true"
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// Use a separate client with no timeout for streaming
	httpClient := c.HTTPClient
	if follow {
		httpClient = &http.Client{}
	}

	req.Header.Set("X-API-Key", c.APIKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return &dockerLogReader{reader: resp.Body}, nil
}

// dockerLogReader strips Docker's 8-byte multiplexed stream headers.
type dockerLogReader struct {
	reader    io.ReadCloser
	remaining int
}

func (d *dockerLogReader) Read(p []byte) (int, error) {
	for {
		// If we have remaining payload bytes from a previous frame, read them
		if d.remaining > 0 {
			toRead := d.remaining
			if toRead > len(p) {
				toRead = len(p)
			}
			n, err := d.reader.Read(p[:toRead])
			d.remaining -= n
			return n, err
		}

		// Read the 8-byte Docker stream header
		var header [8]byte
		if _, err := io.ReadFull(d.reader, header[:]); err != nil {
			return 0, err
		}

		// Bytes 4-7 are the frame size (big-endian uint32)
		frameSize := int(binary.BigEndian.Uint32(header[4:8]))
		if frameSize == 0 {
			continue
		}
		d.remaining = frameSize
	}
}

func (d *dockerLogReader) Close() error {
	return d.reader.Close()
}
