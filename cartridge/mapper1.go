package cartridge

type Mapper1 struct {
	prg        []byte
	chr        []byte
	shiftReg   uint8
	shiftCount int
	control    uint8
	chrBank0   uint8
	chrBank1   uint8
	prgBank    uint8
	mirror     uint8
}

func NewMapper1(prg, chr []byte, mirror uint8) *Mapper1 {
	return &Mapper1{
		prg:     prg,
		chr:     chr,
		mirror:  mirror,
		control: 0x0C, // Default: 16KB PRG mode, fixed high bank
		chrBank0: 0,
		chrBank1: 0,
		prgBank: 0,
	}
}

func (m *Mapper1) PRGRead(addr uint16) uint8 {
	prgBanks := len(m.prg) / 16384
	if prgBanks == 0 {
		prgBanks = 1
	}

	var bank int
	bit3 := (m.control >> 3) & 1
	bit2 := (m.control >> 2) & 1

	if bit2 == 0 {
		// 32KB mode: ignore low bit of prgBank
		bank = int(m.prgBank&0x0E) * 16384
	} else {
		if bit3 == 0 {
			// $8000-$BFFF switchable, $C000-$FFFF fixed to last
			if addr < 0xC000 {
				bank = int(m.prgBank) * 16384
			} else {
				bank = (prgBanks - 1) * 16384
			}
		} else {
			// $8000-$BFFF fixed to first, $C000-$FFFF switchable
			if addr < 0xC000 {
				bank = 0
			} else {
				bank = int(m.prgBank) * 16384
			}
		}
	}

	offset := bank + int(addr&0x3FFF)
	if offset >= len(m.prg) {
		offset %= len(m.prg)
	}
	return m.prg[offset]
}

func (m *Mapper1) PRGWrite(addr uint16, data uint8) {
	// Reset on bit 7
	if data&0x80 != 0 {
		m.shiftReg = 0
		m.shiftCount = 0
		m.control |= 0x0C // Set 16KB PRG mode bits
		return
	}

	// Shift in bit 0
	m.shiftReg = (m.shiftReg >> 1) | ((data & 1) << 4)
	m.shiftCount++

	if m.shiftCount < 5 {
		return
	}

	// 5 bits loaded, store to register
	val := m.shiftReg
	m.shiftReg = 0
	m.shiftCount = 0

	switch {
	case addr < 0xA000: // $8000-$9FFF: Control
		m.control = val
		m.mirror = val & 0x03
	case addr < 0xC000: // $A000-$BFFF: CHR bank 0
		m.chrBank0 = val
	case addr < 0xE000: // $C000-$DFFF: CHR bank 1
		m.chrBank1 = val
	default: // $E000-$FFFF: PRG bank
		m.prgBank = val
	}
}

func (m *Mapper1) CHRRead(addr uint16) uint8 {
	chrBanks := len(m.chr) / 4096
	if chrBanks == 0 {
		chrBanks = 1
	}

	var bank int
	if m.control&0x10 != 0 {
		// 4KB CHR mode
		if addr < 0x1000 {
			bank = int(m.chrBank0) * 4096
		} else {
			bank = int(m.chrBank1) * 4096
		}
	} else {
		// 8KB CHR mode
		bank = int(m.chrBank0&0x1E) * 4096
	}

	offset := bank + int(addr&0xFFF)
	if offset >= len(m.chr) {
		offset %= len(m.chr)
	}
	return m.chr[offset]
}

func (m *Mapper1) CHRWrite(addr uint16, data uint8) {
	chrBanks := len(m.chr) / 4096
	if chrBanks == 0 {
		chrBanks = 1
	}

	var bank int
	if m.control&0x10 != 0 {
		if addr < 0x1000 {
			bank = int(m.chrBank0) * 4096
		} else {
			bank = int(m.chrBank1) * 4096
		}
	} else {
		bank = int(m.chrBank0&0x1E) * 4096
	}

	offset := bank + int(addr&0xFFF)
	if offset < len(m.chr) {
		m.chr[offset] = data
	}
}

func (m *Mapper1) MirrorMode() uint8 { return m.mirror }
