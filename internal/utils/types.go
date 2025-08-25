package utils

type NodeInfo struct{
	ID			string `json: "id"`
	Address		string `json: "address"`
	JoinTime	int64  `json: "join_time"`
}