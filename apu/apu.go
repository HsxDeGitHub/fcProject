package apu

const SampleRate = 44100

var dutyTable = [4][8]float32{
	{0, 1, 0, 0, 0, 0, 0, 0},
	{0, 1, 1, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 1, 0, 0, 0},
	{1, 0, 0, 1, 1, 1, 1, 1},
}

type squareChannel struct {
	enabled         bool
	duty            uint8
	dutyIndex       uint8
	volume          uint8
	constVolume     bool
	timerPeriod     uint16
	timer           uint16
	envelopeLoop    bool
	envelopeVolume  uint8
	envelopeDivider uint8
	envelopeStart   bool
}

func newSquareChannel() *squareChannel {
	return &squareChannel{}
}

func (ch *squareChannel) write(reg uint8, data uint8) {
	switch reg {
	case 0:
		ch.duty = (data >> 6) & 0x03
		ch.envelopeLoop = (data & 0x20) != 0
		ch.constVolume = (data & 0x10) != 0
		ch.volume = data & 0x0F
		ch.envelopeDivider = ch.volume
	case 2:
		ch.timerPeriod = (ch.timerPeriod & 0x0700) | uint16(data)
	case 3:
		ch.timerPeriod = (ch.timerPeriod & 0x00FF) | (uint16(data&0x07) << 8)
		ch.timer = ch.timerPeriod
		ch.dutyIndex = 0
		ch.envelopeStart = true
	}
}

func (ch *squareChannel) sample() float32 {
	if !ch.enabled || ch.timerPeriod < 8 {
		return 0
	}
	ch.timer--
	if ch.timer == 0 {
		ch.timer = ch.timerPeriod + 1
		ch.dutyIndex = (ch.dutyIndex + 1) & 7
	}
	var vol uint8
	if ch.constVolume {
		vol = ch.volume
	} else {
		vol = ch.envelopeVolume
	}
	return dutyTable[ch.duty][ch.dutyIndex] * float32(vol) / 15.0
}

type triangleChannel struct {
	enabled       bool
	timerPeriod   uint16
	timer         uint16
	step          uint8
	linearCounter uint8
	lengthCounter uint8
}

func newTriangleChannel() *triangleChannel {
	return &triangleChannel{}
}

var triangleWave = [32]float32{
	15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
}

func (ch *triangleChannel) sample() float32 {
	if !ch.enabled || ch.timerPeriod < 2 {
		return 0
	}
	ch.timer--
	if ch.timer == 0 {
		ch.timer = ch.timerPeriod + 1
		ch.step = (ch.step + 1) & 31
	}
	return triangleWave[ch.step]/15.0*2 - 1.0
}

func (ch *triangleChannel) write(reg uint8, data uint8) {
	switch reg {
	case 2:
		ch.timerPeriod = (ch.timerPeriod & 0x0700) | uint16(data)
	case 3:
		ch.timerPeriod = (ch.timerPeriod & 0x00FF) | (uint16(data&0x07) << 8)
		ch.timer = ch.timerPeriod
	}
}

type noiseChannel struct {
	enabled        bool
	timerPeriod    uint16
	timer          uint16
	shiftReg       uint16
	volume         uint8
	constVolume    bool
	envelopeVolume uint8
	envelopeLoop   bool
}

func newNoiseChannel() *noiseChannel {
	return &noiseChannel{shiftReg: 1}
}

var noisePeriodTable = [16]uint16{
	4, 8, 16, 32, 64, 96, 128, 160, 202, 254, 380, 508, 762, 1016, 2034, 4068,
}

func (ch *noiseChannel) sample() float32 {
	if !ch.enabled {
		return 0
	}
	ch.timer--
	if ch.timer == 0 {
		ch.timer = ch.timerPeriod
		feedback := (ch.shiftReg & 1) ^ ((ch.shiftReg >> 6) & 1)
		ch.shiftReg = (ch.shiftReg >> 1) | (feedback << 14)
	}
	if ch.shiftReg&1 != 0 {
		var vol uint8
		if ch.constVolume {
			vol = ch.volume
		} else {
			vol = ch.envelopeVolume
		}
		return float32(vol) / 15.0
	}
	return 0
}

func (ch *noiseChannel) write(reg uint8, data uint8) {
	switch reg {
	case 0:
		ch.constVolume = data&0x10 != 0
		ch.volume = data & 0x0F
		ch.envelopeLoop = data&0x20 != 0
	case 2:
		ch.timerPeriod = noisePeriodTable[data&0x0F]
	}
}

type APU struct {
	sq1   *squareChannel
	sq2   *squareChannel
	tri   *triangleChannel
	noise *noiseChannel
}

func New() *APU {
	return &APU{
		sq1:   newSquareChannel(),
		sq2:   newSquareChannel(),
		tri:   newTriangleChannel(),
		noise: newNoiseChannel(),
	}
}

func (a *APU) WriteRegister(addr uint16, data uint8) {
	switch {
	case addr >= 0x4000 && addr <= 0x4003:
		a.sq1.write(uint8(addr-0x4000), data)
	case addr >= 0x4004 && addr <= 0x4007:
		a.sq2.write(uint8(addr-0x4004), data)
	case addr >= 0x4008 && addr <= 0x400B:
		a.tri.write(uint8(addr-0x4008), data)
	case addr >= 0x400C && addr <= 0x400F:
		a.noise.write(uint8(addr-0x400C), data)
	case addr == 0x4015:
		a.sq1.enabled = data&0x01 != 0
		a.sq2.enabled = data&0x02 != 0
		a.tri.enabled = data&0x04 != 0
		a.noise.enabled = data&0x08 != 0
	}
}

func (a *APU) ReadStatus() uint8 {
	var status uint8
	if a.sq1.enabled {
		status |= 0x01
	}
	if a.sq2.enabled {
		status |= 0x02
	}
	if a.tri.enabled {
		status |= 0x04
	}
	if a.noise.enabled {
		status |= 0x08
	}
	return status
}

func (a *APU) GenerateSamples() []float32 {
	samplesPerFrame := SampleRate / 60
	samples := make([]float32, samplesPerFrame)
	for i := range samples {
		s := a.sq1.sample() + a.sq2.sample() + a.tri.sample() + a.noise.sample()
		samples[i] = s / 4.0
	}
	return samples
}
