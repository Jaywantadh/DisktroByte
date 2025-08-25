package discovery

import (
	"sync"

	"github.com/jaywantadh/DisktroByte/internal/utils"
)

type Registry struct{
	nodes	map[string]utils.NodeInfo
	mu		sync.RWMutex
}

func NewRegistry() *Registry{
	return &Registry{
		nodes: make(map[string]utils.NodeInfo),
	}
}

func (r *Registry) RegisterNode(n utils.NodeInfo){
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes[n.ID] = n
}

func (r *Registry) GetAllNodes() []utils.NodeInfo{
	r.mu.RLock()
	defer r.mu.Unlock()
	nodes := make([]utils.NodeInfo, 0, len(r.nodes))
	for _, n := range r.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

