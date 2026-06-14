package ppu

type CartridgeInterface interface {
	CHRRead(addr uint16) uint8
	CHRWrite(addr uint16, data uint8)
	MirrorMode() uint8
}

var NESPalette = [64]uint32{
	0x7C7C7C, 0x0000FC, 0x0000BC, 0x4428BC, 0x940084, 0xA80020, 0xA81000, 0x881400,
	0x503000, 0x007800, 0x006800, 0x005800, 0x004058, 0x000000, 0x000000, 0x000000,
	0xBCBCBC, 0x0078F8, 0x0058F8, 0x6844FC, 0xD800CC, 0xE40058, 0xF83800, 0xE45C10,
	0xAC7C00, 0x00B800, 0x00A800, 0x00A844, 0x008888, 0x000000, 0x000000, 0x000000,
	0xF8F8F8, 0x3CBCFC, 0x6888FC, 0x9878F8, 0xF878F8, 0xF85898, 0xF87858, 0xFCA044,
	0xF8B800, 0xB8F818, 0x58D854, 0x58F898, 0x00E8D8, 0x787878, 0x000000, 0x000000,
	0xFCFCFC, 0xA4E4FC, 0xB8B8F8, 0xD8B8F8, 0xF8B8F8, 0xF8A4C0, 0xF0D0B0, 0xFCE0A8,
	0xF8D878, 0xD8F878, 0xB8F8B8, 0xB8F8D8, 0x00FCFC, 0xF8D8F8, 0x000000, 0x000000,
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
	Latch      bool   // shared toggle for $2005/$2006
	mirrorMode uint8  // cartridge-determined nametable mirroring

	VRAM    [2048]uint8
	OAM     [256]uint8
	Palette [32]uint8

	Frame      [ScreenWidth * ScreenHeight * 4]uint8
	tileBuffer [ScreenWidth * ScreenHeight]uint8
	readBuffer uint8

	Scanline          int
	Cycle             int
	NMI               bool
	VblankReasserts   int    // re-assert VBlank N times to support multi-poll init
	PalWrites         uint64 // diagnostic: counts palette writes
}

func New(cart CartridgeInterface) *PPU {
	return &PPU{
		Cart:       cart,
		mirrorMode: cart.MirrorMode(),
	}
}

