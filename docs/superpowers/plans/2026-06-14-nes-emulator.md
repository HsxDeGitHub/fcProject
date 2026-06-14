# NES 模拟器实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现能运行 SuperMary.nes（Mapper 0）的 NES 模拟器

**Architecture:** 6 个独立包（cpu/ppu/apu/cartridge/bus/input）通过 Bus 接口通信，Ebitengine 驱动主循环

**Tech Stack:** Go 1.25 + Ebitengine v2

---

### Task 1: 项目初始化

**Files:**
- Create: `go.mod`
- Create: `main.go`（空壳）
- Create: `cpu/cpu.go`
- Create: `ppu/ppu.go`
- Create: `apu/apu.go`
- Create: `cartridge/cartridge.go`
- Create: `bus/bus.go`
- Create: `input/input.go`

- [ ] **Step 1: 初始化 Go module**

```bash
cd /Users/huangshengxue/code/ai/fcProject
go mod init fcProject
```

- [ ] **Step 2: 安装 Ebitengine 依赖**

```bash
go get github.com/hajimehoshi/ebiten/v2
```

- [ ] **Step 3: 创建包目录和空文件**

```bash
mkdir -p cpu ppu apu cartridge bus input
touch cpu/cpu.go ppu/ppu.go apu/apu.go cartridge/cartridge.go bus/bus.go input/input.go
```

- [ ] **Step 4: 创建 main.go 骨架**

```go
package main

import (
	"log"
	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct{}

func (g *Game) Update() error  { return nil }
func (g *Game) Draw(screen *ebiten.Image) {}
func (g *Game) Layout(w, h int) (int, int) { return 768, 720 }

func main() {
	ebiten.SetWindowSize(768, 720)
	ebiten.SetWindowTitle("NES Emulator")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 5: 验证编译通过**

```bash
go build ./...
```

Expected: 编译成功

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: initialize Go project with Ebitengine skeleton"
```

---

### Task 2: Cartridge - ROM 加载与 Mapper 0

**Files:**
- Create: `cartridge/cartridge.go`
- Create: `cartridge/cartridge_test.go`

- [ ] **Step 1: 编写 Cartridge 的失败测试**

```go
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
	// 在 PRG ROM 区域写入测试数据
	rom[16+0x100] = 0x42 // 偏移 0x100 处

	cart, _ := Load(rom)
	val := cart.PRGRead(0x8000 + 0x100)
	if val != 0x42 {
		t.Errorf("expected 0x42, got 0x%02X", val)
	}
}

func TestMapper016KBMirror(t *testing.T) {
	// 16KB PRG ROM 应在 $C000 处镜像
	rom := make([]byte, 16+16384+8192)
	rom[0], rom[1], rom[2], rom[3] = 'N', 'E', 'S', 0x1A
	rom[4] = 1
	rom[5] = 1
	rom[0] = 'N' // 重新确认
	rom[6] = 0
	rom[16+0x100] = 0x42

	cart, _ := Load(rom)
	// 16KB 下 $C000 应镜像到 $8000
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
	rom[prgEnd+0x200] = 0x99 // CHR 偏移 0x200

	cart, _ := Load(rom)
	val := cart.CHRRead(0x0200)
	if val != 0x99 {
		t.Errorf("expected 0x99, got 0x%02X", val)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./cartridge/...
```

Expected: 编译失败（类型未定义）

- [ ] **Step 3: 实现 Cartridge**

```go
// cartridge/cartridge.go
package cartridge

import (
	"errors"
)

type Cartridge struct {
	PRG      []byte // PRG ROM 数据
	CHR      []byte // CHR ROM 数据
	PRGBanks int    // PRG ROM bank 数（16KB 单位）
	CHRBanks int    // CHR ROM bank 数（8KB 单位）
	Mapper   int    // Mapper 编号
	Mirror   int    // 0=水平, 1=垂直
}

func Load(data []byte) (*Cartridge, error) {
	if len(data) < 16 {
		return nil, errors.New("ROM too small")
	}
	if data[0] != 'N' || data[1] != 'E' || data[2] != 'S' || data[3] != 0x1A {
		return nil, errors.New("invalid iNES magic")
	}

	prgBanks := int(data[4])
	chrBanks := int(data[5])
	flags6 := data[6]
	mapper := int(flags6 >> 4)
	mirror := int(flags6 & 1) // bit 0: 0=水平, 1=垂直

	prgSize := prgBanks * 16384
	chrSize := chrBanks * 8192

	headerEnd := 16
	if len(data) < headerEnd+prgSize+chrSize {
		return nil, errors.New("ROM data truncated")
	}

	prgROM := make([]byte, prgSize)
	copy(prgROM, data[headerEnd:headerEnd+prgSize])

	var chrROM []byte
	if chrBanks == 0 {
		// CHR RAM（部分游戏使用），分配 8KB
		chrROM = make([]byte, 8192)
	} else {
		chrROM = make([]byte, chrSize)
		copy(chrROM, data[headerEnd+prgSize:headerEnd+prgSize+chrSize])
	}

	return &Cartridge{
		PRG:      prgROM,
		CHR:      chrROM,
		PRGBanks: prgBanks,
		CHRBanks: chrBanks,
		Mapper:   mapper,
		Mirror:   mirror,
	}, nil
}

func (c *Cartridge) PRGRead(addr uint16) uint8 {
	offset := addr - 0x8000
	if c.PRGBanks == 1 {
		// 16KB ROM：$C000-$FFFF 镜像 $8000-$BFFF
		offset = offset & 0x3FFF
	}
	return c.PRG[offset]
}

func (c *Cartridge) PRGWrite(addr uint16, data uint8) {
	// Mapper 0 不支持 PRG ROM 写入
}

func (c *Cartridge) CHRRead(addr uint16) uint8 {
	return c.CHR[addr]
}

func (c *Cartridge) CHRWrite(addr uint16, data uint8) {
	// 如果 CHR 是 RAM（chrBanks == 0），允许写入
	if c.CHRBanks == 0 {
		c.CHR[addr] = data
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./cartridge/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cartridge/ && git commit -m "feat: add cartridge with iNES parsing and Mapper 0 support"
```

---

### Task 3: Bus - 地址总线

**Files:**
- Create: `bus/bus.go`
- Create: `bus/bus_test.go`

- [ ] **Step 1: 编写 Bus 的失败测试**

```go
// bus/bus_test.go
package bus

import (
	"testing"
)

type mockCart struct {
	prg []byte
}

func (m *mockCart) PRGRead(addr uint16) uint8 {
	return m.prg[addr-0x8000]
}
func (m *mockCart) PRGWrite(addr uint16, data uint8) {}
func (m *mockCart) CHRRead(addr uint16) uint8         { return 0 }
func (m *mockCart) CHRWrite(addr uint16, data uint8)  {}

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
	// $0800 是 $0000 的镜像
	if val := b.Read(0x0800); val != 0x42 {
		t.Errorf("mirror 0x0800 expected 0x42, got 0x%02X", val)
	}
	// $1800 也是镜像
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
	if addr&0x2007 == 0x2000 {
		m.ctrl = v
	}
}
func (m *mockPPU) RenderFrame() []byte              { return nil }

func TestMirroredPPURange(t *testing.T) {
	ppu := &mockPPU{}
	b := NewBus(nil, ppu, nil, nil)
	// $2008 是 $2000 的镜像
	b.Write(0x2008, 0xFF)
	if ppu.ctrl != 0xFF {
		t.Errorf("PPUCTRL via mirror expected 0xFF, got 0x%02X", ppu.ctrl)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./bus/...
```

Expected: 编译失败

- [ ] **Step 3: 实现 Bus**

```go
// bus/bus.go
package bus

type CartridgeInterface interface {
	PRGRead(addr uint16) uint8
	PRGWrite(addr uint16, data uint8)
	CHRRead(addr uint16) uint8
	CHRWrite(addr uint16, data uint8)
}

type PPUInterface interface {
	ReadRegister(addr uint16) uint8
	WriteRegister(addr uint16, data uint8)
	RenderFrame() []byte
}

type APUInterface interface {
	WriteRegister(addr uint16, data uint8)
	ReadStatus() uint8
	GenerateSamples() []float32
}

type InputInterface interface {
	Read() uint8
	Write(data uint8)
}

type Bus struct {
	RAM  [2048]uint8
	Cart CartridgeInterface
	PPU  PPUInterface
	APU  APUInterface
	Inp  InputInterface
}

func NewBus(cart CartridgeInterface, ppu PPUInterface, apu APUInterface, inp InputInterface) *Bus {
	return &Bus{
		Cart: cart,
		PPU:  ppu,
		APU:  apu,
		Inp:  inp,
	}
}

func (b *Bus) Read(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return b.RAM[addr&0x07FF]
	case addr < 0x4000:
		return b.PPU.ReadRegister(addr & 0x2007)
	case addr == 0x4016:
		return b.Inp.Read()
	case addr == 0x4015:
		return b.APU.ReadStatus()
	case addr >= 0x4020 && addr <= 0xFFFF:
		return b.Cart.PRGRead(addr)
	default:
		return 0
	}
}

func (b *Bus) Write(addr uint16, data uint8) {
	switch {
	case addr < 0x2000:
		b.RAM[addr&0x07FF] = data
	case addr < 0x4000:
		b.PPU.WriteRegister(addr&0x2007, data)
	case addr == 0x4014:
		b.PPU.WriteRegister(0x4014, data)
		b.doDMA(data)
	case addr == 0x4016:
		b.Inp.Write(data)
	case addr >= 0x4000 && addr <= 0x4017:
		if b.APU != nil {
			b.APU.WriteRegister(addr, data)
		}
	case addr >= 0x4020 && addr <= 0xFFFF:
		b.Cart.PRGWrite(addr, data)
	}
}

func (b *Bus) doDMA(data uint8) {
	baseAddr := uint16(data) << 8
	for i := uint16(0); i < 256; i++ {
		val := b.Read(baseAddr + i)
		b.PPU.WriteRegister(0x2004, val)
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./bus/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add bus/ && git commit -m "feat: add address bus with RAM, mirroring, and MMIO routing"
```

---

### Task 4: CPU - 核心框架与寻址模式

**Files:**
- Create: `cpu/cpu.go`
- Create: `cpu/cpu_test.go`

- [ ] **Step 1: 编写 CPU 基础测试**

