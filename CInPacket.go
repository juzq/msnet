package msnet

import (
	"encoding/binary"
	"fmt"
	"log/slog"

	"github.com/zhyonc/msnet/crypt"
	"github.com/zhyonc/msnet/enum"
	"github.com/zhyonc/msnet/setting"

	"strings"
	"time"
)

type iPacket struct {
	RawSeq   uint16
	DataLen  int
	RecvBuff []byte
	Length   int
	Offset   int
}

func NewCInPacket(buf []byte) CInPacket {
	p := &iPacket{}
	p.RecvBuff = buf
	p.Length = len(buf)
	return p
}

// AppendBuffer implements CInPacket.
func (p *iPacket) AppendBuffer(pBuff []byte, bEnc bool) {
	// Decode packet length
	p.RecvBuff = pBuff
	p.Length = len(pBuff)
	p.Offset = 0
	p.RawSeq = uint16(p.Decode2())
	temp := uint16(p.Decode2())
	if !setting.GSetting.IsXORCipher && bEnc {
		temp ^= p.RawSeq
	}
	p.DataLen = int(temp)
}

// DecryptData implements CInPacket.
func (p *iPacket) DecryptData(dwKey []byte) {
	if p.Length <= 0 && p.Length > maxDataLength {
		slog.Warn("Invalid data length")
		return
	}

	if setting.GSetting.IsXORCipher {
		(*crypt.XORCipher).Decrypt(nil, p.RecvBuff, dwKey)
		return
	}

	if setting.GSetting.IsCycleAESKey {
		(*crypt.CAESCipher).Decrypt(nil, crypt.CycleAESKeys[setting.GSetting.MSVersion%20], p.RecvBuff, dwKey)
	} else {
		(*crypt.CAESCipher).Decrypt(nil, setting.GSetting.AESKeyDecrypt, p.RecvBuff, dwKey)
	}
	if setting.GSetting.MSRegion > enum.TMS || (setting.GSetting.MSRegion == enum.CMS && setting.GSetting.MSVersion < 86) {
		(*crypt.CIOBufferManipulator).De(nil, p.RecvBuff)
	}
}

// GetType implements CInPacket.
func (p *iPacket) GetType() uint16 {
	if len(p.RecvBuff) >= 2 {
		return uint16(p.RecvBuff[0]) | uint16(p.RecvBuff[1])<<8
	}
	return 0
}

// GetTypeByte implements CInPacket.
func (p *iPacket) GetTypeByte() uint8 {
	if len(p.RecvBuff) >= 1 {
		return uint8(p.RecvBuff[0])
	}
	return 0
}

// GetRemain implements CInPacket.
func (p *iPacket) GetRemain() int {
	return p.Length - p.Offset
}

// GetOffset implements CInPacket.
func (p *iPacket) GetOffset() int {
	return p.Offset
}

// GetLength implements CInPacket.
func (p *iPacket) GetLength() int {
	return p.Length
}

// DecodeBool implements CInPacket.
func (p *iPacket) DecodeBool() bool {
	return p.Decode1() == 1
}

// Decode1 implements CInPacket.
func (p *iPacket) Decode1() uint8 {
	if p.GetRemain() <= 0 {
		return 0
	}
	result := p.RecvBuff[p.Offset]
	p.Offset += 1
	return result
}

// Decode1s implements CInPacket.
func (p *iPacket) Decode1s() int8 {
	return int8(p.Decode1())
}

// Decode2 implements CInPacket.
func (p *iPacket) Decode2() uint16 {
	if p.GetRemain() < 2 {
		return 0
	}
	result := uint16(p.RecvBuff[p.Offset]) | uint16(p.RecvBuff[p.Offset+1])<<8
	p.Offset += 2
	return result
}

// Decode2s implements CInPacket.
func (p *iPacket) Decode2s() int16 {
	return int16(p.Decode2())
}

// Decode4 implements CInPacket.
func (p *iPacket) Decode4() uint32 {
	if p.GetRemain() < 4 {
		return 0
	}
	result := binary.LittleEndian.Uint32(p.RecvBuff[p.Offset:])
	p.Offset += 4
	return result
}

// Decode4s implements CInPacket.
func (p *iPacket) Decode4s() int32 {
	return int32(p.Decode4())
}

// Decode8 implements CInPacket.
func (p *iPacket) Decode8() uint64 {
	if p.GetRemain() < 8 {
		return 0
	}
	result := binary.LittleEndian.Uint64(p.RecvBuff[p.Offset:])
	p.Offset += 8
	return result
}

// Decode8s implements CInPacket.
func (p *iPacket) Decode8s() int64 {
	return int64(p.Decode8())
}

// DecodeFT implements CInPacket.
func (p *iPacket) DecodeFT() time.Time {
	// FileTime is in 100-nanosecond intervals
	// Convert to nanoseconds by multiplying by 100
	// FileTime epoch is January 1, 1601
	// Unix epoch is January 1, 1970
	// Calculate the difference between the two in nanoseconds
	ft := p.Decode8s()
	nano := (ft - fileTimeEpochDiff) * 100
	return time.Unix(0, nano)
}

// DecodeStr implements CInPacket.
func (p *iPacket) DecodeStr() string {
	if p.GetRemain() < 2 {
		return ""
	}
	strLen := p.Decode2()
	if p.GetRemain() < int(strLen) {
		return ""
	}
	start := p.Offset
	end := p.Offset + int(strLen)
	str := string(p.RecvBuff[start:end])
	p.Offset = end
	return str
}

// DecodeLocalStr implements CInPacket.
func (p *iPacket) DecodeLocalStr() string {
	strLen := p.Decode2()
	buf := p.DecodeBuffer(int(strLen))
	return GetLangStr(buf)
}

// DecodeLocalName implements CInPacket.
func (p *iPacket) DecodeLocalName() string {
	buf := p.DecodeBuffer(13)
	return GetLangStr(buf)
}

// DecodeBuffer implements CInPacket.
func (p *iPacket) DecodeBuffer(uSize int) []byte {
	if p.GetRemain() < uSize {
		return nil
	}
	if uSize < 0 {
		uSize = 0
	}
	result := make([]byte, uSize)
	for i := range uSize {
		result[i] = byte(p.Decode1())
	}
	return result
}

// DumpString implements CInPacket.
func (p *iPacket) DumpString(nSize int) string {
	length := len(p.RecvBuff)
	if nSize <= 0 || nSize > length {
		nSize = length
	}
	var builder strings.Builder
	for i := range nSize {
		v := p.RecvBuff[i]
		builder.WriteString(fmt.Sprintf("%02X", v))
		if i < nSize-1 {
			builder.WriteString(" ")
		}
	}
	return builder.String()
}

// Clear implements CInPacket.
func (p *iPacket) Clear() {
	p.Length = 0
	p.Offset = 0
	p.RecvBuff = p.RecvBuff[:0]
}
