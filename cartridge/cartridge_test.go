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
	rom[6] = 0  // Mapper 0 lower nibble, horizontal mirroring
	rom[7] = 0  // Mapper 0 upper nibble

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

func TestLoadROMTooSmall(t *testing.T) {
	rom := make([]byte, 15)
	_, err := Load(rom)
	if err == nil {
		t.Fatal("expected error for ROM too small")
	}
}

func TestLoadROMTruncated(t *testing.T) {
	// Header says 2 PRG banks + 1 CHR bank, but data is too short
	rom := make([]byte, 16+16384) // only 1 PRG bank of data
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 2 // claims 2 PRG banks
	rom[5] = 1
	rom[6] = 0
	rom[7] = 0
	_, err := Load(rom)
	if err == nil {
		t.Fatal("expected error for truncated ROM")
	}
}

func TestLoadUnsupportedMapper(t *testing.T) {
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1
	rom[6] = 0x10 // Mapper 1 (lower nibble)
	rom[7] = 0x00
	_, err := Load(rom)
	if err == nil {
		t.Fatal("expected error for unsupported mapper")
	}
}

func TestMapper0PRGRead(t *testing.T) {
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1
	rom[6] = 0
	rom[7] = 0
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
	rom[7] = 0
	rom[16+0x100] = 0x42

	cart, _ := Load(rom)
	val := cart.PRGRead(0xC000 + 0x100)
	if val != 0x42 {
		t.Errorf("16KB mirror: expected 0x42, got 0x%02X", val)
	}
}

func TestMapper032KBPRGNoMirror(t *testing.T) {
	// 2 PRG banks => 32KB, no mirroring
	rom := make([]byte, 16+32768+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 2 // 2 x 16KB PRG ROM
	rom[5] = 1
	rom[6] = 0
	rom[7] = 0

	// Write different values to first and second PRG banks
	rom[16+0x100] = 0x11       // offset 0x0100 in first 16KB bank
	rom[16+16384+0x100] = 0x22 // offset 0x0100 in second 16KB bank

	cart, _ := Load(rom)

	// First bank read: 0x8000 + 0x100
	if v := cart.PRGRead(0x8000 + 0x100); v != 0x11 {
		t.Errorf("first bank: expected 0x11, got 0x%02X", v)
	}
	// Second bank read: 0xC000 + 0x100
	if v := cart.PRGRead(0xC000 + 0x100); v != 0x22 {
		t.Errorf("second bank: expected 0x22, got 0x%02X", v)
	}
}

func TestMapper0CHRRead(t *testing.T) {
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1
	rom[6] = 0
	rom[7] = 0
	prgEnd := 16 + 16384
	rom[prgEnd+0x200] = 0x99

	cart, _ := Load(rom)
	val := cart.CHRRead(0x0200)
	if val != 0x99 {
		t.Errorf("expected 0x99, got 0x%02X", val)
	}
}

func TestCHRRAMAllocation(t *testing.T) {
	// chrBanks == 0: should allocate 8KB CHR RAM
	rom := make([]byte, 16+16384)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 0 // no CHR ROM
	rom[6] = 0
	rom[7] = 0

	cart, err := Load(rom)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cart.CHR) != 8192 {
		t.Errorf("expected 8192 bytes CHR RAM, got %d", len(cart.CHR))
	}
	if cart.CHRBanks != 0 {
		t.Errorf("expected 0 CHR banks, got %d", cart.CHRBanks)
	}

	// CHR RAM should be writable
	cart.CHRWrite(0x0000, 0xAB)
	if v := cart.CHRRead(0x0000); v != 0xAB {
		t.Errorf("CHR RAM: expected 0xAB after write, got 0x%02X", v)
	}
}

func TestCHRROMWriteProtection(t *testing.T) {
	// chrBanks > 0: CHR ROM, writes should be ignored
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1 // CHR ROM present
	rom[6] = 0
	rom[7] = 0
	prgEnd := 16 + 16384
	rom[prgEnd+0x200] = 0x99

	cart, _ := Load(rom)

	// Attempt to overwrite CHR ROM
	cart.CHRWrite(0x0200, 0x77)

	// Should still read original value
	if v := cart.CHRRead(0x0200); v != 0x99 {
		t.Errorf("CHR ROM should be write-protected: expected 0x99, got 0x%02X", v)
	}
}

func TestPRGReadBounds(t *testing.T) {
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1
	rom[6] = 0
	rom[7] = 0

	cart, _ := Load(rom)

	// Address below PRG range should return 0
	if v := cart.PRGRead(0x6000); v != 0 {
		t.Errorf("below PRG range: expected 0, got 0x%02X", v)
	}
}

func TestCHRReadBounds(t *testing.T) {
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1
	rom[6] = 0
	rom[7] = 0

	cart, _ := Load(rom)

	// Address beyond CHR range should return 0
	if v := cart.CHRRead(0x3000); v != 0 {
		t.Errorf("beyond CHR range: expected 0, got 0x%02X", v)
	}
}
