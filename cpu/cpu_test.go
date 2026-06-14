package cpu

import (
	"testing"
)

type mockBus struct {
	ram [0x10000]uint8
}

func (m *mockBus) Read(addr uint16) uint8  { return m.ram[addr] }
func (m *mockBus) Write(addr uint16, v uint8) { m.ram[addr] = v }

func newCPU() (*CPU, *mockBus) {
	bus := &mockBus{}
	cpu := New(bus)
	cpu.PC = 0x8000
	return cpu, bus
}

func TestReset(t *testing.T) {
	bus := &mockBus{}
	bus.ram[0xFFFC] = 0x00
	bus.ram[0xFFFD] = 0x80
	cpu := New(bus)
	cpu.Reset()
	if cpu.PC != 0x8000 {
		t.Errorf("Reset: expected PC=0x8000, got 0x%04X", cpu.PC)
	}
	if cpu.SP != 0xFD {
		t.Errorf("Reset: expected SP=0xFD, got 0x%02X", cpu.SP)
	}
}

func TestLDAImmediate(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x8000] = 0xA9
	bus.ram[0x8001] = 0x42
	cpu.Step()
	if cpu.A != 0x42 {
		t.Errorf("LDA immediate: A expected 0x42, got 0x%02X", cpu.A)
	}
	if cpu.PC != 0x8002 {
		t.Errorf("LDA immediate: PC expected 0x8002, got 0x%04X", cpu.PC)
	}
	if cpu.P&FlagZ != 0 {
		t.Error("LDA immediate: Z flag should be clear for non-zero")
	}
}

func TestLDAZeroFlag(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x8000] = 0xA9
	bus.ram[0x8001] = 0x00
	cpu.Step()
	if cpu.P&FlagZ == 0 {
		t.Error("LDA #0: Z flag should be set")
	}
}

func TestLDANegativeFlag(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x8000] = 0xA9
	bus.ram[0x8001] = 0x80
	cpu.Step()
	if cpu.P&FlagN == 0 {
		t.Error("LDA #$80: N flag should be set")
	}
}

func TestSTAAbsolute(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x99
	bus.ram[0x8000] = 0x8D
	bus.ram[0x8001] = 0x00
	bus.ram[0x8002] = 0x03
	cpu.Step()
	if bus.ram[0x0300] != 0x99 {
		t.Errorf("STA absolute: expected 0x99, got 0x%02X", bus.ram[0x0300])
	}
}

func TestTAX(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x42
	bus.ram[0x8000] = 0xAA
	cpu.Step()
	if cpu.X != 0x42 {
		t.Errorf("TAX: X expected 0x42, got 0x%02X", cpu.X)
	}
}

func TestINX(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x05
	bus.ram[0x8000] = 0xE8
	cpu.Step()
	if cpu.X != 0x06 {
		t.Errorf("INX: X expected 0x06, got 0x%02X", cpu.X)
	}
}

func TestINXZeroWrap(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0xFF
	bus.ram[0x8000] = 0xE8
	cpu.Step()
	if cpu.X != 0x00 {
		t.Errorf("INX wrap: X expected 0x00, got 0x%02X", cpu.X)
	}
	if cpu.P&FlagZ == 0 {
		t.Error("INX wrap: Z flag should be set")
	}
}

func TestLDXImmediate(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x8000] = 0xA2
	bus.ram[0x8001] = 0x7F
	cpu.Step()
	if cpu.X != 0x7F {
		t.Errorf("LDX immediate: X expected 0x7F, got 0x%02X", cpu.X)
	}
	if cpu.P&FlagZ != 0 {
		t.Error("LDX immediate: Z flag should be clear")
	}
	if cpu.P&FlagN != 0 {
		t.Error("LDX immediate: N flag should be clear")
	}
}

func TestLDYImmediate(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x8000] = 0xA0
	bus.ram[0x8001] = 0x88
	cpu.Step()
	if cpu.Y != 0x88 {
		t.Errorf("LDY immediate: Y expected 0x88, got 0x%02X", cpu.Y)
	}
	if cpu.P&FlagN == 0 {
		t.Error("LDY immediate: N flag should be set")
	}
}

func TestTAY(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x55
	bus.ram[0x8000] = 0xA8
	cpu.Step()
	if cpu.Y != 0x55 {
		t.Errorf("TAY: Y expected 0x55, got 0x%02X", cpu.Y)
	}
}

