package ppu

type CartridgeInterface interface {
	CHRRead(addr uint16) uint8
	CHRWrite(addr uint16, data uint8)
}

const (
	ScreenWidth  = 256
	ScreenHeight = 240
)

type PPU struct {
	Cart CartridgeInterface

	Ctrl    uint8 // $2000
	Mask    uint8 // $2001
	Status  uint8 // $2002
	OAMAddr uint8 // $2003
	ScrollX uint8
	ScrollY uint8

	V          uint16
	T          uint16
	FineX      uint8
	W          bool
	scrollLatch bool

	VRAM    [2048]uint8
	OAM     [256]uint8
	Palette [32]uint8

	Frame     [ScreenWidth * ScreenHeight * 4]uint8
	readBuffer uint8

	Scanline int
	Cycle    int
	NMI      bool
}

func New(cart CartridgeInterface) *PPU {
	return &PPU{Cart: cart}
}

func (p *PPU) ReadRegister(addr uint16) uint8 {
	switch addr {
	case 0x2002:
		result := p.Status
		p.Status &^= 0x80
		p.W = false
		p.scrollLatch = false
		return result
	case 0x2007:
		return p.readData()
	default:
		return 0
	}
}

func (p *PPU) WriteRegister(addr uint16, data uint8) {
	switch addr {
	case 0x2000:
		p.Ctrl = data
		p.T = (p.T & 0xF3FF) | ((uint16(data) & 0x03) << 10)
	case 0x2001:
		p.Mask = data
	case 0x2003:
		p.OAMAddr = data
	case 0x2004:
		p.OAM[p.OAMAddr] = data
		p.OAMAddr++
	case 0x2005:
		if !p.scrollLatch {
			p.ScrollX = data
			p.FineX = data & 0x07
			p.T = (p.T & 0xFFE0) | (uint16(data) >> 3)
		} else {
			p.ScrollY = data
			p.T = (p.T & 0x8C1F) | ((uint16(data) & 0x07) << 12) | ((uint16(data) & 0xF8) << 2)
		}
		p.scrollLatch = !p.scrollLatch
	case 0x2006:
		if !p.W {
			p.T = (p.T & 0x00FF) | ((uint16(data) & 0x3F) << 8)
		} else {
			p.T = (p.T & 0xFF00) | uint16(data)
			p.V = p.T
		}
		p.W = !p.W
	case 0x2007:
		p.writeData(data)
	}
}

func (p *PPU) readData() uint8 {
	addr := p.V & 0x3FFF
	p.incrementVRAM()
	switch {
	case addr < 0x2000:
		val := p.readBuffer
		p.readBuffer = p.Cart.CHRRead(addr)
		return val
	case addr < 0x3F00:
		val := p.readBuffer
		p.readBuffer = p.VRAM[addr&0x07FF]
		return val
	default:
		idx := p.paletteIndex(addr)
		val := p.Palette[idx]
		p.readBuffer = p.VRAM[addr&0x07FF]
		return val
	}
}

func (p *PPU) writeData(data uint8) {
	addr := p.V & 0x3FFF
	p.incrementVRAM()
	switch {
	case addr < 0x2000:
		p.Cart.CHRWrite(addr, data)
	case addr < 0x3F00:
		p.VRAM[addr&0x07FF] = data
	default:
		idx := p.paletteIndex(addr)
		p.Palette[idx] = data
	}
}

func (p *PPU) incrementVRAM() {
	if p.Ctrl&0x04 != 0 {
		p.V += 32
	} else {
		p.V++
	}
}

func (p *PPU) paletteIndex(addr uint16) uint8 {
	idx := uint8(addr & 0x1F)
	// $3F10/$3F14/$3F18/$3F1C mirror $3F00/$3F04/$3F08/$3F0C
	if idx >= 16 && idx&3 == 0 {
		idx -= 16
	}
	return idx
}

func (p *PPU) RenderFrame() []byte {
	return p.Frame[:]
}
