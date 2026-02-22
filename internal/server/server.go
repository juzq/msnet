package server

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"

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
	key := iPacket.GetType()
	_, ok := opcode.NotLogCP[key]
	if !ok {
		slog.Info("[CInPacket]", "id", id, "length", iPacket.GetLength(), "opcode", opcode.CPMap[key], "data", iPacket.DumpString(-1))
	}
}

// DebugOutPacketLog implements msnet.CClientSocketDelegate.
func (s *server) DebugOutPacketLog(id int32, oPacket msnet.COutPacket) {
	key := oPacket.GetType()
	_, ok := opcode.NotLogLP[key]
	if !ok {
		slog.Info("[COutPacket]", "id", id, "length", oPacket.GetLength(), "opcode", opcode.LPMap[key], "data", oPacket.DumpString(-1))
	}
}

// ProcessPacket implements msnet.CClientSocketDelegate.
func (s *server) ProcessPacket(cs msnet.CClientSocket, iPacket msnet.CInPacket) {
	op := iPacket.Decode2()
	switch op {
	case 0x1: // 登录账号
		account := "aaaaa"
		accountId := int32(123456789)
		p := msnet.NewCOutPacket(0x00)
		p.Encode1(0)         // 登录状态
		p.Encode4(accountId) // 账号ID
		p.Encode1(0)         // 性别
		p.Encode1(0)         // Admin byte - Commands (v29 >> 5) & 1;
		p.Encode1(0)         // Admin byte - Commands (v29 >> 4) & 1;
		p.EncodeStr(account)
		p.EncodeBuffer([]byte{0x00, 0x00, 0x00, 0x03, 0x01, 0x00, 0x00, 0x00, 0xE2, 0xED, 0xA3, 0x7A, 0xFA, 0xC9, 0x01})
		p.Encode1(0)
		p.Encode8(0)
		p.Encode8(0)
		p.Encode2(0) //writeMapleAsciiString  CInPacket::DecodeStr
		p.Encode1(0) //是否显示登录弹出框
		p.EncodeStr(strconv.Itoa(int(accountId)))
		p.EncodeStr(account)
		p.Encode1(1) //0 = 提示没填身份证
		cs.SendPacket(p)
		pushWorldList(cs)
	case 0x20: // 登录界面背景
		p := msnet.NewCOutPacket(0x1F)
		p.EncodeStr("MapLogin2")
		cs.SendPacket(p)
	default:
		slog.Info("Unprocessed CInPacket", "opcode", fmt.Sprintf("0x%X", op))
	}
}

func pushWorldList(cs msnet.CClientSocket) {
	p := msnet.NewCOutPacket(0x09)
	serverId := 0
	p.Encode1(int8(serverId)) //serverId // 0 = Aquilla, 1 = bootes, 2 = cass, 3 = delphinus
	serverName := "World"
	p.EncodeStr(serverName)
	p.Encode1(0) //p.write(LoginServer.getFlag());
	p.EncodeStr("Tip")
	p.Encode2(100)
	p.Encode2(100)

	p.Encode1(1)     // channel num
	p.Encode4(10000) // load

	for i := 0; i < 1; i++ {
		p.EncodeStr(fmt.Sprintf("%s-%d", serverName, i+1))
		p.Encode4(10000) // load
		p.Encode1(int8(serverId))
		p.Encode2(int16(i))
	}
	p.Encode2(0)
	cs.SendPacket(p)
	// 服务器列表结束
	p2 := msnet.NewCOutPacket(0x09)
	p2.Encode1(-1)
	cs.SendPacket(p2)
}

// SocketClose implements msnet.CClientSocketDelegate.
func (s *server) SocketClose(id int32) {
	slog.Info("Socket closed", "id", id)
}
