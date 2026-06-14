// cartridge/cartridge.go
package cartridge

import (
	"errors"
)

type Mapper interface {
	PRGRead(addr uint16) uint8
	PRGWrite(addr uint16, data uint8)
	CHRRead(addr uint16) uint8
	CHRWrite(addr uint16, data uint8)
	MirrorMode() uint8
}

type Cartridge struct {
	PRG      []byte // PRG ROM 数据
	CHR      []byte // CHR ROM 数据
	PRGBanks int    // PRG ROM bank 数（16KB 单位）
	CHRBanks int    // CHR ROM bank 数（8KB 单位）
	Mapper   int    // Mapper 编号
	mapper   Mapper
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

	var m Mapper
	switch mapper {
	case 0:
		m = NewMapper0(prgROM, chrROM, chrBanks == 0)
	case 1:
		m = NewMapper1(prgROM, chrROM, uint8(mirror))
	case 4:
		m = NewMapper4(prgROM, chrROM, uint8(mirror))
	default:
		return nil, errors.New("unsupported mapper")
	}

	return &Cartridge{
		PRG:      prgROM,
		CHR:      chrROM,
		PRGBanks: prgBanks,
		CHRBanks: chrBanks,
		Mapper:   mapper,
		mapper:   m,
	}, nil
}

func (c *Cartridge) PRGRead(addr uint16) uint8 {
	if c.mapper != nil {
		return c.mapper.PRGRead(addr)
	}
	return 0
}

func (c *Cartridge) PRGWrite(addr uint16, data uint8) {
	if c.mapper != nil {
		c.mapper.PRGWrite(addr, data)
	}
}

func (c *Cartridge) CHRRead(addr uint16) uint8 {
	if c.mapper != nil {
		return c.mapper.CHRRead(addr)
	}
	return 0
}

func (c *Cartridge) CHRWrite(addr uint16, data uint8) {
	if c.mapper != nil {
		c.mapper.CHRWrite(addr, data)
	}
}

func (c *Cartridge) MirrorMode() uint8 {
	if c.mapper != nil {
		return c.mapper.MirrorMode()
	}
	return 0
}
