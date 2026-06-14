package cpu

// Bus is the memory interface the CPU uses to read and write.
type Bus interface {
	Read(addr uint16) uint8
	Write(addr uint16, data uint8)
}

// Processor status flags.
const (
	FlagC = 1 << 0 // Carry
	FlagZ = 1 << 1 // Zero
	FlagI = 1 << 2 // Interrupt Disable
	FlagD = 1 << 3 // Decimal
	FlagB = 1 << 4 // Break
	FlagU = 1 << 5 // Unused (always 1 in pushed P)
	FlagV = 1 << 6 // Overflow
	FlagN = 1 << 7 // Negative
)

// CPU represents the MOS 6502 processor.
type CPU struct {
	A     uint8   // Accumulator
	X     uint8   // X index register
	Y     uint8   // Y index register
	SP    uint8   // Stack pointer
	PC    uint16  // Program counter
	P     uint8   // Processor status

	Bus    Bus
	Cycles uint64
	NMI    bool
	IRQ    bool
}

// New creates a new CPU with the given bus.
func New(bus Bus) *CPU {
	return &CPU{
		Bus: bus,
		SP:  0xFD,
		P:   0x34 | FlagU, // unused bit 5 set, I=1
	}
}

// Reset simulates a CPU reset: reads the reset vector and initialises registers.
func (c *CPU) Reset() {
	c.PC = uint16(c.Bus.Read(0xFFFC)) | uint16(c.Bus.Read(0xFFFD))<<8
	c.SP = 0xFD
	c.P = 0x34 | FlagU
	c.A = 0
	c.X = 0
	c.Y = 0
}

// Step fetches and executes a single instruction. Returns the number of cycles taken.
func (c *CPU) Step() int {
	if c.NMI {
		c.nmi()
		c.NMI = false
	}

	opcode := c.Bus.Read(c.PC)
	c.PC++
	cycles := c.execute(opcode)
	c.Cycles += uint64(cycles)
	return cycles
}

// RunCycles executes instructions until at least targetCycles have elapsed.
func (c *CPU) RunCycles(targetCycles uint64) {
	for c.Cycles < targetCycles {
		c.Step()
	}
}

func (c *CPU) nmi() {
	c.push16(c.PC)
	c.push8(c.P | FlagB | FlagU)
	c.P |= FlagI
	c.PC = uint16(c.Bus.Read(0xFFFA)) | uint16(c.Bus.Read(0xFFFB))<<8
	c.Cycles += 7
}

// --- Flag helpers ---

func (c *CPU) setZN(val uint8) {
	if val == 0 {
		c.P |= FlagZ
	} else {
		c.P &^= FlagZ
	}
	if val&0x80 != 0 {
		c.P |= FlagN
	} else {
		c.P &^= FlagN
	}
}

// --- Addressing modes ---

// imm returns the immediate operand and advances PC.
func (c *CPU) imm() uint8 {
	v := c.Bus.Read(c.PC)
	c.PC++
	return v
}

// abs returns a 16-bit absolute address from the next two bytes and advances PC.
func (c *CPU) abs() uint16 {
	lo := c.Bus.Read(c.PC)
	c.PC++
	hi := c.Bus.Read(c.PC)
	c.PC++
	return uint16(lo) | uint16(hi)<<8
}

// absX returns abs() + X. Sets *cross to true if a page boundary was crossed.
func (c *CPU) absX(cross *bool) uint16 {
	base := c.abs()
	addr := base + uint16(c.X)
	if cross != nil && (addr&0xFF00) != (base&0xFF00) {
		*cross = true
	}
	return addr
}

// absY returns abs() + Y. Sets *cross to true if a page boundary was crossed.
func (c *CPU) absY(cross *bool) uint16 {
	base := c.abs()
	addr := base + uint16(c.Y)
	if cross != nil && (addr&0xFF00) != (base&0xFF00) {
		*cross = true
	}
	return addr
}

// zp reads the zero-page address from the byte at PC-1.
// The caller must have already incremented PC past the zero-page byte.
func (c *CPU) zp() uint16 {
	return uint16(c.Bus.Read(c.PC - 1))
}

// zpX returns the zero-page address (from PC-1) plus X, wrapping within zero page.
func (c *CPU) zpX() uint16 {
	return uint16(uint8(c.zp()) + c.X)
}

// zpY returns the zero-page address (from PC-1) plus Y, wrapping within zero page.
func (c *CPU) zpY() uint16 {
	return uint16(uint8(c.zp()) + c.Y)
}

// ind returns the indirect address (used by JMP indirect). Implements the 6502
// page-boundary bug: when the pointer straddles a page, the high byte wraps.
func (c *CPU) ind() uint16 {
	lo := c.Bus.Read(c.PC)
	c.PC++
	hi := c.Bus.Read(c.PC)
	c.PC++
	ptr := uint16(lo) | uint16(hi)<<8
	addrLo := c.Bus.Read(ptr)
	addrHi := c.Bus.Read(ptr&0xFF00 | ((ptr + 1) & 0x00FF))
	return uint16(addrLo) | uint16(addrHi)<<8
}

