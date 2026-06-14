package cartridge

import "testing"

func TestMapper1ShiftRegister(t *testing.T) {
	prg := make([]byte, 256*16384)
	chr := make([]byte, 128*8192)
	m := NewMapper1(prg, chr, 0)

	// 5 writes to set control = 0x11
	for i := 0; i < 5; i++ {
		m.PRGWrite(0x8000, uint8(0x11>>i)&1)
	}
	if m.control != 0x11 {
		t.Errorf("expected control 0x11, got 0x%02X", m.control)
	}
}

func TestMapper1ResetShift(t *testing.T) {
	prg := make([]byte, 256*16384)
	chr := make([]byte, 128*8192)
	m := NewMapper1(prg, chr, 0)

	m.PRGWrite(0x8000, 0x01)
	m.PRGWrite(0x8000, 0x01)
	m.PRGWrite(0x8000, 0x80) // Reset

	if m.shiftCount != 0 {
		t.Errorf("shift count should be 0 after reset, got %d", m.shiftCount)
	}
}

func TestMapper1PRGBankSwitch(t *testing.T) {
	prg := make([]byte, 256*16384)
	chr := make([]byte, 128*8192)
	// Write data to bank 3
	prg[3*16384+0x100] = 0x42
	m := NewMapper1(prg, chr, 0)

	// Set PRG bank to 3 in 16KB mode
	for i := 0; i < 5; i++ {
		m.PRGWrite(0xE000, uint8(3>>i)&1)
	}

	// Default control=0x0C (bit3=1): $C000-$FFFF is switchable, $8000-$BFFF fixed to first.
	// Read from $C100 should now map to bank 3.
	if val := m.PRGRead(0x8100); val != 0x42 {
		t.Errorf("expected 0x42 from bank 3 at $C100, got 0x%02X", val)
	}
}