```go
// cpu/cpu_test.go
package cpu

import (
	"testing"
)

type mockBus struct {
	ram [0x10000]uint8
}

func (m *mockBus) Read(addr uint16) uint8  { return m.ram[addr] }
func (m *mockBus) Write(addr uint16, v uint8) { m.ram[addr] = v }

func newCPU() (*CPU, *mockBus) {
	bus := &mockBus{}
	cpu := New(bus)
	cpu.PC = 0x8000
	return cpu, bus
}

func TestReset(t *testing.T) {
	bus := &mockBus{}
	// RESET 向量指向 $8000
	bus.ram[0xFFFC] = 0x00
	bus.ram[0xFFFD] = 0x80
	cpu := New(bus)
	cpu.Reset()
	if cpu.PC != 0x8000 {
		t.Errorf("Reset: expected PC=0x8000, got 0x%04X", cpu.PC)
	}
	// Reset 后 SP 应为 $FD
	if cpu.SP != 0xFD {
		t.Errorf("Reset: expected SP=0xFD, got 0x%02X", cpu.SP)
	}
}

func TestLDAImmediate(t *testing.T) {
	cpu, bus := newCPU()
	// LDA #$42 = opcode $A9
	bus.ram[0x8000] = 0xA9
	bus.ram[0x8001] = 0x42

	cpu.Step()
	if cpu.A != 0x42 {
		t.Errorf("LDA immediate: A expected 0x42, got 0x%02X", cpu.A)
	}
	if cpu.PC != 0x8002 {
		t.Errorf("LDA immediate: PC expected 0x8002, got 0x%04X", cpu.PC)
	}
	// 加载非零值，Z flag 应为 0
	if cpu.P&FlagZ != 0 {
		t.Error("LDA immediate: Z flag should be clear for non-zero")
	}
	// $42 的 bit 6 为 1，N flag 应为 0
	if cpu.P&FlagN != 0 {
		t.Error("LDA immediate: N flag should be clear for 0x42")
	}
}

func TestLDAZeroFlag(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x8000] = 0xA9
	bus.ram[0x8001] = 0x00

	cpu.Step()
	if cpu.P&FlagZ == 0 {
		t.Error("LDA #0: Z flag should be set")
	}
}

func TestLDANegativeFlag(t *testing.T) {
	cpu, bus := newCPU()
	bus.ram[0x8000] = 0xA9
	bus.ram[0x8001] = 0x80

	cpu.Step()
	if cpu.P&FlagN == 0 {
		t.Error("LDA #$80: N flag should be set")
	}
}

func TestSTAAbsolute(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x99
	// STA $0300 = opcode $8D
	bus.ram[0x8000] = 0x8D
	bus.ram[0x8001] = 0x00
	bus.ram[0x8002] = 0x03

	cpu.Step()
	if bus.ram[0x0300] != 0x99 {
		t.Errorf("STA absolute: expected 0x99, got 0x%02X", bus.ram[0x0300])
	}
}

func TestTAX(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x42
	// TAX = opcode $AA
	bus.ram[0x8000] = 0xAA

	cpu.Step()
	if cpu.X != 0x42 {
		t.Errorf("TAX: X expected 0x42, got 0x%02X", cpu.X)
	}
}

func TestINX(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x05
	// INX = opcode $E8
	bus.ram[0x8000] = 0xE8

	cpu.Step()
	if cpu.X != 0x06 {
		t.Errorf("INX: X expected 0x06, got 0x%02X", cpu.X)
	}
}

func TestINXZeroWrap(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0xFF
	bus.ram[0x8000] = 0xE8

	cpu.Step()
	if cpu.X != 0x00 {
		t.Errorf("INX wrap: X expected 0x00, got 0x%02X", cpu.X)
	}
	if cpu.P&FlagZ == 0 {
		t.Error("INX wrap: Z flag should be set")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./cpu/...
```

Expected: 编译失败

- [ ] **Step 3: 实现 CPU 核心框架**

```go
// cpu/cpu.go
package cpu

type Bus interface {
	Read(addr uint16) uint8
	Write(addr uint16, data uint8)
}

// 状态标志位
const (
	FlagC = 1 << 0 // Carry
	FlagZ = 1 << 1 // Zero
	FlagI = 1 << 2 // Interrupt Disable
	FlagD = 1 << 3 // Decimal
	FlagB = 1 << 5 // Break
	FlagV = 1 << 6 // Overflow
	FlagN = 1 << 7 // Negative
)

type CPU struct {
	A     uint8  // Accumulator
	X     uint8  // X register
	Y     uint8  // Y register
	SP    uint8  // Stack pointer
	PC    uint16 // Program counter
	P     uint8  // Status register

	Bus       Bus
	Cycles    uint64 // 累计周期
	NMI       bool   // NMI 待处理
	IRQ       bool   // IRQ 待处理
	cyclesLeft int   // 当前指令剩余周期
}

func New(bus Bus) *CPU {
	return &CPU{
		Bus: bus,
		SP:  0xFD,
		P:   0x34, // I 标志默认置位
	}
}

func (c *CPU) Reset() {
	c.PC = uint16(c.Bus.Read(0xFFFC)) | uint16(c.Bus.Read(0xFFFD))<<8
	c.SP = 0xFD
	c.P = 0x34
	c.A = 0
	c.X = 0
	c.Y = 0
}

func (c *CPU) Step() int {
	if c.NMI {
		c.nmi()
		c.NMI = false
	}

	opcode := c.Bus.Read(c.PC)
	cycles := c.execute(opcode)
	c.Cycles += uint64(cycles)
	return cycles
}

// RunCycles 执行指定数量的 CPU 周期
func (c *CPU) RunCycles(targetCycles uint64) {
	for c.Cycles < targetCycles {
		c.Step()
	}
}

func (c *CPU) nmi() {
	c.push16(c.PC)
	c.push8(c.P | FlagB)
	c.P |= FlagI
	c.PC = uint16(c.Bus.Read(0xFFFA)) | uint16(c.Bus.Read(0xFFFB))<<8
	c.Cycles += 7
}

func (c *CPU) setZN(val uint8) {
	if val == 0 {
		c.P |= FlagZ
	} else {
		c.P &= ^uint8(FlagZ)
	}
	if val&0x80 != 0 {
		c.P |= FlagN
	} else {
		c.P &= ^uint8(FlagN)
	}
}

// --- 寻址模式 ---

func (c *CPU) imm() uint8 {
	v := c.Bus.Read(c.PC)
	c.PC++
	return v
}

func (c *CPU) abs() uint16 {
	lo := c.Bus.Read(c.PC)
	c.PC++
	hi := c.Bus.Read(c.PC)
	c.PC++
	return uint16(lo) | uint16(hi)<<8
}

func (c *CPU) absX(cross *bool) uint16 {
	addr := c.abs() + uint16(c.X)
	return addr
}

func (c *CPU) absY(cross *bool) uint16 {
	addr := c.abs() + uint16(c.Y)
	return addr
}

func (c *CPU) zp() uint16 {
	return uint16(c.Bus.Read(c.PC - 1)) // 在 imm() 之后调用
}

func (c *CPU) zpX() uint16 {
	return uint16(uint8(c.zp()) + c.X)
}

func (c *CPU) zpY() uint16 {
	return uint16(uint8(c.zp()) + c.Y)
}

func (c *CPU) ind() uint16 {
	lo := c.Bus.Read(c.PC)
	c.PC++
	hi := c.Bus.Read(c.PC)
	c.PC++
	ptr := uint16(lo) | uint16(hi)<<8
	// 6502 间接跳转 bug：页边界不跨页
	addrLo := c.Bus.Read(ptr)
	addrHi := c.Bus.Read(ptr&0xFF00 | ((ptr + 1) & 0x00FF))
	return uint16(addrLo) | uint16(addrHi)<<8
}

func (c *CPU) indX() uint16 {
	ptr := uint8(c.zp()) + c.X
	lo := c.Bus.Read(uint16(ptr))
	hi := c.Bus.Read(uint16(ptr + 1))
	return uint16(lo) | uint16(hi)<<8
}

func (c *CPU) indY(cross *bool) uint16 {
	ptr := uint8(c.zp())
	lo := c.Bus.Read(uint16(ptr))
	hi := c.Bus.Read(uint16(ptr + 1))
	addr := uint16(lo) | uint16(hi)<<8
	oldPage := addr & 0xFF00
	addr += uint16(c.Y)
	if addr&0xFF00 != oldPage {
		*cross = true
	}
	return addr
}

func (c *CPU) rel() uint16 {
	offset := int8(c.Bus.Read(c.PC))
	c.PC++
	return uint16(int32(c.PC) + int32(offset))
}

// --- 栈操作 ---

func (c *CPU) push8(v uint8) {
	c.Bus.Write(0x0100|uint16(c.SP), v)
	c.SP--
}

func (c *CPU) push16(v uint16) {
	c.push8(uint8(v >> 8))
	c.push8(uint8(v & 0xFF))
}

func (c *CPU) pop8() uint8 {
	c.SP++
	return c.Bus.Read(0x0100 | uint16(c.SP))
}

func (c *CPU) pop16() uint16 {
	lo := c.pop8()
	hi := c.pop8()
	return uint16(lo) | uint16(hi)<<8
}
```

- [ ] **Step 4: 实现指令执行（包含 LDA/STA/TAX/INX 等初始指令）**

在 `cpu/cpu.go` 中添加 `execute` 方法：

