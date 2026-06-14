// cartridge/cartridge_test.go
package cartridge

import (
	"testing"
)

func TestLoadINES(t *testing.T) {
	// 构造最小 iNES ROM（1 PRG bank, 1 CHR bank, Mapper 0）
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1  // 1 x 16KB PRG ROM
	rom[5] = 1  // 1 x 8KB CHR ROM
	rom[6] = 0  // Mapper 0, horizontal mirroring

	cart, err := Load(rom)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cart.PRGBanks != 1 {
		t.Errorf("expected 1 PRG bank, got %d", cart.PRGBanks)
	}
	if cart.CHRBanks != 1 {
		t.Errorf("expected 1 CHR bank, got %d", cart.CHRBanks)
	}
	if cart.Mapper != 0 {
		t.Errorf("expected Mapper 0, got %d", cart.Mapper)
	}
}

func TestLoadINESBadMagic(t *testing.T) {
	rom := make([]byte, 16)
	rom[0], rom[1], rom[2], rom[3] = 'B', 'A', 'D', 0x00
	_, err := Load(rom)
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestMapper0PRGRead(t *testing.T) {
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1
	rom[6] = 0
	rom[16+0x100] = 0x42

	cart, _ := Load(rom)
	val := cart.PRGRead(0x8000 + 0x100)
	if val != 0x42 {
		t.Errorf("expected 0x42, got 0x%02X", val)
	}
}

func TestMapper016KBMirror(t *testing.T) {
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1
	rom[6] = 0
	rom[16+0x100] = 0x42

	cart, _ := Load(rom)
	val := cart.PRGRead(0xC000 + 0x100)
	if val != 0x42 {
		t.Errorf("16KB mirror: expected 0x42, got 0x%02X", val)
	}
}

func TestMapper0CHRRead(t *testing.T) {
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1
	rom[6] = 0
	prgEnd := 16 + 16384
	rom[prgEnd+0x200] = 0x99

	cart, _ := Load(rom)
	val := cart.CHRRead(0x0200)
	if val != 0x99 {
		t.Errorf("expected 0x99, got 0x%02X", val)
	}
}
