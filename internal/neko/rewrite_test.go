package neko

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRewriteICEServers_ReplacesFrontendAndBackend(t *testing.T) {
	input := []byte("webrtc:\n  iceservers:\n    backend: []\n    frontend: []\n")
	servers := []ICEServer{
		{
			URLs:       []string{"turn:turn.cloudflare.com:3478?transport=udp"},
			Username:   "u",
			Credential: "p",
		},
	}

	output, changed, err := RewriteICEServers(input, servers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !changed {
		t.Fatalf("expected change, got unchanged")
	}

	doc := parseDocument(t, output)
	webrtc := findMapping(t, doc, "webrtc")
	iceservers := findMapping(t, webrtc, "iceservers")

	backend := findSequence(t, iceservers, "backend")
	frontend := findSequence(t, iceservers, "frontend")

	if len(backend.Content) != 1 {
		t.Fatalf("expected backend sequence length 1, got %d", len(backend.Content))
	}
	if len(frontend.Content) != 1 {
		t.Fatalf("expected frontend sequence length 1, got %d", len(frontend.Content))
	}

	backendEntry := backend.Content[0]
	frontendEntry := frontend.Content[0]

	if backendEntry.Kind != yaml.MappingNode || frontendEntry.Kind != yaml.MappingNode {
		t.Fatalf("expected mapping entries inside sequences")
	}

	if sequenceFirstScalarValue(t, backendEntry, "urls") != "turn:turn.cloudflare.com:3478?transport=udp" {
		t.Fatalf("backend urls mismatch")
	}
	if scalarValue(t, backendEntry, "username") != "u" {
		t.Fatalf("backend username mismatch")
	}
	if scalarValue(t, backendEntry, "credential") != "p" {
		t.Fatalf("backend credential mismatch")
	}

	if sequenceFirstScalarValue(t, frontendEntry, "urls") != "turn:turn.cloudflare.com:3478?transport=udp" {
		t.Fatalf("frontend urls mismatch")
	}
	if scalarValue(t, frontendEntry, "username") != "u" {
		t.Fatalf("frontend username mismatch")
	}
	if scalarValue(t, frontendEntry, "credential") != "p" {
		t.Fatalf("frontend credential mismatch")
	}
}

func TestRewriteICEServers_PreservesUnrelatedKeys(t *testing.T) {
	input := []byte("foo: bar\nwebrtc:\n  iceservers:\n    backend: []\n    frontend: []\n")
	servers := []ICEServer{{URLs: []string{"turn:turn.cloudflare.com:3478?transport=udp"}}}

	output, _, err := RewriteICEServers(input, servers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	doc := parseDocument(t, output)
	if scalar := scalarValue(t, doc, "foo"); scalar != "bar" {
		t.Fatalf("expected foo preserved, got %q", scalar)
	}
}

func TestRewriteICEServers_Idempotent(t *testing.T) {
	input := []byte("webrtc:\n  iceservers:\n    backend: []\n    frontend: []\n")
	servers := []ICEServer{{URLs: []string{"turn:turn.cloudflare.com:3478?transport=udp"}}}

	output, changed, err := RewriteICEServers(input, servers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatalf("expected change on first rewrite")
	}

	output2, changed2, err := RewriteICEServers(output, servers)
	if err != nil {
		t.Fatalf("unexpected error on second rewrite: %v", err)
	}
	if changed2 {
		t.Fatalf("expected idempotent rewrite to report no change")
	}
	if string(output) != string(output2) {
		t.Fatalf("expected outputs to match")
	}
}

func TestRewriteICEServers_InvalidStructure(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "webrtc_not_map", input: "webrtc: []\n"},
		{name: "iceservers_not_map", input: "webrtc:\n  iceservers: []\n"},
		{name: "backend_not_seq", input: "webrtc:\n  iceservers:\n    backend: {}\n    frontend: []\n"},
		{name: "frontend_not_seq", input: "webrtc:\n  iceservers:\n    backend: []\n    frontend: {}\n"},
	}

	servers := []ICEServer{{URLs: []string{"turn:turn.cloudflare.com:3478?transport=udp"}}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := RewriteICEServers([]byte(tc.input), servers); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func parseDocument(t *testing.T, data []byte) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(doc.Content) == 0 {
		t.Fatalf("document has no content")
	}
	return doc.Content[0]
}

func findMapping(t *testing.T, parent *yaml.Node, key string) *yaml.Node {
	t.Helper()
	if parent.Kind != yaml.MappingNode {
		t.Fatalf("expected mapping node for %s", key)
	}
	for i := 0; i < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			node := parent.Content[i+1]
			if node.Kind != yaml.MappingNode {
				t.Fatalf("%s not mapping", key)
			}
			return node
		}
	}
	t.Fatalf("missing key %s", key)
	return nil
}

func findSequence(t *testing.T, parent *yaml.Node, key string) *yaml.Node {
	t.Helper()
	if parent.Kind != yaml.MappingNode {
		t.Fatalf("expected mapping node for %s", key)
	}
	for i := 0; i < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			node := parent.Content[i+1]
			if node.Kind != yaml.SequenceNode {
				t.Fatalf("%s not sequence", key)
			}
			return node
		}
	}
	t.Fatalf("missing sequence %s", key)
	return nil
}

func scalarValue(t *testing.T, parent *yaml.Node, key string) string {
	t.Helper()
	if parent.Kind != yaml.MappingNode {
		t.Fatalf("expected mapping node for scalar lookup %s", key)
	}
	for i := 0; i < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			if parent.Content[i+1].Kind != yaml.ScalarNode {
				t.Fatalf("%s is not scalar", key)
			}
			return parent.Content[i+1].Value
		}
	}
	t.Fatalf("missing scalar %s", key)
	return ""
}

func sequenceFirstScalarValue(t *testing.T, parent *yaml.Node, key string) string {
	t.Helper()
	seq := findSequence(t, parent, key)
	if len(seq.Content) == 0 {
		t.Fatalf("sequence %s is empty", key)
	}
	if seq.Content[0].Kind != yaml.ScalarNode {
		t.Fatalf("sequence %s first element is not scalar", key)
	}
	return seq.Content[0].Value
}
