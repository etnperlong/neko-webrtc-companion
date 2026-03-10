package cloudflare

import (
	"encoding/json"
	"fmt"
)

// ICEServer captures the normalized data necessary to wire up a WebRTC peer
// connection.
type ICEServer struct {
	URLs       []string
	Username   string
	Credential string
}

// ParseICEServers converts a Cloudflare TURN allocation response into a slice
// of normalized ICE servers.
func ParseICEServers(body []byte) ([]ICEServer, error) {
	var resp cloudflareResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	servers := make([]ICEServer, 0, len(resp.IceServers))
	for _, entry := range resp.IceServers {
		if len(entry.URLs) == 0 {
			continue
		}
		servers = append(servers, ICEServer{
			URLs:       []string(entry.URLs),
			Username:   entry.Username,
			Credential: entry.Credential,
		})
	}

	return servers, nil
}

type cloudflareResponse struct {
	IceServers []cloudflareICEServer `json:"iceServers"`
}

type cloudflareICEServer struct {
	URLs       urlsField `json:"urls"`
	Username   string    `json:"username"`
	Credential string    `json:"credential"`
}

type urlsField []string

func (u *urlsField) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*u = urlsField{single}
		return nil
	}

	var multi []string
	if err := json.Unmarshal(data, &multi); err == nil {
		*u = urlsField(multi)
		return nil
	}

	return fmt.Errorf("cloudflare: unexpected urls value %s", string(data))
}