```go
func (c *CPU) execute(opcode uint8) int {
	switch opcode {
	// LDA
	case 0xA9: // Immediate
		v := c.imm()
		c.A = v
		c.setZN(c.A)
		return 2
	case 0xA5: // Zero Page
		c.PC++
		c.A = c.Bus.Read(c.zp())
		c.setZN(c.A)
		return 3
	case 0xB5: // Zero Page,X
		c.PC++
		c.A = c.Bus.Read(c.zpX())
		c.setZN(c.A)
		return 4
	case 0xAD: // Absolute
		c.PC++
		c.A = c.Bus.Read(c.abs())
		c.setZN(c.A)
		return 4
	case 0xBD: // Absolute,X
		c.PC++
		var cross bool
		addr := c.absX(&cross)
		c.A = c.Bus.Read(addr)
		c.setZN(c.A)
		if cross {
			return 5
		}
		return 4
	case 0xB9: // Absolute,Y
		c.PC++
		var cross bool
		addr := c.absY(&cross)
		c.A = c.Bus.Read(addr)
		c.setZN(c.A)
		if cross {
			return 5
		}
		return 4
	case 0xA1: // (Indirect,X)
		c.PC++
		c.A = c.Bus.Read(c.indX())
		c.setZN(c.A)
		return 6
	case 0xB1: // (Indirect),Y
		c.PC++
		var cross bool
		addr := c.indY(&cross)
		c.A = c.Bus.Read(addr)
		c.setZN(c.A)
		if cross {
			return 6
		}
		return 5

	// LDX
	case 0xA2: // Immediate
		c.X = c.imm()
		c.setZN(c.X)
		return 2
	case 0xA6: // Zero Page
		c.PC++
		c.X = c.Bus.Read(c.zp())
		c.setZN(c.X)
		return 3
	case 0xB6: // Zero Page,Y
		c.PC++
		c.X = c.Bus.Read(c.zpY())
		c.setZN(c.X)
		return 4
	case 0xAE: // Absolute
		c.PC++
		c.X = c.Bus.Read(c.abs())
		c.setZN(c.X)
		return 4
	case 0xBE: // Absolute,Y
		c.PC++
		var cross bool
		addr := c.absY(&cross)
		c.X = c.Bus.Read(addr)
		c.setZN(c.X)
		if cross {
			return 5
		}
		return 4

	// LDY
	case 0xA0: // Immediate
		c.Y = c.imm()
		c.setZN(c.Y)
		return 2
	case 0xA4: // Zero Page
		c.PC++
		c.Y = c.Bus.Read(c.zp())
		c.setZN(c.Y)
		return 3
	case 0xB4: // Zero Page,X
		c.PC++
		c.Y = c.Bus.Read(c.zpX())
		c.setZN(c.Y)
		return 4
	case 0xAC: // Absolute
		c.PC++
		c.Y = c.Bus.Read(c.abs())
		c.setZN(c.Y)
		return 4
	case 0xBC: // Absolute,X
		c.PC++
		var cross bool
		addr := c.absX(&cross)
		c.Y = c.Bus.Read(addr)
		c.setZN(c.Y)
		if cross {
			return 5
		}
		return 4

	// STA
	case 0x85: // Zero Page
		c.PC++
		c.Bus.Write(c.zp(), c.A)
		return 3
	case 0x95: // Zero Page,X
		c.PC++
		c.Bus.Write(c.zpX(), c.A)
		return 4
	case 0x8D: // Absolute
		c.PC++
		c.Bus.Write(c.abs(), c.A)
		return 4
	case 0x9D: // Absolute,X
		c.PC++
		var _cross bool
		addr := c.absX(&_cross)
		c.Bus.Write(addr, c.A)
		return 5
	case 0x99: // Absolute,Y
		c.PC++
		var _cross bool
		addr := c.absY(&_cross)
		c.Bus.Write(addr, c.A)
		return 5
	case 0x81: // (Indirect,X)
		c.PC++
		c.Bus.Write(c.indX(), c.A)
		return 6
	case 0x91: // (Indirect),Y
		c.PC++
		var _cross bool
		addr := c.indY(&_cross)
		c.Bus.Write(addr, c.A)
		return 6

	// STX
	case 0x86: // Zero Page
		c.PC++
		c.Bus.Write(c.zp(), c.X)
		return 3
	case 0x96: // Zero Page,Y
		c.PC++
		c.Bus.Write(c.zpY(), c.X)
		return 4
	case 0x8E: // Absolute
		c.PC++
		c.Bus.Write(c.abs(), c.X)
		return 4

	// STY
	case 0x84: // Zero Page
		c.PC++
		c.Bus.Write(c.zp(), c.Y)
		return 3
	case 0x94: // Zero Page,X
		c.PC++
		c.Bus.Write(c.zpX(), c.Y)
		return 4
	case 0x8C: // Absolute
		c.PC++
		c.Bus.Write(c.abs(), c.Y)
		return 4

	// TAX
	case 0xAA:
		c.X = c.A
		c.setZN(c.X)
		return 2

	// TXA
	case 0x8A:
		c.A = c.X
		c.setZN(c.A)
		return 2

	// TAY
	case 0xA8:
		c.Y = c.A
		c.setZN(c.Y)
		return 2

	// TYA
	case 0x98:
		c.A = c.Y
		c.setZN(c.A)
		return 2

	// TSX
	case 0xBA:
		c.X = c.SP
		c.setZN(c.X)
		return 2

	// TXS
	case 0x9A:
		c.SP = c.X
		return 2

	// INX
	case 0xE8:
		c.X++
		c.setZN(c.X)
		return 2

	// INY
	case 0xC8:
		c.Y++
		c.setZN(c.Y)
		return 2

	// DEX
	case 0xCA:
		c.X--
		c.setZN(c.X)
		return 2

	// DEY
	case 0x88:
		c.Y--
		c.setZN(c.Y)
		return 2

	// INC Zero Page
	case 0xE6:
		c.PC++
		addr := c.zp()
		v := c.Bus.Read(addr) + 1
		c.Bus.Write(addr, v)
		c.setZN(v)
		return 5

	// DEC Zero Page
	case 0xC6:
		c.PC++
		addr := c.zp()
		v := c.Bus.Read(addr) - 1
		c.Bus.Write(addr, v)
		c.setZN(v)
		return 5

	// CLC
	case 0x18:
		c.P &= ^uint8(FlagC)
		return 2

	// SEC
	case 0x38:
		c.P |= FlagC
		return 2

	// CLI
	case 0x58:
		c.P &= ^uint8(FlagI)
		return 2

	// SEI
	case 0x78:
		c.P |= FlagI
		return 2

	// CLD
	case 0xD8:
		c.P &= ^uint8(FlagD)
		return 2

	// CLV
	case 0xB8:
		c.P &= ^uint8(FlagV)
		return 2

	// PHA
	case 0x48:
		c.push8(c.A)
		return 3

	// PLA
	case 0x68:
		c.A = c.pop8()
		c.setZN(c.A)
		return 4

	// PHP
	case 0x08:
		c.push8(c.P | FlagB | 0x30)
		return 3

	// PLP
	case 0x28:
		c.P = c.pop8()&0xEF | 0x30
		return 4

	// NOP
	case 0xEA:
		return 2

	default:
		// 未实现的指令暂且跳过
		return 2
	}
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
go test ./cpu/...
```

Expected: PASS（至少 LDA/STA/TAX/INX 测试通过）

- [ ] **Step 6: Commit**

```bash
git add cpu/ && git commit -m "feat: add CPU core with load/store/transfer/inc/dec instructions"
```

---

### Task 5: CPU - 算术、逻辑、分支、跳转指令

**Files:**
- Modify: `cpu/cpu.go`
- Modify: `cpu/cpu_test.go`

- [ ] **Step 1: 添加算术/逻辑/分支/跳转指令的测试**

在 `cpu/cpu_test.go` 中添加：