// indX returns (indirect, X): reads the ptr at (zp + X) in zero page.
func (c *CPU) indX() uint16 {
	ptr := uint8(c.zp()) + c.X
	lo := c.Bus.Read(uint16(ptr))
	hi := c.Bus.Read(uint16(ptr + 1))
	return uint16(lo) | uint16(hi)<<8
}

// indY returns (indirect), Y. Reads a 16-bit pointer from zero page and adds Y.
// Sets *cross to true if a page boundary was crossed.
func (c *CPU) indY(cross *bool) uint16 {
	ptr := uint8(c.zp())
	lo := c.Bus.Read(uint16(ptr))
	hi := c.Bus.Read(uint16(ptr + 1))
	addr := uint16(lo) | uint16(hi)<<8
	oldPage := addr & 0xFF00
	addr += uint16(c.Y)
	if cross != nil && (addr&0xFF00) != oldPage {
		*cross = true
	}
	return addr
}

// rel returns the branch target address. Reads the signed offset and adds it to
// the current PC (which already points to the next instruction).
func (c *CPU) rel() uint16 {
	offset := int8(c.Bus.Read(c.PC))
	c.PC++
	return uint16(int32(c.PC) + int32(offset))
}

// --- Stack operations ---

func (c *CPU) push8(v uint8) {
	c.Bus.Write(0x0100|uint16(c.SP), v)
	c.SP--
}

func (c *CPU) push16(v uint16) {
	c.push8(uint8(v >> 8))
	c.push8(uint8(v & 0xFF))
}

func (c *CPU) pop8() uint8 {
	c.SP++
	return c.Bus.Read(0x0100 | uint16(c.SP))
}

func (c *CPU) pop16() uint16 {
	lo := c.pop8()
	hi := c.pop8()
	return uint16(lo) | uint16(hi)<<8
}

// --- Instruction dispatch ---

