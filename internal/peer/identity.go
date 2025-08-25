package peer

import (
	"time"

	"github.com/google/uuid"
	"github.com/jaywantadh/DisktroByte/internal/utils"
)

func GenerateNodeInfo(hostAddr string) utils.NodeInfo{
	return utils.NodeInfo{
		ID:			uuid.New().String(),
		Address: 	hostAddr,
		JoinTime: 	time.Now().Unix(),
	}
} 