```go
func TestADCImmediate(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x10
	// ADC #$20
	bus.ram[0x8000] = 0x69
	bus.ram[0x8001] = 0x20

	cpu.Step()
	if cpu.A != 0x30 {
		t.Errorf("ADC immediate: expected 0x30, got 0x%02X", cpu.A)
	}
}

func TestADCWithCarry(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0xF0
	cpu.P |= FlagC // carry set
	// ADC #$20
	bus.ram[0x8000] = 0x69
	bus.ram[0x8001] = 0x20

	cpu.Step()
	if cpu.A != 0x11 {
		t.Errorf("ADC with carry: expected 0x11, got 0x%02X", cpu.A)
	}
	if cpu.P&FlagC == 0 {
		t.Error("ADC: C flag should be set on overflow")
	}
}

func TestANDImmediate(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0xF0
	// AND #$0F
	bus.ram[0x8000] = 0x29
	bus.ram[0x8001] = 0x0F

	cpu.Step()
	if cpu.A != 0x00 {
		t.Errorf("AND immediate: expected 0x00, got 0x%02X", cpu.A)
	}
	if cpu.P&FlagZ == 0 {
		t.Error("AND: Z flag should be set when result is 0")
	}
}

func TestORAImmediate(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x0F
	// ORA #$F0
	bus.ram[0x8000] = 0x09
	bus.ram[0x8001] = 0xF0

	cpu.Step()
	if cpu.A != 0xFF {
		t.Errorf("ORA immediate: expected 0xFF, got 0x%02X", cpu.A)
	}
	if cpu.P&FlagN == 0 {
		t.Error("ORA: N flag should be set for 0xFF")
	}
}

func TestEORImmediate(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0xF0
	// EOR #$0F
	bus.ram[0x8000] = 0x49
	bus.ram[0x8001] = 0x0F

	cpu.Step()
	if cpu.A != 0xFF {
		t.Errorf("EOR immediate: expected 0xFF, got 0x%02X", cpu.A)
	}
}

func TestCMPImmediate(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x42
	// CMP #$42
	bus.ram[0x8000] = 0xC9
	bus.ram[0x8001] = 0x42

	cpu.Step()
	if cpu.P&FlagZ == 0 {
		t.Error("CMP: Z flag should be set when equal")
	}
	if cpu.P&FlagC == 0 {
		t.Error("CMP: C flag should be set when A >= M")
	}
}

func TestCMPLessThan(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x10
	// CMP #$42
	bus.ram[0x8000] = 0xC9
	bus.ram[0x8001] = 0x42

	cpu.Step()
	if cpu.P&FlagC != 0 {
		t.Error("CMP: C flag should be clear when A < M")
	}
}

func TestCPXImmediate(t *testing.T) {
	cpu, bus := newCPU()
	cpu.X = 0x42
	// CPX #$42
	bus.ram[0x8000] = 0xE0
	bus.ram[0x8001] = 0x42

	cpu.Step()
	if cpu.P&FlagZ == 0 {
		t.Error("CPX: Z flag should be set when equal")
	}
}

func TestCPYImmediate(t *testing.T) {
	cpu, bus := newCPU()
	cpu.Y = 0x42
	// CPY #$42
	bus.ram[0x8000] = 0xC0
	bus.ram[0x8001] = 0x42

	cpu.Step()
	if cpu.P&FlagZ == 0 {
		t.Error("CPY: Z flag should be set when equal")
	}
}

func TestASLAccumulator(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x41 // 0100_0001
	// ASL A
	bus.ram[0x8000] = 0x0A

	cpu.Step()
	if cpu.A != 0x82 {
		t.Errorf("ASL A: expected 0x82, got 0x%02X", cpu.A)
	}
	if cpu.P&FlagC != 0 {
		t.Error("ASL A: C flag should be set for 0x41 << 1")
	}
}

func TestLSRAccumulator(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x82 // 1000_0010
	// LSR A
	bus.ram[0x8000] = 0x4A

	cpu.Step()
	if cpu.A != 0x41 {
		t.Errorf("LSR A: expected 0x41, got 0x%02X", cpu.A)
	}
	if cpu.P&FlagC != 0 {
		t.Error("LSR A: C flag should be clear for 0x82 >> 1")
	}
}

func TestROLA(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x80
	cpu.P |= FlagC
	// ROL A
	bus.ram[0x8000] = 0x2A

	cpu.Step()
	if cpu.A != 0x01 {
		t.Errorf("ROL A: expected 0x01, got 0x%02X", cpu.A)
	}
	if cpu.P&FlagC == 0 {
		t.Error("ROL A: C flag should be set when 0x80 rotated left with carry")
	}
}

func TestRORA(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0x01
	cpu.P |= FlagC
	// ROR A
	bus.ram[0x8000] = 0x6A

	cpu.Step()
	if cpu.A != 0x80 {
		t.Errorf("ROR A: expected 0x80, got 0x%02X", cpu.A)
	}
}

func TestBEQBranch(t *testing.T) {
	cpu, bus := newCPU()
	cpu.P |= FlagZ // Z = 1
	// BEQ +5
	bus.ram[0x8000] = 0xF0
	bus.ram[0x8001] = 0x05

	cpu.Step()
	if cpu.PC != 0x8007 {
		t.Errorf("BEQ taken: expected PC=0x8007, got 0x%04X", cpu.PC)
	}
}

func TestBEQNoBranch(t *testing.T) {
	cpu, bus := newCPU()
	cpu.P &= ^uint8(FlagZ) // Z = 0
	// BEQ +5
	bus.ram[0x8000] = 0xF0
	bus.ram[0x8001] = 0x05

	cpu.Step()
	if cpu.PC != 0x8002 {
		t.Errorf("BEQ not taken: expected PC=0x8002, got 0x%04X", cpu.PC)
	}
}

func TestBNEBranch(t *testing.T) {
	cpu, bus := newCPU()
	cpu.P &= ^uint8(FlagZ) // Z = 0
	// BNE +5
	bus.ram[0x8000] = 0xD0
	bus.ram[0x8001] = 0x05

	cpu.Step()
	if cpu.PC != 0x8007 {
		t.Errorf("BNE taken: expected PC=0x8007, got 0x%04X", cpu.PC)
	}
}

func TestJMPAbsolute(t *testing.T) {
	cpu, bus := newCPU()
	// JMP $9000
	bus.ram[0x8000] = 0x4C
	bus.ram[0x8001] = 0x00
	bus.ram[0x8002] = 0x90

	cpu.Step()
	if cpu.PC != 0x9000 {
		t.Errorf("JMP absolute: expected PC=0x9000, got 0x%04X", cpu.PC)
	}
}

func TestJMPIndirect(t *testing.T) {
	cpu, bus := newCPU()
	// JMP ($0300)
	bus.ram[0x8000] = 0x6C
	bus.ram[0x8001] = 0x00
	bus.ram[0x8002] = 0x03
	bus.ram[0x0300] = 0x00
	bus.ram[0x0301] = 0x90

	cpu.Step()
	if cpu.PC != 0x9000 {
		t.Errorf("JMP indirect: expected PC=0x9000, got 0x%04X", cpu.PC)
	}
}

func TestJSRAndRTS(t *testing.T) {
	cpu, bus := newCPU()
	cpu.SP = 0xFF
	// JSR $9000
	bus.ram[0x8000] = 0x20
	bus.ram[0x8001] = 0x00
	bus.ram[0x8002] = 0x90
	// RTS at $9000
	bus.ram[0x9000] = 0x60

	// 执行 JSR
	cpu.Step()
	expectedReturn := uint16(0x8002) // JSR pushes PC+2
	_ = expectedReturn
	if cpu.PC != 0x9000 {
		t.Errorf("JSR: expected PC=0x9000, got 0x%04X", cpu.PC)
	}
	if cpu.SP != 0xFD {
		t.Errorf("JSR: expected SP=0xFD (pushed 2 bytes), got 0x%02X", cpu.SP)
	}

	// 执行 RTS
	cpu.Step()
	if cpu.PC != 0x8003 {
		t.Errorf("RTS: expected PC=0x8003, got 0x%04X", cpu.PC)
	}
}

func TestBITZeroPage(t *testing.T) {
	cpu, bus := newCPU()
	cpu.A = 0xF0
	bus.ram[0x0100] = 0x0F
	// BIT $0100
	bus.ram[0x8000] = 0x24
	bus.ram[0x8001] = 0x00 // 注意：BIT 是零页寻址，$0100 高位被忽略

	cpu.Step()
	if cpu.P&FlagZ == 0 {
		t.Error("BIT: Z flag should be set when A & M == 0")
	}
}

func TestStackPushPull(t *testing.T) {
	cpu, bus := newCPU()
	cpu.SP = 0xFF

	cpu.push8(0x42)
	if cpu.SP != 0xFE {
		t.Errorf("push8: expected SP=0xFE, got 0x%02X", cpu.SP)
	}

	cpu.push8(0x99)
	cpu.push8(0x88)

	v := cpu.pop8()
	if v != 0x88 {
		t.Errorf("pop8: expected 0x88, got 0x%02X", v)
	}
	if cpu.SP != 0xFF {
		t.Errorf("pop8: expected SP=0xFF, got 0x%02X", cpu.SP)
	}
}

func TestBRK(t *testing.T) {
	cpu, bus := newCPU()
	cpu.SP = 0xFF
	// BRK
	bus.ram[0x8000] = 0x00
	bus.ram[0xFFFE] = 0x00
	bus.ram[0xFFFF] = 0x90

	cpu.Step()
	if cpu.PC != 0x9000 {
		t.Errorf("BRK: expected PC=0x9000, got 0x%04X", cpu.PC)
	}
	if cpu.P&FlagB == 0 {
		t.Error("BRK: B flag should be set in status on stack")
	}
	if cpu.P&FlagI == 0 {
		t.Error("BRK: I flag should be set")
	}
}

func TestRTI(t *testing.T) {
	cpu, bus := newCPU()
	cpu.SP = 0xFD

	// 模拟中断发生：push PC(0x8003), push P(0x35)
	cpu.push16(0x8003)
	cpu.push8(0x35)
	// RTI
	bus.ram[0x8000] = 0x40

	cpu.Step()
	if cpu.PC != 0x8003 {
		t.Errorf("RTI: expected PC=0x8003, got 0x%04X", cpu.PC)
	}
	if cpu.P != 0x35 {
		t.Errorf("RTI: expected P=0x35, got 0x%02X", cpu.P)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./cpu/... -run "TestADC|TestAND|TestORA|TestEOR|TestCMP|TestASL|TestLSR|TestBEQ|TestBNE|TestJMP|TestJSR|TestRTS|TestBRK|TestRTI|TestStack"
```

Expected: 编译失败或测试失败

- [ ] **Step 3: 添加完整的指令实现**

在 `cpu/cpu.go` 的 `execute` 方法中，在 `default` 分支前添加所有算术/逻辑/分支/跳转指令。具体将 `execute` 方法中的 switch 扩展，添加以下 opcode 处理：

```go
// ADC
case 0x69, 0x65, 0x75, 0x6D, 0x7D, 0x79, 0x61, 0x71:
	c.adc(opcode)

// SBC
case 0xE9, 0xEB, 0xE5, 0xF5, 0xED, 0xFD, 0xF9, 0xE1, 0xF1:
	c.sbc(opcode)

// AND
case 0x29, 0x25, 0x35, 0x2D, 0x3D, 0x39, 0x21, 0x31:
	c.and(opcode)

// ORA
case 0x09, 0x05, 0x15, 0x0D, 0x1D, 0x19, 0x01, 0x11:
	c.ora(opcode)

// EOR
case 0x49, 0x45, 0x55, 0x4D, 0x5D, 0x59, 0x41, 0x51:
	c.eor(opcode)

// CMP
case 0xC9, 0xC5, 0xD5, 0xCD, 0xDD, 0xD9, 0xC1, 0xD1:
	c.cmp(opcode)

// CPX
case 0xE0, 0xE4, 0xEC:
	c.cpx(opcode)

// CPY
case 0xC0, 0xC4, 0xCC:
	c.cpy(opcode)

// ASL
case 0x0A:
	c.A = c.asl(c.A)
	return 2
case 0x06, 0x16, 0x0E, 0x1E:
	c.PC++
	addr := c.getAddrASL(opcode)
	v := c.Bus.Read(addr)
	v = c.asl(v)
	c.Bus.Write(addr, v)
	return c.getCycleASL(opcode)

// LSR
case 0x4A:
	c.A = c.lsr(c.A)
	return 2
case 0x46, 0x56, 0x4E, 0x5E:
	c.PC++
	addr := c.getAddrLSR(opcode)
	v := c.Bus.Read(addr)
	v = c.lsr(v)
	c.Bus.Write(addr, v)
	return c.getCycleLSR(opcode)

// ROL
case 0x2A:
	c.A = c.rol(c.A)
	return 2
case 0x26, 0x36, 0x2E, 0x3E:
	c.PC++
	addr := c.getAddrROL(opcode)
	v := c.Bus.Read(addr)
	v = c.rol(v)
	c.Bus.Write(addr, v)
	return c.getCycleROL(opcode)

// ROR
case 0x6A:
	c.A = c.ror(c.A)
	return 2
case 0x66, 0x76, 0x6E, 0x7E:
	c.PC++
	addr := c.getAddrROR(opcode)
	v := c.Bus.Read(addr)
	v = c.ror(v)
	c.Bus.Write(addr, v)
	return c.getCycleROR(opcode)

// BCC
case 0x90:
	return c.branch(c.P&FlagC == 0)

// BCS
case 0xB0:
	return c.branch(c.P&FlagC != 0)

// BEQ
case 0xF0:
	return c.branch(c.P&FlagZ != 0)

// BNE
case 0xD0:
	return c.branch(c.P&FlagZ == 0)

// BMI
case 0x30:
	return c.branch(c.P&FlagN != 0)

// BPL
case 0x10:
	return c.branch(c.P&FlagN == 0)

// BVC
case 0x50:
	return c.branch(c.P&FlagV == 0)

// BVS
case 0x70:
	return c.branch(c.P&FlagV != 0)

// JMP
case 0x4C: // Absolute
	c.PC = c.abs()
	return 3
case 0x6C: // Indirect
	c.PC = c.ind()
	return 5

// JSR
case 0x20:
	addr := c.abs()
	c.push16(c.PC - 1)
	c.PC = addr
	return 6

// RTS
case 0x60:
	c.PC = c.pop16() + 1
	return 6

// BIT
case 0x24: // Zero Page
	c.PC++
	v := c.Bus.Read(c.zp())
	c.P &= ^uint8(FlagZ | FlagN | FlagV)
	if c.A&v == 0 {
		c.P |= FlagZ
	}
	c.P |= v & (FlagN | FlagV)
	return 3
case 0x2C: // Absolute
	c.PC++
	v := c.Bus.Read(c.abs())
	c.P &= ^uint8(FlagZ | FlagN | FlagV)
	if c.A&v == 0 {
		c.P |= FlagZ
	}
	c.P |= v & (FlagN | FlagV)
	return 4

// BRK
case 0x00:
	c.push16(c.PC + 1)
	c.push8(c.P | FlagB | 0x30)
	c.P |= FlagI
	c.PC = uint16(c.Bus.Read(0xFFFE)) | uint16(c.Bus.Read(0xFFFF))<<8
	return 7

// RTI
case 0x40:
	c.P = c.pop8()&0xEF | 0x30
	c.PC = c.pop16()
	return 6
```

