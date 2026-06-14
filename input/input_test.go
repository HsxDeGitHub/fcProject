package input

import (
	"testing"
)

func TestControllerProtocol(t *testing.T) {
	ctrl := New()

	// Manually set button states (simulating what Update() does from keyboard)
	ctrl.buttons[0] = true  // A
	ctrl.buttons[1] = false // B
	ctrl.buttons[2] = false // Select
	ctrl.buttons[3] = true  // Start
	ctrl.buttons[4] = false // Up
	ctrl.buttons[5] = false // Down
	ctrl.buttons[6] = false // Left
	ctrl.buttons[7] = true  // Right

	// Standard NES controller read sequence:
	// 1. Strobe on ($01 → $4016)
	ctrl.Write(1)
	if !ctrl.strobe || ctrl.index != 0 {
		t.Fatal("strobe should be on, index=0")
	}

	// 2. Strobe off ($00 → $4016)
	ctrl.Write(0)
	if ctrl.strobe || ctrl.index != 0 {
		t.Fatal("strobe should be off, index=0")
	}

	// 3. Read 8 times
	expected := []uint8{1, 0, 0, 1, 0, 0, 0, 1} // A, B, Sel, Start, Up, Down, Left, Right
	for i, exp := range expected {
		val := ctrl.Read()
		if val != exp {
			t.Errorf("read %d: expected %d, got %d (index=%d)", i, exp, val, ctrl.index)
		}
	}

	// 4. 9th read should return 0 (index >= 8)
	val := ctrl.Read()
	if val != 0 {
		t.Errorf("9th read: expected 0, got %d", val)
	}
}

func TestControllerStrobeReload(t *testing.T) {
	ctrl := New()
	ctrl.buttons[0] = true // A pressed

	// Strobe on
	ctrl.Write(1)
	if ctrl.index != 0 {
		t.Fatal("index should be 0 after strobe on")
	}

	// Read while strobe is on: should always return A (index reset after each read)
	for i := 0; i < 5; i++ {
		val := ctrl.Read()
		if val != 1 {
			t.Errorf("read %d during strobe: expected 1 (A pressed), got %d", i, val)
		}
	}

	// Strobe off
	ctrl.Write(0)

	// Now reads should advance through all 8 buttons
	read1 := ctrl.Read() // A (button[0]=true)
	if read1 != 1 {
		t.Errorf("after strobe off, read 0: expected 1 (A), got %d", read1)
	}
	read2 := ctrl.Read() // B (button[1]=false)
	if read2 != 0 {
		t.Errorf("after strobe off, read 1: expected 0 (B), got %d", read2)
	}
}

func TestControllerNoButtons(t *testing.T) {
	ctrl := New()

	// No buttons pressed
	ctrl.Write(1)
	ctrl.Write(0)

	for i := 0; i < 8; i++ {
		val := ctrl.Read()
		if val != 0 {
			t.Errorf("read %d: expected 0 (no buttons), got %d", i, val)
		}
	}
}