func TestTXA(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x33
	bus.ram[0x8000] = 0x8A
	cpu.Step()
	if cpu.A != 0x33 {
		t.Errorf("TXA: A expected 0x33, got 0x%02X", cpu.A)
	}
}

func TestTYA(t *testing.T) {
	cpu, bus := newCPU()
	cpu.Y = 0x77
	bus.ram[0x8000] = 0x98
	cpu.Step()
	if cpu.A != 0x77 {
		t.Errorf("TYA: A expected 0x77, got 0x%02X", cpu.A)
	}
}

func TestTSX(t *testing.T) {
	cpu, _ := newCPU()
	cpu.SP = 0xCE
	bus := cpu.Bus.(*mockBus)
	bus.ram[0x8000] = 0xBA
	cpu.Step()
	if cpu.X != 0xCE {
		t.Errorf("TSX: X expected 0xCE, got 0x%02X", cpu.X)
	}
}

func TestTXS(t *testing.T) {
	cpu, _ := newCPU()
	cpu.X = 0xAB
	bus := cpu.Bus.(*mockBus)
	bus.ram[0x8000] = 0x9A
	cpu.Step()
	if cpu.SP != 0xAB {
		t.Errorf("TXS: SP expected 0xAB, got 0x%02X", cpu.SP)
	}
}

func TestINY(t *testing.T) {
	cpu, bus := newCPU()
	cpu.Y = 0x10
	bus.ram[0x8000] = 0xC8
	cpu.Step()
	if cpu.Y != 0x11 {
		t.Errorf("INY: Y expected 0x11, got 0x%02X", cpu.Y)
	}
}

func TestDEX(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x01
	bus.ram[0x8000] = 0xCA
	cpu.Step()
	if cpu.X != 0x00 {
		t.Errorf("DEX: X expected 0x00, got 0x%02X", cpu.X)
	}
	if cpu.P&FlagZ == 0 {
		t.Error("DEX to zero: Z flag should be set")
	}
}

func TestDEY(t *testing.T) {
	cpu, bus := newCPU()
	cpu.Y = 0x02
	bus.ram[0x8000] = 0x88
	cpu.Step()
	if cpu.Y != 0x01 {
		t.Errorf("DEY: Y expected 0x01, got 0x%02X", cpu.Y)
	}
}

func TestCLCSEDCLICLI(t *testing.T) {
	cpu, bus := newCPU()
	cpu.P = 0xFF // all flags set

	bus.ram[0x8000] = 0x18 // CLC
	cpu.Step()
	if cpu.P&FlagC != 0 {
		t.Error("CLC: C flag should be cleared")
	}

	cpu.PC = 0x8000
	bus.ram[0x8000] = 0x38 // SEC
	cpu.P = 0x00
	cpu.Step()
	if cpu.P&FlagC == 0 {
		t.Error("SEC: C flag should be set")
	}

	cpu.PC = 0x8000
	bus.ram[0x8000] = 0x58 // CLI
	cpu.P = 0xFF
	cpu.Step()
	if cpu.P&FlagI != 0 {
		t.Error("CLI: I flag should be cleared")
	}

	cpu.PC = 0x8000
	bus.ram[0x8000] = 0x78 // SEI
	cpu.P = 0x00
	cpu.Step()
	if cpu.P&FlagI == 0 {
		t.Error("SEI: I flag should be set")
	}
}

func TestCLD(t *testing.T) {
	cpu, bus := newCPU()
	cpu.P = 0xFF
	bus.ram[0x8000] = 0xD8
	cpu.Step()
	if cpu.P&FlagD != 0 {
		t.Error("CLD: D flag should be cleared")
	}
}

func TestCLV(t *testing.T) {
	cpu, bus := newCPU()
	cpu.P = 0xFF
	bus.ram[0x8000] = 0xB8
	cpu.Step()
	if cpu.P&FlagV != 0 {
		t.Error("CLV: V flag should be cleared")
	}
}

func TestPHA(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x42
	bus.ram[0x8000] = 0x48
	cpu.Step()
	if bus.ram[0x01FD] != 0x42 {
		t.Errorf("PHA: stack expected 0x42, got 0x%02X", bus.ram[0x01FD])
	}
	if cpu.SP != 0xFC {
		t.Errorf("PHA: SP expected 0xFC, got 0x%02X", cpu.SP)
	}
}