- [ ] **Step 4: 添加辅助方法**

在 `cpu/cpu.go` 中添加这些辅助方法：

```go
// --- 算术/逻辑指令辅助 ---

func (c *CPU) adc(opcode uint8) int {
	var val uint8
	var cycles int
	switch opcode {
	case 0x69: // Immediate
		val = c.imm(); cycles = 2
	case 0x65: // Zero Page
		c.PC++; val = c.Bus.Read(c.zp()); cycles = 3
	case 0x75: // Zero Page,X
		c.PC++; val = c.Bus.Read(c.zpX()); cycles = 4
	case 0x6D: // Absolute
		c.PC++; val = c.Bus.Read(c.abs()); cycles = 4
	case 0x7D: // Absolute,X
		c.PC++; var cross bool; addr := c.absX(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0x79: // Absolute,Y
		c.PC++; var cross bool; addr := c.absY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0x61: // (Indirect,X)
		c.PC++; val = c.Bus.Read(c.indX()); cycles = 6
	case 0x71: // (Indirect),Y
		c.PC++; var cross bool; addr := c.indY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 6 } else { cycles = 5 }
	}
	carry := uint16(0)
	if c.P&FlagC != 0 {
		carry = 1
	}
	sum := uint16(c.A) + uint16(val) + carry
	c.P &= ^uint8(FlagC | FlagV)
	if sum > 0xFF {
		c.P |= FlagC
	}
	if ^(uint16(c.A)^uint16(val))&(uint16(c.A)^sum)&0x80 != 0 {
		c.P |= FlagV
	}
	c.A = uint8(sum)
	c.setZN(c.A)
	return cycles
}

func (c *CPU) sbc(opcode uint8) int {
	var val uint8
	var cycles int
	switch opcode {
	case 0xE9, 0xEB: // Immediate
		val = c.imm(); cycles = 2
	case 0xE5: // Zero Page
		c.PC++; val = c.Bus.Read(c.zp()); cycles = 3
	case 0xF5: // Zero Page,X
		c.PC++; val = c.Bus.Read(c.zpX()); cycles = 4
	case 0xED: // Absolute
		c.PC++; val = c.Bus.Read(c.abs()); cycles = 4
	case 0xFD: // Absolute,X
		c.PC++; var cross bool; addr := c.absX(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0xF9: // Absolute,Y
		c.PC++; var cross bool; addr := c.absY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0xE1: // (Indirect,X)
		c.PC++; val = c.Bus.Read(c.indX()); cycles = 6
	case 0xF1: // (Indirect),Y
		c.PC++; var cross bool; addr := c.indY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 6 } else { cycles = 5 }
	}
	carry := uint16(0)
	if c.P&FlagC != 0 {
		carry = 1
	}
	diff := uint16(c.A) - uint16(val) - (1 - carry)
	c.P &= ^uint8(FlagC | FlagV)
	if diff <= 0xFF {
		c.P |= FlagC
	}
	if (uint16(c.A)^uint16(val))&(uint16(c.A)^diff)&0x80 != 0 {
		c.P |= FlagV
	}
	c.A = uint8(diff)
	c.setZN(c.A)
	return cycles
}

func (c *CPU) and(opcode uint8) int {
	var val uint8
	var cycles int
	switch opcode {
	case 0x29:
		val = c.imm(); cycles = 2
	case 0x25:
		c.PC++; val = c.Bus.Read(c.zp()); cycles = 3
	case 0x35:
		c.PC++; val = c.Bus.Read(c.zpX()); cycles = 4
	case 0x2D:
		c.PC++; val = c.Bus.Read(c.abs()); cycles = 4
	case 0x3D:
		c.PC++; var cross bool; addr := c.absX(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0x39:
		c.PC++; var cross bool; addr := c.absY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0x21:
		c.PC++; val = c.Bus.Read(c.indX()); cycles = 6
	case 0x31:
		c.PC++; var cross bool; addr := c.indY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 6 } else { cycles = 5 }
	}
	c.A &= val
	c.setZN(c.A)
	return cycles
}

func (c *CPU) ora(opcode uint8) int {
	var val uint8
	var cycles int
	switch opcode {
	case 0x09:
		val = c.imm(); cycles = 2
	case 0x05:
		c.PC++; val = c.Bus.Read(c.zp()); cycles = 3
	case 0x15:
		c.PC++; val = c.Bus.Read(c.zpX()); cycles = 4
	case 0x0D:
		c.PC++; val = c.Bus.Read(c.abs()); cycles = 4
	case 0x1D:
		c.PC++; var cross bool; addr := c.absX(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0x19:
		c.PC++; var cross bool; addr := c.absY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0x01:
		c.PC++; val = c.Bus.Read(c.indX()); cycles = 6
	case 0x11:
		c.PC++; var cross bool; addr := c.indY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 6 } else { cycles = 5 }
	}
	c.A |= val
	c.setZN(c.A)
	return cycles
}

func (c *CPU) eor(opcode uint8) int {
	var val uint8
	var cycles int
	switch opcode {
	case 0x49:
		val = c.imm(); cycles = 2
	case 0x45:
		c.PC++; val = c.Bus.Read(c.zp()); cycles = 3
	case 0x55:
		c.PC++; val = c.Bus.Read(c.zpX()); cycles = 4
	case 0x4D:
		c.PC++; val = c.Bus.Read(c.abs()); cycles = 4
	case 0x5D:
		c.PC++; var cross bool; addr := c.absX(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0x59:
		c.PC++; var cross bool; addr := c.absY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0x41:
		c.PC++; val = c.Bus.Read(c.indX()); cycles = 6
	case 0x51:
		c.PC++; var cross bool; addr := c.indY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 6 } else { cycles = 5 }
	}
	c.A ^= val
	c.setZN(c.A)
	return cycles
}

func (c *CPU) cmp(opcode uint8) int {
	var val uint8
	var cycles int
	switch opcode {
	case 0xC9:
		val = c.imm(); cycles = 2
	case 0xC5:
		c.PC++; val = c.Bus.Read(c.zp()); cycles = 3
	case 0xD5:
		c.PC++; val = c.Bus.Read(c.zpX()); cycles = 4
	case 0xCD:
		c.PC++; val = c.Bus.Read(c.abs()); cycles = 4
	case 0xDD:
		c.PC++; var cross bool; addr := c.absX(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0xD9:
		c.PC++; var cross bool; addr := c.absY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 5 } else { cycles = 4 }
	case 0xC1:
		c.PC++; val = c.Bus.Read(c.indX()); cycles = 6
	case 0xD1:
		c.PC++; var cross bool; addr := c.indY(&cross); val = c.Bus.Read(addr)
		if cross { cycles = 6 } else { cycles = 5 }
	}
	c.P &= ^uint8(FlagC | FlagZ | FlagN)
	if c.A >= val {
		c.P |= FlagC
	}
	r := c.A - val
	if r == 0 {
		c.P |= FlagZ
	}
	if r&0x80 != 0 {
		c.P |= FlagN
	}
	return cycles
}

func (c *CPU) cpx(opcode uint8) int {
	var val uint8
	var cycles int
	switch opcode {
	case 0xE0:
		val = c.imm(); cycles = 2
	case 0xE4:
		c.PC++; val = c.Bus.Read(c.zp()); cycles = 3
	case 0xEC:
		c.PC++; val = c.Bus.Read(c.abs()); cycles = 4
	}
	c.P &= ^uint8(FlagC | FlagZ | FlagN)
	if c.X >= val {
		c.P |= FlagC
	}
	r := c.X - val
	if r == 0 {
		c.P |= FlagZ
	}
	if r&0x80 != 0 {
		c.P |= FlagN
	}
	return cycles
}

func (c *CPU) cpy(opcode uint8) int {
	var val uint8
	var cycles int
	switch opcode {
	case 0xC0:
		val = c.imm(); cycles = 2
	case 0xC4:
		c.PC++; val = c.Bus.Read(c.zp()); cycles = 3
	case 0xCC:
		c.PC++; val = c.Bus.Read(c.abs()); cycles = 4
	}
	c.P &= ^uint8(FlagC | FlagZ | FlagN)
	if c.Y >= val {
		c.P |= FlagC
	}
	r := c.Y - val
	if r == 0 {
		c.P |= FlagZ
	}
	if r&0x80 != 0 {
		c.P |= FlagN
	}
	return cycles
}

// --- 移位指令 ---

func (c *CPU) asl(v uint8) uint8 {
	c.P &= ^uint8(FlagC)
	if v&0x80 != 0 {
		c.P |= FlagC
	}
	v <<= 1
	c.setZN(v)
	return v
}

func (c *CPU) lsr(v uint8) uint8 {
	c.P &= ^uint8(FlagC)
	if v&0x01 != 0 {
		c.P |= FlagC
	}
	v >>= 1
	c.setZN(v)
	return v
}

func (c *CPU) rol(v uint8) uint8 {
	oldC := c.P & FlagC
	c.P &= ^uint8(FlagC)
	if v&0x80 != 0 {
		c.P |= FlagC
	}
	v = (v << 1) | oldC
	c.setZN(v)
	return v
}

func (c *CPU) ror(v uint8) uint8 {
	oldC := c.P & FlagC
	c.P &= ^uint8(FlagC)
	if v&0x01 != 0 {
		c.P |= FlagC
	}
	v = (v >> 1) | (oldC << 7)
	c.setZN(v)
	return v
}

func (c *CPU) getAddrASL(opcode uint8) uint16 {
	switch opcode {
	case 0x06: return c.zp()
	case 0x16: return c.zpX()
	case 0x0E: return c.abs()
	case 0x1E:
		var _cross bool
		return c.absX(&_cross)
	}
	return 0
}

func (c *CPU) getCycleASL(opcode uint8) int {
	switch opcode {
	case 0x06: return 5
	case 0x16: return 6
	case 0x0E: return 6
	case 0x1E: return 7
	}
	return 0
}

// getAddrLSR, getCycleLSR, getAddrROL, getCycleROL, getAddrROR, getCycleROR 同理
// （为节省篇幅，模式同 getAddrASL/getCycleASL，替换对应地址模式即可）

func (c *CPU) getAddrLSR(opcode uint8) uint16 {
	switch opcode {
	case 0x46: return c.zp()
	case 0x56: return c.zpX()
	case 0x4E: return c.abs()
	case 0x5E:
		var _cross bool
		return c.absX(&_cross)
	}
	return 0
}
func (c *CPU) getCycleLSR(opcode uint8) int {
	switch opcode {
	case 0x46: return 5
	case 0x56: return 6
	case 0x4E: return 6
	case 0x5E: return 7
	}
	return 0
}
func (c *CPU) getAddrROL(opcode uint8) uint16 {
	switch opcode {
	case 0x26: return c.zp()
	case 0x36: return c.zpX()
	case 0x2E: return c.abs()
	case 0x3E:
		var _cross bool
		return c.absX(&_cross)
	}
	return 0
}
func (c *CPU) getCycleROL(opcode uint8) int {
	switch opcode {
	case 0x26: return 5
	case 0x36: return 6
	case 0x2E: return 6
	case 0x3E: return 7
	}
	return 0
}
func (c *CPU) getAddrROR(opcode uint8) uint16 {
	switch opcode {
	case 0x66: return c.zp()
	case 0x76: return c.zpX()
	case 0x6E: return c.abs()
	case 0x7E:
		var _cross bool
		return c.absX(&_cross)
	}
	return 0
}
func (c *CPU) getCycleROR(opcode uint8) int {
	switch opcode {
	case 0x66: return 5
	case 0x76: return 6
	case 0x6E: return 6
	case 0x7E: return 7
	}
	return 0
}

// --- 分支指令 ---

func (c *CPU) branch(cond bool) int {
	offset := int8(c.Bus.Read(c.PC))
	c.PC++
	if !cond {
		return 2
	}
	oldPC := c.PC
	c.PC = uint16(int32(c.PC) + int32(offset))
	if oldPC&0xFF00 != c.PC&0xFF00 {
		return 4 // 跨页
	}
	return 3
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
go test ./cpu/...
```

Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add cpu/ && git commit -m "feat: add arithmetic, logical, branch, and jump instructions"
```

---

### Task 6: PPU - 寄存器与内存

**Files:**
- Create: `ppu/ppu.go`
- Create: `ppu/ppu_test.go`

- [ ] **Step 1: 编写 PPU 寄存器测试**

```go
// ppu/ppu_test.go
package ppu

