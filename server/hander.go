package server

import "github.com/zhyonc/msnet"

// Packet Handle
type Handler interface {
	Handle(cs msnet.CClientSocket, packet msnet.CInPacket) bool
}