func (c *CPU) execute(opcode uint8) int {
	switch opcode {
	// ===== LDA - Load Accumulator =====
	case 0xA9: // immediate
		c.A = c.imm()
		c.setZN(c.A)
		return 2
	case 0xA5: // zero page
		c.PC++ // skip operand byte (zp adds read from PC-1)
		c.A = c.Bus.Read(c.zp())
		c.setZN(c.A)
		return 3
	case 0xB5: // zero page, X
		c.PC++
		c.A = c.Bus.Read(c.zpX())
		c.setZN(c.A)
		return 4
	case 0xAD: // absolute
		c.A = c.Bus.Read(c.abs())
		c.setZN(c.A)
		return 4
	case 0xBD: // absolute, X
		var cross bool
		addr := c.absX(&cross)
		c.A = c.Bus.Read(addr)
		c.setZN(c.A)
		if cross {
			return 5
		}
		return 4
	case 0xB9: // absolute, Y
		var cross bool
		addr := c.absY(&cross)
		c.A = c.Bus.Read(addr)
		c.setZN(c.A)
		if cross {
			return 5
		}
		return 4
	case 0xA1: // (indirect, X)
		c.PC++
		c.A = c.Bus.Read(c.indX())
		c.setZN(c.A)
		return 6
	case 0xB1: // (indirect), Y
		c.PC++
		var cross bool
		addr := c.indY(&cross)
		c.A = c.Bus.Read(addr)
		c.setZN(c.A)
		if cross {
			return 6
		}
		return 5

	// ===== LDX - Load X =====
	case 0xA2: // immediate
		c.X = c.imm()
		c.setZN(c.X)
		return 2
	case 0xA6: // zero page
		c.PC++
		c.X = c.Bus.Read(c.zp())
		c.setZN(c.X)
		return 3
	case 0xB6: // zero page, Y
		c.PC++
		c.X = c.Bus.Read(c.zpY())
		c.setZN(c.X)
		return 4
	case 0xAE: // absolute
		c.X = c.Bus.Read(c.abs())
		c.setZN(c.X)
		return 4
	case 0xBE: // absolute, Y
		var cross bool
		addr := c.absY(&cross)
		c.X = c.Bus.Read(addr)
		c.setZN(c.X)
		if cross {
			return 5
		}
		return 4

	// ===== LDY - Load Y =====
	case 0xA0: // immediate
		c.Y = c.imm()
		c.setZN(c.Y)
		return 2
	case 0xA4: // zero page
		c.PC++
		c.Y = c.Bus.Read(c.zp())
		c.setZN(c.Y)
		return 3
	case 0xB4: // zero page, X
		c.PC++
		c.Y = c.Bus.Read(c.zpX())
		c.setZN(c.Y)
		return 4
	case 0xAC: // absolute
		c.Y = c.Bus.Read(c.abs())
		c.setZN(c.Y)
		return 4
	case 0xBC: // absolute, X
		var cross bool
		addr := c.absX(&cross)
		c.Y = c.Bus.Read(addr)
		c.setZN(c.Y)
		if cross {
			return 5
		}
		return 4

	// ===== STA - Store Accumulator =====
	case 0x85: // zero page
		c.PC++
		c.Bus.Write(c.zp(), c.A)
		return 3
	case 0x95: // zero page, X
		c.PC++
		c.Bus.Write(c.zpX(), c.A)
		return 4
	case 0x8D: // absolute
		c.Bus.Write(c.abs(), c.A)
		return 4
	case 0x9D: // absolute, X
		var cross bool
		addr := c.absX(&cross)
		_ = cross // STA abs,X always takes 5 cycles
		c.Bus.Write(addr, c.A)
		return 5
	case 0x99: // absolute, Y
		var cross bool
		addr := c.absY(&cross)
		_ = cross // STA abs,Y always takes 5 cycles
		c.Bus.Write(addr, c.A)
		return 5
	case 0x81: // (indirect, X)
		c.PC++
		c.Bus.Write(c.indX(), c.A)
		return 6
	case 0x91: // (indirect), Y
		c.PC++
		var cross bool
		addr := c.indY(&cross)
		_ = cross // STA (ind),Y always takes 6 cycles
		c.Bus.Write(addr, c.A)
		return 6

	// ===== STX - Store X =====
	case 0x86: // zero page
		c.PC++
		c.Bus.Write(c.zp(), c.X)
		return 3
	case 0x96: // zero page, Y
		c.PC++
		c.Bus.Write(c.zpY(), c.X)
		return 4
	case 0x8E: // absolute
		c.Bus.Write(c.abs(), c.X)
		return 4

	// ===== STY - Store Y =====
	case 0x84: // zero page
		c.PC++
		c.Bus.Write(c.zp(), c.Y)
		return 3
	case 0x94: // zero page, X
		c.PC++
		c.Bus.Write(c.zpX(), c.Y)
		return 4
	case 0x8C: // absolute
		c.Bus.Write(c.abs(), c.Y)
		return 4

	// ===== Register transfers =====
	case 0xAA: // TAX - Transfer A to X
		c.X = c.A
		c.setZN(c.X)
		return 2
	case 0x8A: // TXA - Transfer X to A
		c.A = c.X
		c.setZN(c.A)
		return 2
	case 0xA8: // TAY - Transfer A to Y
		c.Y = c.A
		c.setZN(c.Y)
		return 2
	case 0x98: // TYA - Transfer Y to A
		c.A = c.Y
		c.setZN(c.A)
		return 2
	case 0xBA: // TSX - Transfer SP to X
		c.X = c.SP
		c.setZN(c.X)
		return 2
	case 0x9A: // TXS - Transfer X to SP
		c.SP = c.X
		return 2

	// ===== Increments and Decrements =====
	case 0xE8: // INX
		c.X++
		c.setZN(c.X)
		return 2
	case 0xC8: // INY
		c.Y++
		c.setZN(c.Y)
		return 2
	case 0xCA: // DEX
		c.X--
		c.setZN(c.X)
		return 2
	case 0x88: // DEY
		c.Y--
		c.setZN(c.Y)
		return 2

	// ===== INC / DEC zero page =====
	case 0xE6: // INC zp
		c.PC++
		addr := c.zp()
		v := c.Bus.Read(addr) + 1
		c.Bus.Write(addr, v)
		c.setZN(v)
		return 5
	case 0xC6: // DEC zp
		c.PC++
		addr := c.zp()
		v := c.Bus.Read(addr) - 1
		c.Bus.Write(addr, v)
		c.setZN(v)
		return 5

	// ===== Flag instructions =====
	case 0x18: // CLC - Clear Carry
		c.P &^= FlagC
		return 2
	case 0x38: // SEC - Set Carry
		c.P |= FlagC
		return 2
	case 0x58: // CLI - Clear Interrupt Disable
		c.P &^= FlagI
		return 2
	case 0x78: // SEI - Set Interrupt Disable
		c.P |= FlagI
		return 2
	case 0xD8: // CLD - Clear Decimal
		c.P &^= FlagD
		return 2
	case 0xB8: // CLV - Clear Overflow
		c.P &^= FlagV
		return 2

	// ===== Stack instructions =====
	case 0x48: // PHA - Push Accumulator
		c.push8(c.A)
		return 3
	case 0x68: // PLA - Pull Accumulator
		c.A = c.pop8()
		c.setZN(c.A)
		return 4
	case 0x08: // PHP - Push Processor Status
		c.push8(c.P | FlagB | FlagU)
		return 3
	case 0x28: // PLP - Pull Processor Status
		c.P = c.pop8()&0xEF | FlagU
		return 4

	// ===== JMP =====
	case 0x6C: // indirect
		c.PC = c.ind()
		return 5

	// ===== NOP =====
	case 0xEA: // NOP
		return 2

	default:
		return 2
	}
}