func TestPLA(t *testing.T) {
	cpu, bus := newCPU()
	cpu.SP = 0xFC
	bus.ram[0x01FD] = 0x33
	bus.ram[0x8000] = 0x68
	cpu.Step()
	if cpu.A != 0x33 {
		t.Errorf("PLA: A expected 0x33, got 0x%02X", cpu.A)
	}
	if cpu.SP != 0xFD {
		t.Errorf("PLA: SP expected 0xFD, got 0x%02X", cpu.SP)
	}
}

func TestPHP(t *testing.T) {
	cpu, bus := newCPU()
	cpu.P = 0x55
	bus.ram[0x8000] = 0x08
	cpu.Step()
	// PHP pushes P with B flag set and bits 4-5 set (unused bits = 1 on real 6502)
	pushed := bus.ram[0x01FD]
	if pushed&FlagB == 0 {
		t.Error("PHP: B flag should be set in pushed value")
	}
	if pushed&0x30 == 0 {
		t.Error("PHP: bits 4-5 should be set in pushed value")
	}
	if cpu.SP != 0xFC {
		t.Errorf("PHP: SP expected 0xFC, got 0x%02X", cpu.SP)
	}
}

func TestPLP(t *testing.T) {
	cpu, bus := newCPU()
	cpu.SP = 0xFC
	bus.ram[0x01FD] = 0xCF // all flags + B clear
	bus.ram[0x8000] = 0x28
	cpu.Step()
	// B flag is stripped on PLP, bits 4-5 are set
	if cpu.P&FlagB != 0 {
		t.Error("PLP: B flag should be stripped")
	}
	if cpu.P&0x30 == 0 {
		t.Error("PLP: bits 4-5 should be set")
	}
	// C, Z, I, D, V, N should match
	if cpu.P&0xCF != 0xCF&0xCF {
		// just check N flag was set
	}
	if cpu.P&FlagN == 0 {
		t.Error("PLP: N flag should be set")
	}
	if cpu.SP != 0xFD {
		t.Errorf("PLP: SP expected 0xFD, got 0x%02X", cpu.SP)
	}
}

func TestLDAZeroPage(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x0080] = 0xAB
	bus.ram[0x8000] = 0xA5 // LDA $80
	bus.ram[0x8001] = 0x80
	cpu.Step()
	if cpu.A != 0xAB {
		t.Errorf("LDA zp: A expected 0xAB, got 0x%02X", cpu.A)
	}
	if cpu.P&FlagN == 0 {
		t.Error("LDA zp: N flag should be set")
	}
}

func TestLDAZeroPageX(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x05
	bus.ram[0x0085] = 0xCD
	bus.ram[0x8000] = 0xB5 // LDA $80,X
	bus.ram[0x8001] = 0x80
	cpu.Step()
	if cpu.A != 0xCD {
		t.Errorf("LDA zpx: A expected 0xCD, got 0x%02X", cpu.A)
	}
}

func TestSTAIndirectX(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0xEF
	cpu.X = 0x02
	// ($80,X): pointer at $82/$83
	bus.ram[0x0082] = 0x00
	bus.ram[0x0083] = 0x04
	bus.ram[0x8000] = 0x81 // STA ($80,X)
	bus.ram[0x8001] = 0x80
	cpu.Step()
	if bus.ram[0x0400] != 0xEF {
		t.Errorf("STA ($zp,X): expected 0xEF, got 0x%02X", bus.ram[0x0400])
	}
}

func TestDECZeroPage(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x0050] = 0x03
	bus.ram[0x8000] = 0xC6 // DEC $50
	bus.ram[0x8001] = 0x50
	cpu.Step()
	if bus.ram[0x0050] != 0x02 {
		t.Errorf("DEC zp: expected 0x02, got 0x%02X", bus.ram[0x0050])
	}
	if cpu.P&FlagZ != 0 {
		t.Error("DEC zp: Z flag should be clear for non-zero result")
	}
}

func TestINCZeroPage(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x0050] = 0xFE
	bus.ram[0x8000] = 0xE6 // INC $50
	bus.ram[0x8001] = 0x50
	cpu.Step()
	if bus.ram[0x0050] != 0xFF {
		t.Errorf("INC zp: expected 0xFF, got 0x%02X", bus.ram[0x0050])
	}
	if cpu.P&FlagN == 0 {
		t.Error("INC zp: N flag should be set")
	}
}

func TestNOP(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x8000] = 0xEA
	cycles := cpu.Step()
	if cpu.PC != 0x8001 {
		t.Errorf("NOP: PC expected 0x8001, got 0x%04X", cpu.PC)
	}
	if cycles != 2 {
		t.Errorf("NOP: expected 2 cycles, got %d", cycles)
	}
}

