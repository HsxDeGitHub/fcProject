package main

import (
	"fmt"
	"os"
	"testing"

	"fcProject/apu"
	"fcProject/bus"
	"fcProject/cartridge"
	"fcProject/cpu"
	"fcProject/input"
	"fcProject/ppu"
)

const (
	diagCyclesPerFrame = 29781
)

func TestHeadlessDiagnostic(t *testing.T) {
	romPath := "rom/SuperMary.nes"
	data, err := os.ReadFile(romPath)
	if err != nil {
		t.Skipf("Skipping: ROM not found at %s: %v", romPath, err)
	}

	cart, err := cartridge.Load(data)
	if err != nil {
		t.Fatalf("Failed to load cartridge: %v", err)
	}

	inp := input.New()
	p := ppu.New(cart)
	a := apu.New()
	b := bus.NewBus(cart, p, a, inp)
	c := cpu.New(b)
	c.Reset()

	fmt.Printf("=== Headless Diagnostic: %s (Mapper %d, %d PRG, %d CHR) ===\n\n",
		romPath, cart.Mapper, cart.PRGBanks, cart.CHRBanks)

	fmt.Printf("Frame | PC      | Ctrl  | Mask  | Status | PalWrites | NonZeroPx | Pal[0]\n")
	fmt.Printf("------+---------+-------+-------+--------+-----------+-----------+-------\n")

	firstPalWriteFrame := -1
	firstPixelFrame := -1

	for frame := 0; frame < 60; frame++ {
		// Simulate VBlank at frame start
		p.Status |= 0x80
		p.VblankReasserts = 10

		// Trigger NMI if VBlank + NMI enabled
		if p.Ctrl&0x80 != 0 {
			c.NMI = true
		}

		// Run CPU for one frame of cycles
		c.RunCycles(c.Cycles + uint64(diagCyclesPerFrame))

		// Render the frame
		p.RenderFrame()

		// Count non-zero pixels
		nonZeroPx := 0
		for i := 0; i < ppu.ScreenWidth*ppu.ScreenHeight*4; i += 4 {
			if p.Frame[i] != 0 || p.Frame[i+1] != 0 || p.Frame[i+2] != 0 {
				nonZeroPx++
			}
		}

		palWrites := p.PalWrites

		// Track first occurrences
		if firstPalWriteFrame < 0 && palWrites > 0 {
			firstPalWriteFrame = frame + 1
		}
		if firstPixelFrame < 0 && nonZeroPx > 0 {
			firstPixelFrame = frame + 1
		}

		fmt.Printf("%5d | $%04X   | $%02X   | $%02X   | $%02X    | %9d | %9d | $%02X\n",
			frame+1, c.PC, p.Ctrl, p.Mask, p.Status, palWrites, nonZeroPx, p.Palette[0])
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("First palette write: frame %d\n", firstPalWriteFrame)
	fmt.Printf("First visible pixels: frame %d\n", firstPixelFrame)
	fmt.Printf("Total palette writes: %d\n", p.PalWrites)

	// Final assertions
	if p.PalWrites == 0 {
		t.Error("CRITICAL: No palette writes occurred in 60 frames. The game is not writing to the palette.")
	} else {
		t.Logf("SUCCESS: %d total palette writes across 60 frames. First palette write at frame %d.",
			p.PalWrites, firstPalWriteFrame)
	}

	// Check that at least some non-zero pixels appear
	hasPixels := false
	for i := 0; i < ppu.ScreenWidth*ppu.ScreenHeight*4; i += 4 {
		if p.Frame[i] != 0 || p.Frame[i+1] != 0 || p.Frame[i+2] != 0 {
			hasPixels = true
			break
		}
	}
	if !hasPixels {
		t.Error("CRITICAL: No visible pixels rendered after 60 frames.")
	} else {
		t.Logf("SUCCESS: Visible pixels are being rendered. First visible frame: %d.", firstPixelFrame)
	}

	// Check palette writes happen within reasonable timeframe
	if firstPalWriteFrame > 15 {
		t.Errorf("Palette writes started too late (frame %d, expected by frame 12)", firstPalWriteFrame)
	}
}
