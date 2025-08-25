package httpserver

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jaywantadh/DisktroByte/internal/discovery"
	"github.com/jaywantadh/DisktroByte/internal/utils"
)

var Registry *discovery.Registry

func Init(reg *discovery.Registry) {
	Registry = reg
	http.HandleFunc("/join", handleJoin)
	go http.ListenAndServe(":8080", nil)
}

func handleJoin(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodPost{
		http.Error(w, "Method not allowed!", http.StatusMethodNotAllowed)
		return
	}

	var node utils.NodeInfo
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil{
		http.Error(w, "Invalid body!", http.StatusBadRequest)
		return
	}

	Registry.RegisterNode(node)
	log.Printf("New Node Joined: %s @ %s\n", node.ID, node.Address)
	w.WriteHeader(http.StatusOK)

}