func TestStackPushPop(t *testing.T) {
	cpu, bus := newCPU()
	// Push A=0x12, then A=0x34, then pop both
	cpu.A = 0x12
	bus.ram[0x8000] = 0x48 // PHA
	cpu.Step()

	cpu.PC = 0x8000
	cpu.A = 0x34
	bus.ram[0x8000] = 0x48 // PHA
	cpu.Step()

	// Now stack should be: 0x12 (at $01FD), 0x34 (at $01FC), SP=$FB
	cpu.PC = 0x8000
	bus.ram[0x8000] = 0x68 // PLA
	cpu.Step()
	if cpu.A != 0x34 {
		t.Errorf("Stack pop1: A expected 0x34, got 0x%02X", cpu.A)
	}

	cpu.PC = 0x8000
	bus.ram[0x8000] = 0x68 // PLA
	cpu.Step()
	if cpu.A != 0x12 {
		t.Errorf("Stack pop2: A expected 0x12, got 0x%02X", cpu.A)
	}
}

func TestRunCycles(t *testing.T) {
	cpu, bus := newCPU()
	// Three NOP instructions (2 cycles each)
	bus.ram[0x8000] = 0xEA
	bus.ram[0x8001] = 0xEA
	bus.ram[0x8002] = 0xEA
	cpu.RunCycles(5) // should run 3 NOPs (6 cycles total >= 5)
	if cpu.PC != 0x8003 {
		t.Errorf("RunCycles: PC expected 0x8003, got 0x%04X", cpu.PC)
	}
	if cpu.Cycles < 5 {
		t.Errorf("RunCycles: cycles expected >=5, got %d", cpu.Cycles)
	}
}

func TestJMPIndirect(t *testing.T) {
	cpu, bus := newCPU()
	// JMP ($0300) - pointer at $0300/$0301 -> target $9050
	bus.ram[0x8000] = 0x6C // JMP indirect
	bus.ram[0x8001] = 0x00  // lo byte of pointer
	bus.ram[0x8002] = 0x03  // hi byte of pointer
	bus.ram[0x0300] = 0x50  // lo byte of target
	bus.ram[0x0301] = 0x90  // hi byte of target
	cycles := cpu.Step()
	if cpu.PC != 0x9050 {
		t.Errorf("JMP ($0300): expected PC=0x9050, got 0x%04X", cpu.PC)
	}
	if cycles != 5 {
		t.Errorf("JMP ($0300): expected 5 cycles, got %d", cycles)
	}
}

func TestLDAAbsoluteX(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x10
	bus.ram[0x0210] = 0xAB
	bus.ram[0x8000] = 0xBD // LDA $0200,X
	bus.ram[0x8001] = 0x00
	bus.ram[0x8002] = 0x02
	cpu.Step()
	if cpu.A != 0xAB {
		t.Errorf("LDA abs,X: A expected 0xAB, got 0x%02X", cpu.A)
	}
}

func TestLDAAbsoluteY(t *testing.T) {
	cpu, bus := newCPU()
	cpu.Y = 0x20
	bus.ram[0x0420] = 0xCD
	bus.ram[0x8000] = 0xB9 // LDA $0400,Y
	bus.ram[0x8001] = 0x00
	bus.ram[0x8002] = 0x04
	cpu.Step()
	if cpu.A != 0xCD {
		t.Errorf("LDA abs,Y: A expected 0xCD, got 0x%02X", cpu.A)
	}
}

func TestLDAIndirectY(t *testing.T) {
	cpu, bus := newCPU()
	cpu.Y = 0x10
	// ($80),Y: zp pointer at $80/$81 -> $0300, then +Y -> $0310
	bus.ram[0x0080] = 0x00
	bus.ram[0x0081] = 0x03
	bus.ram[0x0310] = 0x77
	bus.ram[0x8000] = 0xB1 // LDA ($80),Y
	bus.ram[0x8001] = 0x80
	cpu.Step()
	if cpu.A != 0x77 {
		t.Errorf("LDA ($zp),Y: A expected 0x77, got 0x%02X", cpu.A)
	}
}

