package neko

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// RewriteICEServers ensures the webrtc.iceservers.backend and frontend
// sections mirror the provided servers and returns the rendered YAML plus a flag
// indicating whether the serialized bytes changed from the input.
func RewriteICEServers(input []byte, servers []ICEServer) ([]byte, bool, error) {
	var root yaml.Node
	if len(input) > 0 {
		if err := yaml.Unmarshal(input, &root); err != nil {
			return nil, false, err
		}
	} else {
		root.Kind = yaml.DocumentNode
	}

	docMap, err := ensureDocumentMap(&root)
	if err != nil {
		return nil, false, err
	}

	webrtc, err := ensureMappingNode(docMap, "webrtc")
	if err != nil {
		return nil, false, err
	}

	iceservers, err := ensureMappingNode(webrtc, "iceservers")
	if err != nil {
		return nil, false, err
	}

	if err := setSequence(iceservers, "backend", buildICEServerSequence(servers)); err != nil {
		return nil, false, err
	}
	if err := setSequence(iceservers, "frontend", buildICEServerSequence(servers)); err != nil {
		return nil, false, err
	}

	output, err := yaml.Marshal(&root)
	if err != nil {
		return nil, false, err
	}

	return output, !bytes.Equal(input, output), nil
}

func ensureDocumentMap(doc *yaml.Node) (*yaml.Node, error) {
	if doc.Kind == 0 {
		doc.Kind = yaml.DocumentNode
	}

	if doc.Kind != yaml.DocumentNode {
		return nil, fmt.Errorf("document node is not a mapping")
	}

	if len(doc.Content) == 0 {
		mapNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		doc.Content = append(doc.Content, mapNode)
		return mapNode, nil
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("document root is not a mapping")
	}

	return root, nil
}

func ensureMappingNode(parent *yaml.Node, key string) (*yaml.Node, error) {
	if parent.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node for %q", key)
	}

	for i := 0; i < len(parent.Content); i += 2 {
		k := parent.Content[i]
		v := parent.Content[i+1]
		if k.Value == key {
			if v.Kind != yaml.MappingNode {
				return nil, fmt.Errorf("%s is not a mapping node", key)
			}
			return v, nil
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valueNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content, keyNode, valueNode)
	return valueNode, nil
}

func setSequence(parent *yaml.Node, key string, entries []*yaml.Node) error {
	if parent.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node for %q", key)
	}

	for i := 0; i < len(parent.Content); i += 2 {
		k := parent.Content[i]
		v := parent.Content[i+1]
		if k.Value == key {
			if v.Kind != yaml.SequenceNode {
				return fmt.Errorf("%s is not a sequence node", key)
			}
			v.Content = entries
			return nil
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: entries}
	parent.Content = append(parent.Content, keyNode, seq)
	return nil
}

func buildICEServerSequence(servers []ICEServer) []*yaml.Node {
	if len(servers) == 0 {
		return []*yaml.Node{}
	}

	result := make([]*yaml.Node, 0, len(servers))
	for _, server := range servers {
		entry := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		entry.Content = append(entry.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "urls"})
		entry.Content = append(entry.Content, buildStringSequence(server.URLs))

		if server.Username != "" {
			entry.Content = append(entry.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "username"})
			entry.Content = append(entry.Content, scalarNode(server.Username))
		}
		if server.Credential != "" {
			entry.Content = append(entry.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "credential"})
			entry.Content = append(entry.Content, scalarNode(server.Credential))
		}

		result = append(result, entry)
	}

	return result
}

func buildStringSequence(values []string) *yaml.Node {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, v := range values {
		seq.Content = append(seq.Content, scalarNode(v))
	}
	return seq
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}
