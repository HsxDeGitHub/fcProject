package cartridge

type Mapper4 struct {
	prg        []byte
	chr        []byte
	bankSelect uint8
	prgMode    uint8
	chrMode    uint8
	mirror     uint8
	registers  [8]uint8
}

func NewMapper4(prg, chr []byte, mirror uint8) *Mapper4 {
	m := &Mapper4{
		prg:    prg,
		chr:    chr,
		mirror: mirror,
	}
	// Default PRG banks: last two at $C000-$FFFF
	prgBanks := len(prg) / 8192
	if prgBanks > 1 {
		m.registers[6] = 0
		m.registers[7] = 1
	} else {
		m.registers[6] = 0
		m.registers[7] = 0
	}
	return m
}

func (m *Mapper4) PRGRead(addr uint16) uint8 {
	prgBanks := len(m.prg) / 8192
	if prgBanks == 0 {
		prgBanks = 1
	}

	var bank int
	switch {
	case addr < 0xA000: // $8000-$9FFF
		bank = int(m.registers[6]) * 8192
	case addr < 0xC000: // $A000-$BFFF
		bank = int(m.registers[7]) * 8192
	case addr < 0xE000: // $C000-$DFFF (second-to-last)
		bank = (prgBanks - 2) * 8192
	default: // $E000-$FFFF (last)
		bank = (prgBanks - 1) * 8192
	}

	offset := bank + int(addr&0x1FFF)
	if offset >= len(m.prg) {
		offset %= len(m.prg)
	}
	return m.prg[offset]
}

func (m *Mapper4) PRGWrite(addr uint16, data uint8) {
	switch {
	case addr >= 0x8000 && addr < 0xA000:
		if addr&1 == 0 {
			// Even: select register
			m.bankSelect = data & 0x07
			m.prgMode = (data >> 6) & 1
			m.chrMode = (data >> 7) & 1
		} else {
			// Odd: write data
			m.registers[m.bankSelect] = data
		}
	case addr >= 0xA000 && addr < 0xC000:
		if addr&1 == 0 {
			m.mirror = data & 1
		}
	// IRQ handling at $C000-$FFFF (simplified: we just acknowledge)
	default:
		// IRQ registers - basic handling for compatibility
		// $C000/$C001, $E000/$E001 pairs
	}
}

func (m *Mapper4) CHRRead(addr uint16) uint8 {
	bank := m.getCHRBank(addr)
	offset := bank*1024 + int(addr&0x3FF)
	if offset >= len(m.chr) {
		offset %= len(m.chr)
	}
	return m.chr[offset]
}

func (m *Mapper4) CHRWrite(addr uint16, data uint8) {
	bank := m.getCHRBank(addr)
	offset := bank*1024 + int(addr&0x3FF)
	if offset < len(m.chr) {
		m.chr[offset] = data
	}
}

func (m *Mapper4) getCHRBank(addr uint16) int {
	reg := 0
	switch {
	case addr < 0x0800:
		reg = 0 // 2KB bank, low bit ignored
	case addr < 0x1000:
		reg = 1 // 2KB bank, low bit ignored
	case addr < 0x1400:
		reg = 2
	case addr < 0x1800:
		reg = 3
	case addr < 0x1C00:
		reg = 4
	default:
		reg = 5
	}
	return int(m.registers[reg])
}

func (m *Mapper4) MirrorMode() uint8 { return m.mirror }