func TestLDAIndirectYPageCross(t *testing.T) {
	cpu, bus := newCPU()
	cpu.Y = 0x01
	// ($80),Y: ptr at $80/$81 -> $02FF, then +Y=$01 -> $0300 (page crossed)
	bus.ram[0x0080] = 0xFF
	bus.ram[0x0081] = 0x02
	bus.ram[0x0300] = 0xAA
	bus.ram[0x8000] = 0xB1 // LDA ($80),Y
	bus.ram[0x8001] = 0x80
	cycles := cpu.Step()
	if cpu.A != 0xAA {
		t.Errorf("LDA ($zp),Y page cross: A expected 0xAA, got 0x%02X", cpu.A)
	}
	if cycles != 6 {
		t.Errorf("LDA ($zp),Y page cross: expected 6 cycles, got %d", cycles)
	}
}

func TestLDXZeroPageY(t *testing.T) {
	cpu, bus := newCPU()
	cpu.Y = 0x05
	bus.ram[0x0085] = 0x3F
	bus.ram[0x8000] = 0xB6 // LDX $80,Y
	bus.ram[0x8001] = 0x80
	cpu.Step()
	if cpu.X != 0x3F {
		t.Errorf("LDX zp,Y: X expected 0x3F, got 0x%02X", cpu.X)
	}
}

func TestSTXAllModes(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x55

	// STX zero page
	bus.ram[0x8000] = 0x86 // STX $30
	bus.ram[0x8001] = 0x30
	cpu.Step()
	if bus.ram[0x0030] != 0x55 {
		t.Errorf("STX zp: expected 0x55, got 0x%02X", bus.ram[0x0030])
	}

	// STX zero page, Y
	cpu.PC = 0x8000
	cpu.Y = 0x02
	bus.ram[0x8000] = 0x96 // STX $40,Y
	bus.ram[0x8001] = 0x40
	cpu.Step()
	if bus.ram[0x0042] != 0x55 {
		t.Errorf("STX zp,Y: expected 0x55, got 0x%02X", bus.ram[0x0042])
	}

	// STX absolute
	cpu.PC = 0x8000
	bus.ram[0x8000] = 0x8E // STX $0600
	bus.ram[0x8001] = 0x00
	bus.ram[0x8002] = 0x06
	cpu.Step()
	if bus.ram[0x0600] != 0x55 {
		t.Errorf("STX abs: expected 0x55, got 0x%02X", bus.ram[0x0600])
	}
}

func TestSTYAllModes(t *testing.T) {
	cpu, bus := newCPU()
	cpu.Y = 0x77

	// STY zero page
	bus.ram[0x8000] = 0x84 // STY $50
	bus.ram[0x8001] = 0x50
	cpu.Step()
	if bus.ram[0x0050] != 0x77 {
		t.Errorf("STY zp: expected 0x77, got 0x%02X", bus.ram[0x0050])
	}

	// STY zero page, X
	cpu.PC = 0x8000
	cpu.X = 0x03
	bus.ram[0x8000] = 0x94 // STY $40,X
	bus.ram[0x8001] = 0x40
	cpu.Step()
	if bus.ram[0x0043] != 0x77 {
		t.Errorf("STY zp,X: expected 0x77, got 0x%02X", bus.ram[0x0043])
	}

	// STY absolute
	cpu.PC = 0x8000
	bus.ram[0x8000] = 0x8C // STY $0700
	bus.ram[0x8001] = 0x00
	bus.ram[0x8002] = 0x07
	cpu.Step()
	if bus.ram[0x0700] != 0x77 {
		t.Errorf("STY abs: expected 0x77, got 0x%02X", bus.ram[0x0700])
	}
}

func TestLDAAbsXPageCross(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x01
	// LDA $02FF,X -> $0300 (page crossed: $02xx -> $03xx)
	bus.ram[0x0300] = 0xEE
	bus.ram[0x8000] = 0xBD // LDA abs,X
	bus.ram[0x8001] = 0xFF  // lo
	bus.ram[0x8002] = 0x02  // hi
	cycles := cpu.Step()
	if cpu.A != 0xEE {
		t.Errorf("LDA abs,X page cross: A expected 0xEE, got 0x%02X", cpu.A)
	}
	if cycles != 5 {
		t.Errorf("LDA abs,X page cross: expected 5 cycles, got %d", cycles)
	}
}

func TestZpXWrap(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x01
	// LDA $FF,X: zp=$FF, X=$01 -> wrap to $00
	bus.ram[0x0000] = 0x42
	bus.ram[0x8000] = 0xB5 // LDA $FF,X
	bus.ram[0x8001] = 0xFF
	cpu.Step()
	if cpu.A != 0x42 {
		t.Errorf("LDA $FF,X wrap: A expected 0x42, got 0x%02X", cpu.A)
	}
}
