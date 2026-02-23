package server

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/zhyonc/msnet"
	"github.com/zhyonc/msnet/internal/opcode"
)

type server struct {
	addr string
	lis  net.Listener
}

func NewServer(addr string) *server {
	s := &server{
		addr: addr,
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
}

func (s *server) Shutdown() {
	s.lis.Close()
	s.lis = nil
}

// DebugInPacketLog implements msnet.CClientSocketDelegate.
func (s *server) DebugInPacketLog(id int32, iPacket msnet.CInPacket) {
	key := iPacket.GetTypeByte()
	_, ok := opcode.NotLogCP[key]
	if !ok {
		slog.Info("[CInPacket]", "id", id, "length", iPacket.GetLength(), "opcode", opcode.CPMap[key], "data", iPacket.DumpString(-1))
	}
}

// DebugOutPacketLog implements msnet.CClientSocketDelegate.
func (s *server) DebugOutPacketLog(id int32, oPacket msnet.COutPacket) {
	key := oPacket.GetTypeByte()
	_, ok := opcode.NotLogLP[key]
	if !ok {
		slog.Info("[COutPacket]", "id", id, "length", oPacket.GetLength(), "opcode", opcode.LPMap[key], "data", oPacket.DumpString(-1))
	}
}

// ProcessPacket implements msnet.CClientSocketDelegate.
func (s *server) ProcessPacket(cs msnet.CClientSocket, iPacket msnet.CInPacket) {
	op := iPacket.Decode1()
	switch op {
	case 0x01:
		// 登录返回
		accountId := 123456
		account := "aaaa"
		p := msnet.NewCOutPacketByte(0x01)
		p.Encode1(0)
		p.Encode4(int32(accountId))
		p.Encode1(0) // 性别
		p.Encode1(0) // gm
		p.EncodeStr(account)
		p.Encode4(int32(accountId))
		p.Encode1(0)
		cs.SendPacket(p)
		// 服务器列表
		p1 := msnet.NewCOutPacketByte(0x05)
		p1.Encode1(0)    // 服务器id
		p1.EncodeStr("") // 服务器名字
		p1.Encode1(1)    // 频道数量

		p1.EncodeLocalStr("蓝蜗牛-1")
		p1.Encode4(10000)
		p1.Encode1(0) // 服务器id
		p1.Encode1(0) // 频道下标，从0开始
		p1.Encode1(0)
		cs.SendPacket(p1)
		p2 := msnet.NewCOutPacketByte(0x05)
		p2.Encode1(-1)
		cs.SendPacket(p2)
	case 0x03:
		// 服务器状态
		p := msnet.NewCOutPacketByte(0x04)
		p.Encode1(0) // 状态正常
		cs.SendPacket(p)
	case 0x04:
		// 角色列表
		p := msnet.NewCOutPacketByte(0x06)
		p.Encode1(0)
		p.Encode4(0)
		p.Encode1(0) // 角色数量
		// TODO 角色信息
		cs.SendPacket(p)
	default:
		slog.Info("Unprocessed CInPacket", "opcode", fmt.Sprintf("0x%X", op))
	}
}

// SocketClose implements msnet.CClientSocketDelegate.
func (s *server) SocketClose(id int32) {
	slog.Info("Socket closed", "id", id)
}