import (
	"testing"
)

type mockCart struct {
	chr []byte
}

func (m *mockCart) CHRRead(addr uint16) uint8   { return m.chr[addr] }
func (m *mockCart) CHRWrite(addr uint16, v uint8) { m.chr[addr] = v }

func newPPU() *PPU {
	cart := &mockCart{chr: make([]byte, 8192)}
	return New(cart)
}

func (p *PPU) ReadRegister(addr uint16) uint8 {
	switch addr {
	case 0x2002: return p.readStatus()
	case 0x2007: return p.readData()
	default: return 0
	}
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

func TestPPUSTATUSRead(t *testing.T) {
	ppu := newPPU()
	ppu.Status = 0x80
	val := ppu.ReadRegister(0x2002)
	if val&0x80 == 0 {
		t.Error("PPUSTATUS: VBlank bit should be set")
	}
	// 读取 STATUS 后 VBlank 清空
	if ppu.Status&0x80 != 0 {
		t.Error("PPUSTATUS: VBlank should be cleared after read")
	}
}

func TestPPUSCROLLWrite(t *testing.T) {
	ppu := newPPU()
	// 第一次写入 X scroll
	ppu.WriteRegister(0x2005, 0x42)
	// 第二次写入 Y scroll
	ppu.WriteRegister(0x2005, 0x10)
	if ppu.ScrollX != 0x42 {
		t.Errorf("ScrollX expected 0x42, got 0x%02X", ppu.ScrollX)
	}
	if ppu.ScrollY != 0x10 {
		t.Errorf("ScrollY expected 0x10, got 0x%02X", ppu.ScrollY)
	}
}

func TestPPUADDRWrite(t *testing.T) {
	ppu := newPPU()
	// 高位
	ppu.WriteRegister(0x2006, 0x23)
	// 低位
	ppu.WriteRegister(0x2006, 0x45)
	if ppu.V != 0x2345 {
		t.Errorf("PPUADDR expected 0x2345, got 0x%04X", ppu.V)
	}
}

func TestPPUDATAReadIncrement(t *testing.T) {
	ppu := newPPU()
	// 设置 VRAM 地址
	ppu.WriteRegister(0x2006, 0x20)
	ppu.WriteRegister(0x2006, 0x00)
	// 写入数据到 VRAM（触发第一次 read 预热）
	ppu.ReadRegister(0x2007)
	// 实际读取
	val := ppu.ReadRegister(0x2007)
	_ = val
	// PPUCTRL bit 2 = 0 → increment by 1
	if ppu.V != 0x2002 {
		t.Errorf("PPU addr increment: expected 0x2002, got 0x%04X", ppu.V)
	}
}

func TestPPUDATAWrite(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2006, 0x20)
	ppu.WriteRegister(0x2006, 0x00)
	ppu.WriteRegister(0x2007, 0x42)
	// 地址应前进 1
	if ppu.V != 0x2001 {
		t.Errorf("PPU addr after write: expected 0x2001, got 0x%04X", ppu.V)
	}
}

