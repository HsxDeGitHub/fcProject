package cartridge

type Mapper0 struct {
	prg    []byte
	chr    []byte
	chrRAM bool
}

func NewMapper0(prg, chr []byte, chrRAM bool) *Mapper0 {
	return &Mapper0{prg: prg, chr: chr, chrRAM: chrRAM}
}

func (m *Mapper0) PRGRead(addr uint16) uint8 {
	if addr < 0x8000 {
		return 0
	}
	offset := int(addr - 0x8000)
	if len(m.prg) == 16384 {
		offset &= 0x3FFF
	}
	if offset >= len(m.prg) {
		return 0
	}
	return m.prg[offset]
}

func (m *Mapper0) PRGWrite(addr uint16, data uint8) {}

func (m *Mapper0) CHRRead(addr uint16) uint8 {
	if int(addr) >= len(m.chr) {
		return 0
	}
	return m.chr[addr]
}

func (m *Mapper0) CHRWrite(addr uint16, data uint8) {
	if m.chrRAM && int(addr) < len(m.chr) {
		m.chr[addr] = data
	}
}

func (m *Mapper0) MirrorMode() uint8 { return 0 }
