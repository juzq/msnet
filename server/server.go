package server

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/zhyonc/msnet"
	"github.com/zhyonc/msnet/internal/opcode"
	"github.com/zhyonc/msnet/setting"
)

type server struct {
	addr    string
	lis     net.Listener
	handler Handler // packet handle
}

func NewServer(addr string) *server {
	s := &server{
		addr: addr,
	}
	return s
}

// new server with handler
func NewServerHandle(addr string, h Handler) *server {
	s := &server{
		addr:    addr,
		handler: h,
	}
	return s
}

func (s *server) Run() {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		slog.Error("Failed to create tcp listener", "err", err)
		return
	}
	slog.Info("TCPListener is starting on " + s.addr)
	s.lis = lis
	var idCount int32 = 0
	for {
		if s.lis == nil {
			slog.Warn("TCPListener is nil")
			break
		}
		conn, err := s.lis.Accept()
		if err != nil {
			slog.Error("Failed to accept conn", "err", err)
			continue
		}
		slog.Info("New client connected", "addr", conn.RemoteAddr())
		cs := msnet.NewCClientSocket(s, conn, nil, nil)
		go cs.OnRead()
		cs.OnConnect()
		cs.SetID(idCount)
		idCount++
	}
	slog.Info("Server stopped.")
}

func (s *server) Shutdown() {
	s.lis.Close()
	s.lis = nil
}

// DebugInPacketLog implements msnet.CClientSocketDelegate.
func (s *server) DebugInPacketLog(id int32, iPacket msnet.CInPacket) {
	key := iPacket.GetType()
	if setting.GSetting.SingleByteOpcode {
		key = uint16(iPacket.GetTypeByte())
	}
	_, ok := opcode.NotLogCP[key]
	if !ok {
		slog.Info("[CInPacket]", "id", id, "length", iPacket.GetLength(), "opcode", opcode.CPMap[key], "data", iPacket.DumpString(-1))
	}
}

// DebugOutPacketLog implements msnet.CClientSocketDelegate.
func (s *server) DebugOutPacketLog(id int32, oPacket msnet.COutPacket) {
	key := oPacket.GetType()
	if setting.GSetting.SingleByteOpcode {
		key = uint16(oPacket.GetTypeByte())
	}
	_, ok := opcode.NotLogLP[key]
	if !ok {
		slog.Info("[COutPacket]", "id", id, "length", oPacket.GetLength(), "opcode", opcode.LPMap[key], "data", oPacket.DumpString(-1))
	}
}

// ProcessPacket implements msnet.CClientSocketDelegate.
func (s *server) ProcessPacket(cs msnet.CClientSocket, iPacket msnet.CInPacket) {
	var op uint16
	if setting.GSetting.SingleByteOpcode {
		op = uint16(iPacket.Decode1())
	} else {
		op = iPacket.Decode2()
	}
	if s.handler != nil && s.handler.Handle(cs, iPacket) {
		return
	}
	slog.Info("Unprocessed CInPacket", "opcode", fmt.Sprintf("0x%X", op))
}

// SocketClose implements msnet.CClientSocketDelegate.
func (s *server) SocketClose(id int32) {
	slog.Info("Socket closed", "id", id)
}