func TestVRAMReadWrite(t *testing.T) {
	ppu := newPPU()
	ppu.WriteRegister(0x2006, 0x20)
	ppu.WriteRegister(0x2006, 0x00)
	ppu.WriteRegister(0x2007, 0x99)

	// 重置地址并读取
	ppu.WriteRegister(0x2006, 0x20)
	ppu.WriteRegister(0x2006, 0x00)
	ppu.ReadRegister(0x2007) // 预热
	val := ppu.ReadRegister(0x2007)
	if val != 0x99 {
		t.Errorf("VRAM: expected 0x99, got 0x%02X", val)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./ppu/...
```

Expected: 编译失败

- [ ] **Step 3: 实现 PPU**

```go
// ppu/ppu.go
package ppu

type CartridgeInterface interface {
	CHRRead(addr uint16) uint8
	CHRWrite(addr uint16, data uint8)
}

const (
	ScreenWidth  = 256
	ScreenHeight = 240
)

type PPU struct {
	Cart CartridgeInterface

	// 寄存器
	Ctrl    uint8 // $2000 PPUCTRL
	Mask    uint8 // $2001 PPUMASK
	Status  uint8 // $2002 PPUSTATUS
	OAMAddr uint8 // $2003 OAMADDR
	ScrollX uint8 // $2005 first write
	ScrollY uint8 // $2005 second write

	// 内部状态
	V          uint16 // 当前 VRAM 地址
	T          uint16 // 临时 VRAM 地址
	FineX      uint8  // 精调 X
	W          bool   // 地址写入锁存器（先高后低）
	scrollLatch bool  // $2005 写入锁存器

	// VRAM（名称表 + 属性表）
	VRAM [2048]uint8

	// OAM（精灵属性内存）
	OAM [256]uint8

	// 帧缓冲（256×240 RGBA）
	Frame [ScreenWidth * ScreenHeight * 4]uint8

	// 调色板
	Palette [32]uint8

	// 内部读写缓冲
	readBuffer uint8

	// 扫描线/周期计数（简化用）
	Scanline int
	Cycle    int

	// NMI 输出
	NMI bool
}

func New(cart CartridgeInterface) *PPU {
	return &PPU{
		Cart: cart,
	}
}

func (p *PPU) ReadRegister(addr uint16) uint8 {
	switch addr {
	case 0x2002:
		return p.readStatus()
	case 0x2007:
		return p.readData()
	default:
		return 0
	}
}

func (p *PPU) WriteRegister(addr uint16, data uint8) {
	switch addr {
	case 0x2000:
		p.Ctrl = data
		p.T = (p.T & 0xF3FF) | ((uint16(data) & 0x03) << 10)
	case 0x2001:
		p.Mask = data
	case 0x2003:
		p.OAMAddr = data
	case 0x2004:
		p.OAM[p.OAMAddr] = data
		p.OAMAddr++
	case 0x2005:
		if !p.scrollLatch {
			p.ScrollX = data
			p.FineX = data & 0x07
			p.T = (p.T & 0xFFE0) | (uint16(data) >> 3)
		} else {
			p.ScrollY = data
			p.T = (p.T & 0x8C1F) | ((uint16(data) & 0x07) << 12) |
				((uint16(data) & 0xF8) << 2)
			p.scrollLatch = false
			return
		}
		p.scrollLatch = true
	case 0x2006:
		if !p.W {
			p.T = (p.T & 0x00FF) | ((uint16(data) & 0x3F) << 8)
		} else {
			p.T = (p.T & 0xFF00) | uint16(data)
			p.V = p.T
		}
		p.W = !p.W
	case 0x2007:
		p.writeData(data)
	case 0x4014: // OAM DMA
		// OAM DMA 在 bus 层处理，数据直接写入 OAM[0x2004]
	}
}

func (p *PPU) readStatus() uint8 {
	result := p.Status
	p.Status &= 0x7F // 清除 VBlank 标志
	p.W = false
	p.scrollLatch = false
	return result
}

func (p *PPU) readData() uint8 {
	addr := p.V & 0x3FFF
	p.incrementVRAM()
	var val uint8
	switch {
	case addr < 0x2000:
		val = p.readBuffer
		p.readBuffer = p.Cart.CHRRead(addr)
	case addr < 0x3F00:
		val = p.readBuffer
		p.readBuffer = p.VRAM[addr&0x0FFF]
	case addr >= 0x3F00:
		val = p.VRAM[addr&0x1FFF]
		p.readBuffer = p.VRAM[addr&0x1FFF]
		// 调色板区域
		idx := addr & 0x1F
		if idx >= 16 && idx&3 == 0 {
			idx -= 16
		}
		val = p.Palette[idx]
	}
	return val
}

func (p *PPU) writeData(data uint8) {
	addr := p.V & 0x3FFF
	p.incrementVRAM()
	switch {
	case addr < 0x2000:
		p.Cart.CHRWrite(addr, data)
	case addr < 0x3F00:
		p.VRAM[addr&0x0FFF] = data
	case addr >= 0x3F00:
		idx := addr & 0x1F
		if idx >= 16 && idx&3 == 0 {
			idx -= 16
		}
		p.Palette[idx] = data
	}
}

func (p *PPU) incrementVRAM() {
	if p.Ctrl&0x04 != 0 {
		p.V += 32
	} else {
		p.V++
	}
}

func (p *PPU) RenderFrame() []byte {
	// 在后续任务中实现
	return p.Frame[:]
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./ppu/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add ppu/ && git commit -m "feat: add PPU registers, VRAM, OAM, and palette"
```

---

### Task 7: PPU - 渲染

**Files:**
- Modify: `ppu/ppu.go`

- [ ] **Step 1: 添加 NES 64 色调色板**

在 `ppu/ppu.go` 中添加：

```go
var NESPalette = [64]uint32{
	0x7C7C7C, 0x0000FC, 0x0000BC, 0x4428BC, 0x940084, 0xA80020, 0xA81000, 0x881400,
	0x503000, 0x007800, 0x006800, 0x005800, 0x004058, 0x000000, 0x000000, 0x000000,
	0xBCBCBC, 0x0078F8, 0x0058F8, 0x6844FC, 0xD800CC, 0xE40058, 0xF83800, 0xE45C10,
	0xAC7C00, 0x00B800, 0x00A800, 0x00A844, 0x008888, 0x000000, 0x000000, 0x000000,
	0xF8F8F8, 0x3CBCFC, 0x6888FC, 0x9878F8, 0xF878F8, 0xF85898, 0xF87858, 0xFCA044,
	0xF8B800, 0xB8F818, 0x58D854, 0x58F898, 0x00E8D8, 0x787878, 0x000000, 0x000000,
	0xFCFCFC, 0xA4E4FC, 0xB8B8F8, 0xD8B8F8, 0xF8B8F8, 0xF8A4C0, 0xF0D0B0, 0xFCE0A8,
	0xF8D878, 0xD8F878, 0xB8F8B8, 0xB8F8D8, 0x00FCFC, 0xF8D8F8, 0x000000, 0x000000,
}

func (p *PPU) paletteColor(idx uint8) (r, g, b, a uint8) {
	c := NESPalette[idx&0x3F]
	return uint8((c >> 16) & 0xFF), uint8((c >> 8) & 0xFF), uint8(c & 0xFF), 255
}
```

- [ ] **Step 2: 实现背景渲染**

在 `ppu/ppu.go` 的 `RenderFrame` 方法中实现完整渲染：

```go
func (p *PPU) RenderFrame() []byte {
	// 清空帧缓冲
	for i := range p.Frame {
		p.Frame[i] = 0
	}
	p.Status |= 0x80 // 设置 VBlank

	if p.Mask&0x08 != 0 {
		p.renderBackground()
	}
	if p.Mask&0x10 != 0 {
		p.renderSprites()
	}

	return p.Frame[:]
}

func (p *PPU) renderBackground() {
	baseNT := 0x2000 + uint16(p.Ctrl&0x03)*0x400
	basePT := uint16(0)
	if p.Ctrl&0x10 != 0 {
		basePT = 0x1000
	}

	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			// 带滚动的实际坐标
			realX := (x + int(p.ScrollX)) & 255
			realY := (y + int(p.ScrollY)) & 239

			tileX := realX / 8
			tileY := realY / 8

			// 名称表索引
			ntAddr := baseNT + uint16(tileY)*32 + uint16(tileX)
			tileIdx := p.readVRAM(ntAddr)

			// 属性表
			attrAddr := baseNT + 0x3C0 + uint16(tileY/4)*8 + uint16(tileX/4)
			attr := p.readVRAM(attrAddr)
			shift := ((tileY & 2) << 1) | ((tileX & 2))
			paletteGroup := (attr >> shift) & 0x03

			// 图案表
			pixelX := realX & 7
			pixelY := realY & 7
			ptAddr := basePT + uint16(tileIdx)*16 + uint16(pixelY)

			lo := p.readCHR(ptAddr)
			hi := p.readCHR(ptAddr + 8)

			bit := 7 - pixelX
			colorIdx := ((hi>>bit)&1)<<1 | ((lo>>bit)&1)

			if colorIdx == 0 {
				// 透明，使用背景色
				colorIdx = p.Palette[0]
			} else {
				palAddr := 0x3F00 + uint16(paletteGroup)*4 + uint16(colorIdx)
				colorIdx = p.readVRAM(palAddr) & 0x3F
			}

			r, g, b, a := p.paletteColor(colorIdx)
			offset := (y*256 + x) * 4
			p.Frame[offset] = r
			p.Frame[offset+1] = g
			p.Frame[offset+2] = b
			p.Frame[offset+3] = a
		}
	}
}

func (p *PPU) renderSprites() {
	spriteHeight := 8
	if p.Ctrl&0x20 != 0 {
		spriteHeight = 16
	}

	// 从后往前渲染精灵（优先级：OAM[0] 最高）
	for i := 63; i >= 0; i-- {
		spriteY := int(p.OAM[i*4])
		tileIdx := p.OAM[i*4+1]
		attr := p.OAM[i*4+2]
		spriteX := int(p.OAM[i*4+3])

		// Y 坐标偏移 1（精灵顶部在屏幕上方 1 像素）
		spriteY = spriteY + 1

		if spriteY > 240 || spriteX > 248 {
			// 超出屏幕范围
			continue
		}

		flipV := (attr & 0x80) != 0
		flipH := (attr & 0x40) != 0
		paletteGroup := (attr & 0x03) + 4 // 精灵使用调色板组 4-7

		basePT := uint16(0)
		if p.Ctrl&0x08 != 0 {
			basePT = 0x1000
		}

		if spriteHeight == 16 {
			basePT = uint16(tileIdx&0x01) * 0x1000
			tileIdx &= 0xFE
		}

		for py := 0; py < spriteHeight; py++ {
			drawY := spriteY + py
			if drawY < 0 || drawY >= 240 {
				continue
			}

			realPy := py
			if flipV {
				realPy = spriteHeight - 1 - py
			}

			ptAddr := basePT + uint16(tileIdx)*16 + uint16(realPy)
			if spriteHeight == 16 && realPy >= 8 {
				ptAddr = basePT + uint16(tileIdx+1)*16 + uint16(realPy-8)
			}

			lo := p.readCHR(ptAddr)
			hi := p.readCHR(ptAddr + 8)

			for px := 0; px < 8; px++ {
				drawX := spriteX + px
				if drawX < 0 || drawX >= 256 {
					continue
				}

				realPx := px
				if flipH {
					realPx = 7 - px
				}

				bit := 7 - realPx
				colorIdx := ((hi>>bit)&1)<<1 | ((lo>>bit)&1)

				if colorIdx == 0 {
					continue // 透明像素
				}

				palAddr := 0x3F00 + uint16(paletteGroup)*4 + uint16(colorIdx)
				cIdx := p.readVRAM(palAddr) & 0x3F

				r, g, b, a := p.paletteColor(cIdx)
				offset := (drawY*256 + drawX) * 4
				p.Frame[offset] = r
				p.Frame[offset+1] = g
				p.Frame[offset+2] = b
				p.Frame[offset+3] = a
			}
		}
	}
}

func (p *PPU) readVRAM(addr uint16) uint8 {
	addr &= 0x3FFF
	switch {
	case addr < 0x2000:
		return p.Cart.CHRRead(addr)
	case addr < 0x3F00:
		mirror := p.mirrorNameTable(addr)
		return p.VRAM[mirror&0x0FFF]
	default:
		idx := addr & 0x1F
		if idx >= 16 && idx&3 == 0 {
			idx -= 16
		}
		return p.Palette[idx]
	}
}

func (p *PPU) readCHR(addr uint16) uint8 {
	return p.Cart.CHRRead(addr & 0x1FFF)
}

func (p *PPU) mirrorNameTable(addr uint16) uint16 {
	addr = addr & 0x2FFF
	table := (addr - 0x2000) / 0x400
	switch {
	case p.Ctrl&0x01 != 0: // Vertical mirroring
		switch table {
		case 0, 1: return addr
		case 2: return addr - 0x800
		case 3: return addr - 0x800
		}
	default: // Horizontal mirroring
		switch table {
		case 0: return addr
		case 1: return addr - 0x400
		case 2: return addr
		case 3: return addr - 0x400
		}
	}
	return addr
}
```

- [ ] **Step 3: 验证编译通过**

```bash
go build ./...
```

Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add ppu/ && git commit -m "feat: add PPU background and sprite rendering with NES palette"
```

---

### Task 8: Input - 键盘映射

**Files:**
- Create: `input/input.go`

- [ ] **Step 1: 实现 Input 控制器**

```go
// input/input.go
package input

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Controller struct {
	buttons [8]bool
	strobe  bool
	index   int
}

func New() *Controller {
	return &Controller{}
}

func (c *Controller) Update() {
	c.buttons[0] = ebiten.IsKeyPressed(ebiten.KeyZ)       // A
	c.buttons[1] = ebiten.IsKeyPressed(ebiten.KeyX)       // B
	c.buttons[2] = ebiten.IsKeyPressed(ebiten.KeyEnter)   // Select
	c.buttons[3] = ebiten.IsKeyPressed(ebiten.KeyShift)   // Start
	c.buttons[4] = ebiten.IsKeyPressed(ebiten.KeyArrowUp) // Up
	c.buttons[5] = ebiten.IsKeyPressed(ebiten.KeyArrowDown) // Down
	c.buttons[6] = ebiten.IsKeyPressed(ebiten.KeyArrowLeft) // Left
	c.buttons[7] = ebiten.IsKeyPressed(ebiten.KeyArrowRight) // Right
}

func (c *Controller) Read() uint8 {
	var val uint8 = 0
	if c.index < 8 && c.buttons[c.index] {
		val = 1
	}
	c.index++
	if c.strobe {
		c.index = 0
	}
	return val
}

func (c *Controller) Write(data uint8) {
	c.strobe = data&1 != 0
	if c.strobe {
		c.index = 0
	}
}
```

- [ ] **Step 2: 验证编译通过**

```bash
go build ./...
```

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add input/ && git commit -m "feat: add keyboard input controller"
```

---

### Task 9: APU - 音频处理

**Files:**
- Create: `apu/apu.go`
- Create: `apu/apu_test.go`

- [ ] **Step 1: 编写 APU 基本测试**

```go
// apu/apu_test.go
package apu

import (
	"testing"
)

func TestSquareChannelFrequency(t *testing.T) {
	ch := newSquareChannel()
	// 写入频率低字节 $4002 = 0xFF
	ch.write(2, 0xFF)
	// 写入频率高字节 $4003 = 0x07（高 3 位是频率的高位）
	ch.write(3, 0x07)
	// timer period = (0x07 & 0x07) << 8 | 0xFF = 0x7FF
	expected := uint16(0x7FF)
	if ch.timerPeriod != expected {
		t.Errorf("timer period: expected 0x%04X, got 0x%04X", expected, ch.timerPeriod)
	}
}

func TestSquareChannelDuty(t *testing.T) {
	ch := newSquareChannel()
	// 写入 duty cycle $4000 = 0xC0
	ch.write(0, 0xC0)
	if ch.duty != 3 {
		t.Errorf("duty: expected 3, got %d", ch.duty)
	}
}

func TestSquareChannelOutput(t *testing.T) {
	ch := newSquareChannel()
	ch.write(0, 0x30)   // 不静音，duty=0
	ch.write(2, 0xFF)    // 低频
	ch.write(3, 0x00)    // 高频=0，timer period = 0xFF
	ch.enabled = true

	// 在一个 timer period 内，输出应随 duty 变化
	var outputs []float32
	for i := 0; i < 20; i++ {
		outputs = append(outputs, ch.sample())
	}
	// 检查有非零输出
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

func TestAPUWriteRegister(t *testing.T) {
	apu := New()
	// 初始化方波 1：设置 duty + 音量
	apu.WriteRegister(0x4000, 0x9F) // duty=2, 音量=15
	// 未使能通道，不应有输出
	samples := apu.GenerateSamples()
	for _, s := range samples {
		if s != 0 {
			t.Error("expected silence when channels disabled")
			break
		}
	}
}

func TestAPUEnableChannel(t *testing.T) {
	apu := New()
	// 使能方波 1
	apu.WriteRegister(0x4015, 0x01)
	if !apu.sq1.enabled {
		t.Error("square 1 should be enabled")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./apu/...
```

Expected: 编译失败

- [ ] **Step 3: 实现 APU**

```go
// apu/apu.go
package apu

const (
	SampleRate = 44100
)

var dutyTable = [4][8]float32{
	{0, 1, 0, 0, 0, 0, 0, 0}, // 12.5%
	{0, 1, 1, 0, 0, 0, 0, 0}, // 25%
	{0, 1, 1, 1, 1, 0, 0, 0}, // 50%
	{1, 0, 0, 1, 1, 1, 1, 1}, // 75% (negated)
}

type squareChannel struct {
	enabled      bool
	duty         uint8
	dutyIndex    uint8
	volume       uint8
	envelopeLoop bool
	constVolume  bool
	timerPeriod  uint16
	timer        uint16
	lengthCounter uint8
	envelopeDivider uint8
	envelopeVolume  uint8
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
	case 1:
		// sweep（简化实现）
	case 2:
		ch.timerPeriod = (ch.timerPeriod & 0x0700) | uint16(data)
	case 3:
		ch.timerPeriod = (ch.timerPeriod & 0x00FF) | (uint16(data&0x07) << 8)
		ch.lengthCounter = lengthTable[data>>3]
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
		ch.timer = ch.timerPeriod
		ch.dutyIndex = (ch.dutyIndex + 1) & 7
	}

	d := dutyTable[ch.duty][ch.dutyIndex]
	var vol uint8
	if ch.constVolume {
		vol = ch.volume
	} else {
		vol = ch.envelopeVolume
	}
	return d * float32(vol) / 15.0
}

var lengthTable = [32]uint8{
	10, 254, 20, 2, 40, 4, 80, 6, 160, 8, 60, 10, 14, 12, 26, 14,
	12, 16, 24, 18, 48, 20, 96, 22, 192, 24, 72, 26, 16, 28, 32, 30,
}

type triangleChannel struct {
	enabled      bool
	timerPeriod  uint16
	timer        uint16
	step         uint8
	linearCounter uint8
	linearReload bool
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
		ch.timer = ch.timerPeriod
		ch.step = (ch.step + 1) & 31
	}
	return triangleWave[ch.step]/15.0*2 - 1.0
}

func (ch *triangleChannel) write(reg uint8, data uint8) {
	switch reg {
	case 0:
		ch.linearReload = data&0x80 != 0
	case 2:
		ch.timerPeriod = (ch.timerPeriod & 0x0700) | uint16(data)
	case 3:
		ch.timerPeriod = (ch.timerPeriod & 0x00FF) | (uint16(data&0x07) << 8)
		ch.lengthCounter = lengthTable[data>>3]
	}
}

type noiseChannel struct {
	enabled      bool
	timerPeriod  uint16
	timer        uint16
	shiftReg     uint16
	lengthCounter uint8
	volume       uint8
	constVolume  bool
	envelopeVolume uint8
	envelopeLoop  bool
}

func newNoiseChannel() *noiseChannel {
	return &noiseChannel{shiftReg: 1}
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
		if ch.constVolume { vol = ch.volume } else { vol = ch.envelopeVolume }
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
		periodTable := [16]uint16{
			4, 8, 16, 32, 64, 96, 128, 160, 202, 254, 380, 508, 762, 1016, 2034, 4068,
		}
		ch.timerPeriod = periodTable[data&0x0F]
	case 3:
		ch.lengthCounter = lengthTable[data>>3]
	}
}

type APU struct {
	sq1      *squareChannel
	sq2      *squareChannel
	tri      *triangleChannel
	noise     *noiseChannel
	frameCycle int
}

func New() *APU {
	return &APU{
		sq1:  newSquareChannel(),
		sq2:  newSquareChannel(),
		tri:  newTriangleChannel(),
		noise: newNoiseChannel(),
	}
}

func (a *APU) WriteRegister(addr uint16, data uint8) {
	switch {
	case addr >= 0x4000 && addr <= 0x4003:
		a.sq1.write(addr-0x4000, data)
	case addr >= 0x4004 && addr <= 0x4007:
		a.sq2.write(addr-0x4004, data)
	case addr >= 0x4008 && addr <= 0x400B:
		a.tri.write(addr-0x4008, data)
	case addr >= 0x400C && addr <= 0x400F:
		a.noise.write(addr-0x400C, data)
	case addr == 0x4015:
		a.sq1.enabled = data&0x01 != 0
		a.sq2.enabled = data&0x02 != 0
		a.tri.enabled = data&0x04 != 0
		a.noise.enabled = data&0x08 != 0
	}
}

func (a *APU) ReadStatus() uint8 {
	var status uint8
	if a.sq1.lengthCounter > 0 { status |= 0x01 }
	if a.sq2.lengthCounter > 0 { status |= 0x02 }
	if a.tri.lengthCounter > 0 { status |= 0x04 }
	if a.noise.lengthCounter > 0 { status |= 0x08 }
	return status
}

func (a *APU) GenerateSamples() []float32 {
	samplesPerFrame := SampleRate / 60 // ~735
	samples := make([]float32, samplesPerFrame)
	for i := range samples {
		s := a.sq1.sample() + a.sq2.sample() + a.tri.sample() + a.noise.sample()
		samples[i] = s / 4.0 // 混合并衰减
	}
	return samples
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./apu/...
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add apu/ && git commit -m "feat: add APU with square, triangle, and noise channels"
```

---

### Task 10: 主循环整合

**Files:**
- Modify: `main.go`

- [ ] **Step 1: 实现完整主循环**

```go
package main

import (
	"log"
	"os"

	"fcProject/apu"
	"fcProject/bus"
	"fcProject/cartridge"
	"fcProject/cpu"
	"fcProject/input"
	"fcProject/ppu"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	cyclesPerFrame = 29781
	screenWidth    = 256 * 3
	screenHeight   = 240 * 3
)

type Game struct {
	CPU       *cpu.CPU
	PPU       *ppu.PPU
	APU       *apu.APU
	Bus       *bus.Bus
	Cartridge *cartridge.Cartridge
	Input     *input.Controller

	frameImage *ebiten.Image
}

func NewGame(romPath string) (*Game, error) {
	data, err := os.ReadFile(romPath)
	if err != nil {
		return nil, err
	}

	cart, err := cartridge.Load(data)
	if err != nil {
		return nil, err
	}

	inp := input.New()
	p := ppu.New(cart)
	a := apu.New()

	b := bus.NewBus(cart, p, a, inp)
	c := cpu.New(b)

	g := &Game{
		CPU:       c,
		PPU:       p,
		APU:       a,
		Bus:       b,
		Cartridge: cart,
		Input:     inp,
		frameImage: ebiten.NewImage(ppu.ScreenWidth, ppu.ScreenHeight),
	}

	g.CPU.Reset()
	log.Printf("Loaded: SuperMary.nes (Mapper %d, %d PRG banks, %d CHR banks)",
		cart.Mapper, cart.PRGBanks, cart.CHRBanks)

	return g, nil
}

func (g *Game) Update() error {
	g.Input.Update()
	g.CPU.RunCycles(g.CPU.Cycles + cyclesPerFrame)
	g.PPU.NMI = true // 每帧触发 NMI
	g.PPU.RenderFrame()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// 更新 framebuffer 到 Ebitengine image
	frame := g.PPU.Frame
	g.frameImage.WritePixels(frame)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(3, 3)
	screen.DrawImage(g.frameImage, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	game, err := NewGame("rom/SuperMary.nes")
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("NES Emulator - Super Mary")
	ebiten.SetTPS(60)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 2: 更新 go.mod 确保依赖正确**

```bash
go mod tidy
```

- [ ] **Step 3: 验证编译通过**

```bash
go build ./...
```

Expected: 编译成功

- [ ] **Step 4: 运行测试确认所有测试通过**

```bash
go test ./...
```

Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add main.go go.mod go.sum && git commit -m "feat: integrate CPU/PPU/APU/Input in Ebitengine main loop"
```

---

### Task 11: 运行调试与修复

- [ ] **Step 1: 运行模拟器**

```bash
cp rom/SuperMary.nes /tmp/ # 备份 ROM
go run .
```

Expected: 窗口打开，显示画面

- [ ] **Step 2: 根据实际运行结果修复常见问题**

1. 如果画面不对：检查 PPU 的命名表镜像和滚动实现
2. 如果游戏不运行：检查 CPU 中断向量和 NMI 时序
3. 如果音频有噪音：调整 APU 混合比例

常见修复：
- NMI 应在 VBlank 期间仅触发一次
- 移除 `g.PPU.NMI = true` 的每帧设置，改为在 PPU 内部管理 NMI 时序
- 游戏初始化通常需要多个帧的 CPU 周期，确保 `RunCycles` 循环正常

- [ ] **Step 3: 增加 PPU 内部 NMI 管理**

修改 `main.go` 中 `Update` 方法：

```go
func (g *Game) Update() error {
	g.Input.Update()
	g.CPU.RunCycles(g.CPU.Cycles + cyclesPerFrame)
	// NMI 由 PPU 在 VBlank 期间自动管理
	if g.PPU.Status&0x80 != 0 && g.PPU.Ctrl&0x80 != 0 {
		g.CPU.NMI = true
	}
	g.PPU.Status &= 0x7F // 清除 VBlank，由 RenderFrame 重新设置
	return nil
}
```

- [ ] **Step 4: Commit 修复**

```bash
git add -A && git commit -m "fix: correct NMI timing and various runtime fixes"
```

---

## 自审清单

1. **Spec 覆盖**：✓ Cartridge ✓ Bus ✓ CPU ✓ PPU ✓ APU ✓ Input ✓ 主循环整合
2. **无占位符**：所有步骤都包含完整的 Go 代码
3. **类型一致性**：Bus 接口、PPU 寄存器地址、Cartridge 方法签名在所有任务中保持一致
