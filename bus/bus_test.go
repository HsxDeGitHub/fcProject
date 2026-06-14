package bus

import (
	"testing"
)

type mockCart struct {
	prg []byte
}

func (m *mockCart) PRGRead(addr uint16) uint8  { return m.prg[addr-0x8000] }
func (m *mockCart) PRGWrite(addr uint16, v uint8) {}
func (m *mockCart) CHRRead(addr uint16) uint8     { return 0 }
func (m *mockCart) CHRWrite(addr uint16, v uint8) {}

func TestRAMReadWrite(t *testing.T) {
	b := NewBus(nil, nil, nil, nil)
	b.Write(0x0000, 0x42)
	if val := b.Read(0x0000); val != 0x42 {
		t.Errorf("RAM[0x0000] expected 0x42, got 0x%02X", val)
	}
}

func TestRAMMirror(t *testing.T) {
	b := NewBus(nil, nil, nil, nil)
	b.Write(0x0000, 0x42)
	if val := b.Read(0x0800); val != 0x42 {
		t.Errorf("mirror 0x0800 expected 0x42, got 0x%02X", val)
	}
	if val := b.Read(0x1800); val != 0x42 {
		t.Errorf("mirror 0x1800 expected 0x42, got 0x%02X", val)
	}
}

func TestPRGROMRead(t *testing.T) {
	prg := make([]byte, 16384)
	prg[0x100] = 0x99
	cart := &mockCart{prg: prg}
	b := NewBus(cart, nil, nil, nil)
	val := b.Read(0x8000 + 0x100)
	if val != 0x99 {
		t.Errorf("PRG ROM expected 0x99, got 0x%02X", val)
	}
}

func TestPPURegisterWrite(t *testing.T) {
	ppu := &mockPPU{}
	b := NewBus(nil, ppu, nil, nil)
	b.Write(0x2000, 0x9A)
	if ppu.ctrl != 0x9A {
		t.Errorf("PPUCTRL expected 0x9A, got 0x%02X", ppu.ctrl)
	}
}

type mockPPU struct {
	ctrl byte
}

func (m *mockPPU) ReadRegister(addr uint16) uint8   { return 0 }
func (m *mockPPU) WriteRegister(addr uint16, v uint8) {
	if addr == 0x2000 {
		m.ctrl = v
	}
}
func (m *mockPPU) RenderFrame() []byte { return nil }

func TestMirroredPPURange(t *testing.T) {
	ppu := &mockPPU{}
	b := NewBus(nil, ppu, nil, nil)
	b.Write(0x2008, 0xFF) // $2008 is mirror of $2000
	if ppu.ctrl != 0xFF {
		t.Errorf("PPUCTRL via mirror expected 0xFF, got 0x%02X", ppu.ctrl)
	}
}
