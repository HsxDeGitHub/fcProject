package apu

import (
	"testing"
)

func TestSquareChannelFrequency(t *testing.T) {
	ch := newSquareChannel()
	ch.write(2, 0xFF)
	ch.write(3, 0x07)
	expected := uint16(0x7FF)
	if ch.timerPeriod != expected {
		t.Errorf("timer period: expected 0x%04X, got 0x%04X", expected, ch.timerPeriod)
	}
}

func TestSquareChannelDuty(t *testing.T) {
	ch := newSquareChannel()
	ch.write(0, 0xC0) // duty = 3
	if ch.duty != 3 {
		t.Errorf("duty: expected 3, got %d", ch.duty)
	}
}

func TestSquareChannelOutput(t *testing.T) {
	ch := newSquareChannel()
	ch.write(0, 0x30) // duty=0, loop=0, const=1, volume=0
	ch.write(2, 0x08) // low byte of timer period = 8
	ch.write(3, 0x00) // high byte = 0, period = 8
	ch.enabled = true
	ch.write(0, 0x1F) // duty=0, const volume, volume=15
	var outputs []float32
	for i := 0; i < 20; i++ {
		outputs = append(outputs, ch.sample())
	}
	hasNonZero := false
	for _, v := range outputs {
		if v != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("expected non-zero output from square channel")
	}
}

func TestAPUEnableChannel(t *testing.T) {
	apu := New()
	apu.WriteRegister(0x4015, 0x01) // enable square 1
	if !apu.sq1.enabled {
		t.Error("square 1 should be enabled")
	}
}

func TestAPUDisableChannel(t *testing.T) {
	apu := New()
	apu.WriteRegister(0x4015, 0x01)
	apu.WriteRegister(0x4015, 0x00)
	if apu.sq1.enabled {
		t.Error("square 1 should be disabled")
	}
}

func TestAPUInitialSilence(t *testing.T) {
	apu := New()
	samples := apu.GenerateSamples()
	for _, s := range samples {
		if s != 0 {
			t.Error("expected silence when channels disabled")
			break
		}
	}
}

func TestTriangleOutput(t *testing.T) {
	ch := newTriangleChannel()
	ch.enabled = true
	ch.write(2, 0x02) // low byte of timer period = 2
	ch.write(3, 0x00) // high byte = 0, period = 2
	var outputs []float32
	for i := 0; i < 15; i++ {
		outputs = append(outputs, ch.sample())
	}
	// Triangle should produce varying values
	hasVariation := false
	first := outputs[0]
	for _, v := range outputs[1:] {
		if v != first {
			hasVariation = true
			break
		}
	}
	if !hasVariation {
		t.Error("triangle should produce varying output")
	}
}

func TestNoiseOutput(t *testing.T) {
	ch := newNoiseChannel()
	ch.enabled = true
	ch.write(0, 0x1F) // const volume, volume=15
	ch.write(2, 0x00) // period index 0
	var outputs []float32
	for i := 0; i < 15; i++ {
		outputs = append(outputs, ch.sample())
	}
	hasNonZero := false
	for _, v := range outputs {
		if v != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("noise should produce output")
	}
}
