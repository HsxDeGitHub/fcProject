package cartridge

type Mapper4 struct {
	prg    []byte
	chr    []byte
	mirror uint8
}

func NewMapper4(prg, chr []byte, mirror uint8) *Mapper4 {
	return &Mapper4{prg: prg, chr: chr, mirror: mirror}
}

func (m *Mapper4) PRGRead(addr uint16) uint8 {
	if addr < 0x8000 {
		return 0
	}
	offset := int(addr - 0x8000)
	if offset >= len(m.prg) {
		return 0
	}
	return m.prg[offset]
}

func (m *Mapper4) PRGWrite(addr uint16, data uint8)   {}
func (m *Mapper4) CHRRead(addr uint16) uint8           { return m.chr[int(addr)%len(m.chr)] }
func (m *Mapper4) CHRWrite(addr uint16, data uint8)    {}
func (m *Mapper4) MirrorMode() uint8                   { return m.mirror }