func (p *PPU) ReadRegister(addr uint16) uint8 {
	switch addr {
	case 0x2002:
		result := p.Status
		p.Status &^= 0x80
		p.Latch = false
		// Re-assert VBlank to support multiple VBlank poll loops
		if p.VblankReasserts > 0 {
			p.Status |= 0x80
			p.VblankReasserts--
		}
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
		if !p.Latch {
			p.ScrollX = data
			p.FineX = data & 0x07
			p.T = (p.T & 0xFFE0) | (uint16(data) >> 3)
		} else {
			p.ScrollY = data
			p.T = (p.T & 0x8C1F) | ((uint16(data) & 0x07) << 12) | ((uint16(data) & 0xF8) << 2)
		}
		p.Latch = !p.Latch
	case 0x2006:
		if !p.Latch {
			p.T = (p.T & 0x00FF) | ((uint16(data) & 0x3F) << 8)
		} else {
			p.T = (p.T & 0xFF00) | uint16(data)
			p.V = p.T
		}
		p.Latch = !p.Latch
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
		p.PalWrites++
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

func (p *PPU) paletteColor(idx uint8) (r, g, b, a uint8) {
	c := NESPalette[idx&0x3F]
	return uint8((c >> 16) & 0xFF), uint8((c >> 8) & 0xFF), uint8(c & 0xFF), 255
}

func (p *PPU) readVRAM(addr uint16) uint8 {
	addr &= 0x3FFF
	switch {
	case addr < 0x2000:
		return p.Cart.CHRRead(addr)
	case addr < 0x3F00:
		return p.VRAM[p.mirrorNameTable(addr)&0x0FFF]
	default:
		idx := p.paletteIndex(addr)
		return p.Palette[idx]
	}
}

func (p *PPU) readCHR(addr uint16) uint8 {
	return p.Cart.CHRRead(addr & 0x1FFF)
}

func (p *PPU) mirrorNameTable(addr uint16) uint16 {
	addr &= 0x3FFF
	if addr >= 0x3000 && addr < 0x3F00 {
		addr -= 0x1000 // $3000-$3EFF mirrors $2000-$2EFF
	}
	table := (addr - 0x2000) / 0x400
	switch {
	case p.mirrorMode != 0: // Vertical mirroring (cartridge-determined)
		switch table {
		case 0, 1:
			return addr
		case 2:
			return addr - 0x800
		case 3:
			return addr - 0x800
		}
	default: // Horizontal mirroring
		switch table {
		case 0:
			return addr
		case 1:
			return addr - 0x400
		case 2:
			return addr - 0x400
		case 3:
			return addr - 0x800
		}
	}
	return addr
}

func (p *PPU) RenderFrame() []byte {
	for i := range p.Frame {
		p.Frame[i] = 0
	}
	for i := range p.tileBuffer {
		p.tileBuffer[i] = 0
	}
	p.Status |= 0x80 // Set VBlank

	if p.Mask&0x08 != 0 {
		p.renderBackground()
	}
	if p.Mask&0x10 != 0 {
		p.renderSprites()
	}

	return p.Frame[:]
}

func (p *PPU) renderBackground() {
	baseNT := 0x2000 + uint16(p.Ctrl&0x03)*0x400
	basePT := uint16(0)
	if p.Ctrl&0x10 != 0 {
		basePT = 0x1000
	}

	for y := 0; y < 240; y++ {
		rawY := y + int(p.ScrollY)
		ntRow := uint16(0)
		if rawY >= 256 {
			ntRow = 0x800
		}
		realY := rawY % 256
		tileY := realY / 8
		pixelY := realY & 7

		for x := 0; x < 256; x++ {
			rawX := x + int(p.ScrollX)
			ntCol := uint16(0)
			if rawX >= 256 {
				ntCol = 0x400
			}
			realX := rawX % 256
			tileX := realX / 8
			pixelX := realX & 7

			ntAddr := baseNT + ntRow + ntCol + uint16(tileY)*32 + uint16(tileX)
			tileIdx := p.readVRAM(ntAddr)

			attrAddr := baseNT + ntRow + ntCol + 0x3C0 + uint16(tileY/4)*8 + uint16(tileX/4)
			attr := p.readVRAM(attrAddr)
			shift := ((tileY & 2) << 1) | (tileX & 2)
			paletteGroup := (attr >> shift) & 0x03

			ptAddr := basePT + uint16(tileIdx)*16 + uint16(pixelY)

			lo := p.readCHR(ptAddr)
			hi := p.readCHR(ptAddr + 8)
			bit := 7 - pixelX
			colorIdx := ((hi>>bit)&1)<<1 | ((lo>>bit)&1)

			// Store raw color index for sprite priority checks
			p.tileBuffer[y*256+x] = colorIdx

			if colorIdx == 0 {
				colorIdx = p.Palette[0] & 0x3F
			} else {
				palAddr := 0x3F00 + uint16(paletteGroup)*4 + uint16(colorIdx)
				colorIdx = p.readVRAM(palAddr) & 0x3F
			}

			r, g, b, a := p.paletteColor(colorIdx)
			offset := (y*256 + x) * 4
			p.Frame[offset] = r
			p.Frame[offset+1] = g
			p.Frame[offset+2] = b
			p.Frame[offset+3] = a
		}
	}
}

func (p *PPU) renderSprites() {
	spriteHeight := 8
	if p.Ctrl&0x20 != 0 {
		spriteHeight = 16
	}

	for i := 63; i >= 0; i-- {
		spriteY := int(p.OAM[i*4])
		tileIdx := p.OAM[i*4+1]
		attr := p.OAM[i*4+2]
		spriteX := int(p.OAM[i*4+3])

		// Y=0 sprites are hidden on real NES hardware
		if p.OAM[i*4] == 0 {
			continue
		}
		spriteY = spriteY + 1 // NES: sprites rendered one line lower

		if spriteY >= 240 || spriteX > 248 {
			continue
		}

		flipV := (attr & 0x80) != 0
		flipH := (attr & 0x40) != 0
		paletteGroup := (attr & 0x03) + 4

		basePT := uint16(0)
		if p.Ctrl&0x08 != 0 {
			basePT = 0x1000
		}
		if spriteHeight == 16 {
			basePT = uint16(tileIdx&0x01) * 0x1000
			tileIdx &= 0xFE
		}

		for py := 0; py < spriteHeight; py++ {
			drawY := spriteY + py
			if drawY < 0 || drawY >= 240 {
				continue
			}

			realPy := py
			if flipV {
				realPy = spriteHeight - 1 - py
			}

			ptAddr := basePT + uint16(tileIdx)*16 + uint16(realPy)
			if spriteHeight == 16 && realPy >= 8 {
				ptAddr = basePT + uint16(tileIdx+1)*16 + uint16(realPy-8)
			}

			lo := p.readCHR(ptAddr)
			hi := p.readCHR(ptAddr + 8)

			for px := 0; px < 8; px++ {
				drawX := spriteX + px
				if drawX < 0 || drawX >= 256 {
					continue
				}

				realPx := px
				if flipH {
					realPx = 7 - px
				}

				bit := 7 - realPx
				colorIdx := ((hi>>bit)&1)<<1 | ((lo>>bit)&1)

				if colorIdx == 0 {
					continue // transparent
				}

				palAddr := 0x3F00 + uint16(paletteGroup)*4 + uint16(colorIdx)
				cIdx := p.readVRAM(palAddr) & 0x3F

				r, g, b, a := p.paletteColor(cIdx)
				offset := (drawY*256 + drawX) * 4

				// Sprite priority: bit 5 set = render behind non-transparent background
				if attr&0x20 != 0 && p.tileBuffer[drawY*256+drawX] != 0 {
					continue
				}

				p.Frame[offset] = r
				p.Frame[offset+1] = g
				p.Frame[offset+2] = b
				p.Frame[offset+3] = a
			}
		}
	}
}
