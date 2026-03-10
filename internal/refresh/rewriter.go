package refresh

import (
	"context"

	"github.com/etnperlong/neko-webrtc-companion/internal/neko"
)

// NewNekoRewriter returns a Rewriter that delegates to neko.RewriteICEServers.
func NewNekoRewriter() Rewriter {
	return &nekoRewriter{}
}

type nekoRewriter struct{}

func (n *nekoRewriter) Rewrite(ctx context.Context, current []byte, servers []neko.ICEServer) ([]byte, bool, error) {
	return neko.RewriteICEServers(current, servers)
}
