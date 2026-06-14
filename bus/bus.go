package bus

type CartridgeInterface interface {
	PRGRead(addr uint16) uint8
	PRGWrite(addr uint16, data uint8)
	CHRRead(addr uint16) uint8
	CHRWrite(addr uint16, data uint8)
}

type PPUInterface interface {
	ReadRegister(addr uint16) uint8
	WriteRegister(addr uint16, data uint8)
	RenderFrame() []byte
}

type APUInterface interface {
	WriteRegister(addr uint16, data uint8)
	ReadStatus() uint8
	GenerateSamples() []float32
}

type InputInterface interface {
	Read() uint8
	Write(data uint8)
}

type Bus struct {
	RAM  [2048]uint8
	Cart CartridgeInterface
	PPU  PPUInterface
	APU  APUInterface
	Inp  InputInterface
}

func NewBus(cart CartridgeInterface, ppu PPUInterface, apu APUInterface, inp InputInterface) *Bus {
	return &Bus{
		Cart: cart,
		PPU:  ppu,
		APU:  apu,
		Inp:  inp,
	}
}

func (b *Bus) Read(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return b.RAM[addr&0x07FF]
	case addr < 0x4000:
		if b.PPU != nil {
			return b.PPU.ReadRegister(addr & 0x2007)
		}
		return 0
	case addr == 0x4016:
		if b.Inp != nil {
			return b.Inp.Read()
		}
		return 0
	case addr == 0x4015:
		if b.APU != nil {
			return b.APU.ReadStatus()
		}
		return 0
	case addr >= 0x4020:
		if b.Cart != nil {
			return b.Cart.PRGRead(addr)
		}
		return 0
	default:
		return 0
	}
}

func (b *Bus) Write(addr uint16, data uint8) {
	switch {
	case addr < 0x2000:
		b.RAM[addr&0x07FF] = data
	case addr < 0x4000:
		if b.PPU != nil {
			b.PPU.WriteRegister(addr&0x2007, data)
		}
	case addr == 0x4014:
		if b.PPU != nil {
			b.PPU.WriteRegister(0x4014, data)
			b.doDMA(data)
		}
	case addr == 0x4016:
		if b.Inp != nil {
			b.Inp.Write(data)
		}
	case addr >= 0x4000 && addr <= 0x4017:
		if b.APU != nil {
			b.APU.WriteRegister(addr, data)
		}
	case addr >= 0x4020:
		if b.Cart != nil {
			b.Cart.PRGWrite(addr, data)
		}
	}
}

func (b *Bus) doDMA(data uint8) {
	baseAddr := uint16(data) << 8
	for i := uint16(0); i < 256; i++ {
		val := b.Read(baseAddr + i)
		b.PPU.WriteRegister(0x2004, val)
	}
}
