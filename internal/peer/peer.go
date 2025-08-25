package peer

import(
	"sync"
	"time"

	"github.com/jaywantadh/DisktroByte/internal/utils"
)

type PeerNode struct{
	Info		utils.NodeInfo
	LastSeen	time.Time
	Alive		bool
}

type PeerRegistry struct{
	mu		sync.RWMutex
	peers	map[string]*PeerNode
}

func NewPeerRegistry() *PeerRegistry {
	return &PeerRegistry{
		peers: make(map[string]*PeerNode),
	}
}

func (pr *PeerRegistry) AddPeer(info utils.NodeInfo){
	
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.peers[info.ID]; exists {
		return
	} 

	pr.peers[info.ID] = &PeerNode{
		Info: 		info,
		LastSeen: 	time.Time{},
		Alive: 		false,
	}
}

func (pr *PeerRegistry) Peers() map[string]*PeerNode {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	// Make a shallow copy to avoid race conditions
	copy := make(map[string]*PeerNode)
	for id, peer := range pr.peers {
		copy[id] = peer
	}
	return copy
}