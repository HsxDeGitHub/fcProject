// cartridge/cartridge.go
package cartridge

import (
	"errors"
)

type Cartridge struct {
	PRG      []byte // PRG ROM 数据
	CHR      []byte // CHR ROM 数据
	PRGBanks int    // PRG ROM bank 数（16KB 单位）
	CHRBanks int    // CHR ROM bank 数（8KB 单位）
	Mapper   int    // Mapper 编号
	Mirror   int    // 0=水平, 1=垂直（PPU 需要此字段确定 nametable 镜像模式）
}

func Load(data []byte) (*Cartridge, error) {
	if len(data) < 16 {
		return nil, errors.New("ROM too small")
	}
	if data[0] != 'N' || data[1] != 'E' || data[2] != 'S' || data[3] != 0x1A {
		return nil, errors.New("invalid iNES magic")
	}

	prgBanks := int(data[4])
	chrBanks := int(data[5])
	flags6 := data[6]
	flags7 := data[7]

	// Mapper number: lower nibble from flags6, upper nibble from flags7
	mapper := int(flags6>>4) | int(flags7&0xF0)

	// 当前仅支持 Mapper 0
	if mapper != 0 {
		return nil, errors.New("unsupported mapper")
	}

	mirror := int(flags6 & 1)

	prgSize := prgBanks * 16384
	chrSize := chrBanks * 8192

	headerEnd := 16
	if len(data) < headerEnd+prgSize+chrSize {
		return nil, errors.New("ROM data truncated")
	}

	prgROM := make([]byte, prgSize)
	copy(prgROM, data[headerEnd:headerEnd+prgSize])

	var chrROM []byte
	if chrBanks == 0 {
		chrROM = make([]byte, 8192)
	} else {
		chrROM = make([]byte, chrSize)
		copy(chrROM, data[headerEnd+prgSize:headerEnd+prgSize+chrSize])
	}

	return &Cartridge{
		PRG:      prgROM,
		CHR:      chrROM,
		PRGBanks: prgBanks,
		CHRBanks: chrBanks,
		Mapper:   mapper,
		Mirror:   mirror,
	}, nil
}

func (c *Cartridge) PRGRead(addr uint16) uint8 {
	if addr < 0x8000 {
		return 0
	}
	offset := addr - 0x8000
	if c.PRGBanks == 1 {
		offset = offset & 0x3FFF
	}
	if int(offset) >= len(c.PRG) {
		return 0
	}
	return c.PRG[offset]
}

func (c *Cartridge) PRGWrite(addr uint16, data uint8) {
	// Mapper 0 不支持 PRG ROM 写入
}

func (c *Cartridge) CHRRead(addr uint16) uint8 {
	if int(addr) >= len(c.CHR) {
		return 0
	}
	return c.CHR[addr]
}

func (c *Cartridge) MirrorMode() uint8 {
	return uint8(c.Mirror)
}

func (c *Cartridge) CHRWrite(addr uint16, data uint8) {
	if c.CHRBanks == 0 {
		if int(addr) < len(c.CHR) {
			c.CHR[addr] = data
		}
	}
}
