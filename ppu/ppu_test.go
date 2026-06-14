package ppu

import (
	"testing"
)

type mockCart struct {
	chr []byte
}

func (m *mockCart) CHRRead(addr uint16) uint8   { return m.chr[addr] }
func (m *mockCart) CHRWrite(addr uint16, v uint8) { m.chr[addr] = v }
func (m *mockCart) MirrorMode() uint8            { return 0 }

func newPPU() *PPU {
	cart := &mockCart{chr: make([]byte, 8192)}
	return New(cart)
}

func TestPPUCTRLWrite(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2000, 0x9A)
	if ppu.Ctrl != 0x9A {
		t.Errorf("PPUCTRL expected 0x9A, got 0x%02X", ppu.Ctrl)
	}
}

func TestPPUMASKWrite(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2001, 0x1E)
	if ppu.Mask != 0x1E {
		t.Errorf("PPUMASK expected 0x1E, got 0x%02X", ppu.Mask)
	}
}

func TestPPUSTATUSReadClearsVBlank(t *testing.T) {
	ppu := newPPU()
	ppu.Status = 0x80
	val := ppu.ReadRegister(0x2002)
	if val&0x80 == 0 {
		t.Error("PPUSTATUS: VBlank bit should be set")
	}
	if ppu.Status&0x80 != 0 {
		t.Error("PPUSTATUS: VBlank should be cleared after read")
	}
}

func TestPPUSTATUSClearsLatch(t *testing.T) {
	ppu := newPPU()
	ppu.Latch = true
	ppu.ReadRegister(0x2002)
	if ppu.Latch {
		t.Error("PPUSTATUS read should clear latch")
	}
}

func TestPPUSCROLLWrite(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2005, 0x42) // first write: X scroll
	ppu.WriteRegister(0x2005, 0x10) // second write: Y scroll
	if ppu.ScrollX != 0x42 {
		t.Errorf("ScrollX expected 0x42, got 0x%02X", ppu.ScrollX)
	}
	if ppu.ScrollY != 0x10 {
		t.Errorf("ScrollY expected 0x10, got 0x%02X", ppu.ScrollY)
	}
}

func TestPPUADDRWrite(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2006, 0x23) // high byte
	ppu.WriteRegister(0x2006, 0x45) // low byte
	if ppu.V != 0x2345 {
		t.Errorf("PPUADDR expected 0x2345, got 0x%04X", ppu.V)
	}
}

func TestPPUDATAReadIncrementBy1(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2000, 0x00) // increment by 1 (bit 2 = 0)
	ppu.WriteRegister(0x2006, 0x20)
	ppu.WriteRegister(0x2006, 0x00)
	ppu.ReadRegister(0x2007) // dummy read
	ppu.ReadRegister(0x2007) // real read
	if ppu.V != 0x2002 {
		t.Errorf("V expected 0x2002 after read+buf, got 0x%04X", ppu.V)
	}
}

func TestPPUDATAReadIncrementBy32(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2000, 0x04) // increment by 32
	ppu.WriteRegister(0x2006, 0x20)
	ppu.WriteRegister(0x2006, 0x00)
	ppu.ReadRegister(0x2007) // dummy read
	ppu.ReadRegister(0x2007) // real read
	if ppu.V != 0x2040 {
		t.Errorf("V expected 0x2040, got 0x%04X", ppu.V)
	}
}

func TestVRAMWriteAndReadBack(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2006, 0x20)
	ppu.WriteRegister(0x2006, 0x00)
	ppu.WriteRegister(0x2007, 0x99)
	// reset address
	ppu.WriteRegister(0x2006, 0x20)
	ppu.WriteRegister(0x2006, 0x00)
	ppu.ReadRegister(0x2007) // dummy
	val := ppu.ReadRegister(0x2007)
	if val != 0x99 {
		t.Errorf("VRAM: expected 0x99, got 0x%02X", val)
	}
}

func TestOAMWrite(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2003, 0x00) // OAMADDR = 0
	ppu.WriteRegister(0x2004, 0x42) // OAMDATA
	if ppu.OAM[0] != 0x42 {
		t.Errorf("OAM[0] expected 0x42, got 0x%02X", ppu.OAM[0])
	}
	if ppu.OAMAddr != 1 {
		t.Errorf("OAMAddr should auto-increment to 1, got %d", ppu.OAMAddr)
	}
}

func TestSharedLatchBehavior(t *testing.T) {
	ppu := newPPU()

	// Write $2005 (first write) → Latch should become true
	ppu.WriteRegister(0x2005, 0x42)
	if !ppu.Latch {
		t.Error("after first $2005 write, Latch should be true")
	}

	// Write $2006 → should be second write (Latch was true from $2005)
	// This writes the LOW byte and copies T to V
	ppu.WriteRegister(0x2006, 0x45)
	if ppu.Latch {
		t.Error("after $2006 write, Latch should toggle to false")
	}
	// T should have been set to 0x0045 (low byte only, high byte is 0 since we never wrote it)
	// V should equal T after low byte write
	if ppu.V != ppu.T {
		t.Errorf("after $2006 low byte, V (0x%04X) should equal T (0x%04X)", ppu.V, ppu.T)
	}

	// Read $2002 → Latch should be reset
	ppu.ReadRegister(0x2002)
	if ppu.Latch {
		t.Error("after $2002 read, Latch should be false")
	}

	// Now write $2006 → should be first write again (high byte)
	ppu.WriteRegister(0x2006, 0x23)
	if !ppu.Latch {
		t.Error("after first $2006 write (post-reset), Latch should be true")
	}
	ppu.WriteRegister(0x2006, 0x45) // second write: low byte, copy T to V
	if ppu.V != 0x2345 {
		t.Errorf("V expected 0x2345 after full $2006 sequence, got 0x%04X", ppu.V)
	}
}

func TestPaletteMirroring(t *testing.T) {
	ppu := newPPU()
	// Write to $3F10 (mirror of $3F00)
	ppu.WriteRegister(0x2006, 0x3F)
	ppu.WriteRegister(0x2006, 0x10)
	ppu.WriteRegister(0x2007, 0x42)
	// Read from $3F00 - palette reads return immediately, so the first read after
	// setting PPUADDR works without the usual dummy-buffering delay.
	ppu.WriteRegister(0x2006, 0x3F)
	ppu.WriteRegister(0x2006, 0x00)
	val := ppu.ReadRegister(0x2007)
	if val != 0x42 {
		t.Errorf("Palette mirror: expected 0x42, got 0x%02X", val)
	}
}
