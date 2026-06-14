package cartridge

import "testing"

func TestMapper4BankSelect(t *testing.T) {
	prg := make([]byte, 512*8192)
	chr := make([]byte, 256*8192)
	m := NewMapper4(prg, chr, 1)

	// Select register 6
	m.PRGWrite(0x8000, 0x06)
	m.PRGWrite(0x8001, 0x03)

	if m.registers[6] != 0x03 {
		t.Errorf("expected register 6 = 3, got %d", m.registers[6])
	}
}

func TestMapper4CHRBank(t *testing.T) {
	prg := make([]byte, 512*8192)
	chr := make([]byte, 256*8192)
	chr[5*1024+0x100] = 0x99 // Bank 5, offset 0x100
	m := NewMapper4(prg, chr, 1)

	// Map CHR $1000-$13FF to bank 5
	m.PRGWrite(0x8000, 0x02) // Select register 2
	m.PRGWrite(0x8001, 0x05) // Write bank 5

	if val := m.CHRRead(0x1100); val != 0x99 {
		t.Errorf("expected 0x99 at bank 5 offset 0x100, got 0x%02X", val)
	}
}
