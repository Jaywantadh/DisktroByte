package peer

import(
	"fmt"
	"net/http"
	"time"
)

func (pr *PeerRegistry) Ping(peerID, peerAddr string){
	client := &http.Client{Timeout: 3 * time.Second}
	url := fmt.Sprintf("http://%s/ping", peerAddr)
	
	resp, err := client.Get(url)
	now := time.Now()

	pr.mu.Lock()
	defer pr.mu.Unlock()

	peer, exists := pr.peers[peerID]
	if !exists{
		return
	}

	if err != nil || resp.StatusCode != 200 {
		peer.Alive = false
		return
	}

	peer.Alive = true
	peer.LastSeen = now
}