package peer

import (
	"time"

	"github.com/sirupsen/logrus"
)

func (pr *PeerRegistry) StartMonitor(interval time.Duration){
	go func (){
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
	

		for{
			<-ticker.C
			pr.pingAllpeers()
		}
	}()
}

func (pr *PeerRegistry) pingAllpeers(){
	pr.mu.Lock()
	peers := make([]*PeerNode, 0, len(pr.peers))
	for _, peer := range pr.peers {
		peers = append(peers, peer)
	}
	pr.mu.Unlock()
	
	
	for _, peer := range peers {
		go pr.Ping(peer.Info.ID, peer.Info.Address)
	}
}

func (pr *PeerRegistry) CheckDeadPeers(threshold time.Duration){
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	now := time.Now()
	for _, peer := range pr.peers{
		if peer.Alive && now.Sub(peer.LastSeen) > threshold {
			logrus.Warnf("Peer %s (%s) is marked inactive - last seen %v ago",
				peer.Info.ID, peer.Info.Address, now.Sub(peer.LastSeen))
		// space for pruning logic 
		}
	}
}