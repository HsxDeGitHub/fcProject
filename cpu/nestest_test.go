package cpu

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"fcProject/cartridge"
)

// nestestBus implements a minimal bus for the nestest ROM.
// Layout:
//
//	$0000-$07FF: 2KB RAM (mirrored through $0000-$1FFF)
//	$2000-$401F: PPU/APU registers (ignored for CPU test)
//	$4020-$FFFF: PRG ROM
type nestestBus struct {
	cart *cartridge.Cartridge
	ram  [0x800]uint8
}

func (b *nestestBus) Read(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return b.ram[addr&0x07FF]
	case addr >= 0x4020:
		return b.cart.PRGRead(addr)
	default:
		return 0
	}
}

func (b *nestestBus) Write(addr uint16, data uint8) {
	switch {
	case addr < 0x2000:
		b.ram[addr&0x07FF] = data
	}
}

// nestestLogLine holds the expected CPU state for one instruction.
type nestestLogLine struct {
	PC uint16
	A  uint8
	X  uint8
	Y  uint8
	P  uint8
	SP uint8
}

// parseNestestLog parses the nestest.log file and returns the expected CPU
// state before each instruction. The log format is:
//
//	C000  4C F5 C5  JMP $C5F5                       A:00 X:00 Y:00 P:24 SP:FD PPU:  0, 21 CYC:7
func parseNestestLog(path string) ([]nestestLogLine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}

	// Match: PC (4 hex) ... A:XX X:XX Y:XX P:XX SP:XX
	re := regexp.MustCompile(`^([0-9A-F]{4})\s+.*A:([0-9A-F]{2})\s+X:([0-9A-F]{2})\s+Y:([0-9A-F]{2})\s+P:([0-9A-F]{2})\s+SP:([0-9A-F]{2})`)
	var lines []nestestLogLine

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		matches := re.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		l := nestestLogLine{}
		fmt.Sscanf(matches[1], "%X", &l.PC)
		fmt.Sscanf(matches[2], "%X", &l.A)
		fmt.Sscanf(matches[3], "%X", &l.X)
		fmt.Sscanf(matches[4], "%X", &l.Y)
		fmt.Sscanf(matches[5], "%X", &l.P)
		fmt.Sscanf(matches[6], "%X", &l.SP)
		lines = append(lines, l)
	}
	return lines, nil
}

func TestNestest(t *testing.T) {
	logPath := "testdata/nestest.log"
	romPath := "testdata/nestest.nes"

	// --- Parse expected states ---
	expected, err := parseNestestLog(logPath)
	if err != nil {
		t.Fatalf("Failed to parse nestest log: %v", err)
	}
	if len(expected) == 0 {
		t.Fatalf("No log lines parsed from %s", logPath)
	}
	t.Logf("Parsed %d expected states from nestest.log", len(expected))

	// --- Load ROM ---
	romData, err := os.ReadFile(romPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v\nTo obtain nestest.nes, run:\n  curl -L -o cpu/testdata/nestest.nes 'https://cdn.jsdelivr.net/gh/christopherpow/nes-test-roms@master/other/nestest.nes'", err)
	}

	cart, err := cartridge.Load(romData)
	if err != nil {
		t.Fatalf("Failed to load cartridge: %v", err)
	}

	// --- Set up CPU ---
	bus := &nestestBus{cart: cart}
	cpu := New(bus)

	// Initial state matching first log line:
	//   C000  JMP $C5F5  A:00 X:00 Y:00 P:24 SP:FD
	cpu.PC = expected[0].PC
	cpu.A = expected[0].A
	cpu.X = expected[0].X
	cpu.Y = expected[0].Y
	cpu.P = expected[0].P
	cpu.SP = expected[0].SP

	// --- Execute and compare ---
	mismatches := 0
	maxMismatches := 20 // stop printing after this many
	completed := 0

	for i, exp := range expected {
		// Compare CPU state BEFORE executing this instruction
		if cpu.PC != exp.PC || cpu.A != exp.A || cpu.X != exp.X ||
			cpu.Y != exp.Y || cpu.P != exp.P || cpu.SP != exp.SP {
			mismatches++
			if mismatches <= maxMismatches {
				t.Errorf("Line %d mismatch:\n"+
					"  Expected: PC=%04X A=%02X X=%02X Y=%02X P=%02X SP=%02X\n"+
					"  Got:      PC=%04X A=%02X X=%02X Y=%02X P=%02X SP=%02X",
					i+1, exp.PC, exp.A, exp.X, exp.Y, exp.P, exp.SP,
					cpu.PC, cpu.A, cpu.X, cpu.Y, cpu.P, cpu.SP)
			}
		}

		// Execute one instruction
		cpu.Step()
		completed++

		// After the last log line, nestest enters an infinite loop.
		// Detect that and stop early to avoid hanging.
		if i == len(expected)-1 {
			break
		}
	}

	if mismatches > 0 {
		t.Errorf("Total mismatches: %d out of %d instructions checked (of %d total)",
			mismatches, completed, len(expected))
	} else {
		t.Logf("All %d instructions passed!", completed)
	}
}
