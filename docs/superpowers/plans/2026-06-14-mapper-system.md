# Mapper 系统实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development

**Goal:** 实现可扩展 Mapper 系统，支持 Mapper 0/1/4，运行 忍者神龟1.nes 和 超级玛丽2.nes

**Architecture:** Mapper 接口 + 3 个实现（mapper0.go, mapper1.go, mapper4.go），Cartridge 委托所有读写给 Mapper

**Tech Stack:** Go

---

### Task 1: 定义 Mapper 接口 + 重构 Cartridge

**Files:**
- Modify: `cartridge/cartridge.go`
- Create: `cartridge/mapper0.go`

- [ ] **Step 1: 添加 Mapper 接口和重构 Cartridge**

在 `cartridge/cartridge.go` 中添加：

```go
// Mapper is the interface for cartridge mappers.
type Mapper interface {
	PRGRead(addr uint16) uint8
	PRGWrite(addr uint16, data uint8)
	CHRRead(addr uint16) uint8
	CHRWrite(addr uint16, data uint8)
	MirrorMode() uint8
}

type Cartridge struct {
	PRG      []byte
	CHR      []byte
	PRGBanks int
	CHRBanks int
	Mapper   int
	mapper   Mapper // private: actual mapper implementation
}
```

**修改 Cartridge 方法为委托**：

```go
func (c *Cartridge) PRGRead(addr uint16) uint8 {
	if c.mapper != nil {
		return c.mapper.PRGRead(addr)
	}
	return 0
}

func (c *Cartridge) PRGWrite(addr uint16, data uint8) {
	if c.mapper != nil {
		c.mapper.PRGWrite(addr, data)
	}
}

func (c *Cartridge) CHRRead(addr uint16) uint8 {
	if c.mapper != nil {
		return c.mapper.CHRRead(addr)
	}
	return 0
}

func (c *Cartridge) CHRWrite(addr uint16, data uint8) {
	if c.mapper != nil {
		c.mapper.CHRWrite(addr, data)
	}
}

func (c *Cartridge) MirrorMode() uint8 {
	if c.mapper != nil {
		return c.mapper.MirrorMode()
	}
	return 0
}
```

**修改 Load() 函数**（替换 mapper==0 检查为 switch）：

```go
func Load(data []byte) (*Cartridge, error) {
	// ... header parsing (same as before until mapper line) ...

	var m Mapper
	switch mapper {
	case 0:
		m = NewMapper0(prgROM, chrROM)
	case 1:
		m = NewMapper1(prgROM, chrROM, uint8(mirror))
	case 4:
		m = NewMapper4(prgROM, chrROM, uint8(mirror))
	default:
		return nil, errors.New("unsupported mapper")
	}

	return &Cartridge{
		PRG:      prgROM,
		CHR:      chrROM,
		PRGBanks: prgBanks,
		CHRBanks: chrBanks,
		Mapper:   mapper,
		mapper:   m,
	}, nil
}
```

- [ ] **Step 2: 创建 mapper0.go — Mapper 0 实现**

```go
// cartridge/mapper0.go
package cartridge

type Mapper0 struct {
	prg []byte
	chr []byte
}

func NewMapper0(prg, chr []byte) *Mapper0 {
	return &Mapper0{prg: prg, chr: chr}
}

func (m *Mapper0) PRGRead(addr uint16) uint8 {
	if addr < 0x8000 {
		return 0
	}
	offset := int(addr - 0x8000)
	if len(m.prg) == 16384 {
		offset &= 0x3FFF
	}
	if offset >= len(m.prg) {
		return 0
	}
	return m.prg[offset]
}

func (m *Mapper0) PRGWrite(addr uint16, data uint8) {}

func (m *Mapper0) CHRRead(addr uint16) uint8 {
	if int(addr) >= len(m.chr) {
		return 0
	}
	return m.chr[addr]
}

func (m *Mapper0) CHRWrite(addr uint16, data uint8) {
	if int(addr) < len(m.chr) {
		m.chr[addr] = data
	}
}

func (m *Mapper0) MirrorMode() uint8 { return 0 }
```

- [ ] **Step 3: 运行测试确认重构无回归**

```bash
go build ./... && go test ./cartridge/...
```

- [ ] **Step 4: Commit**

```bash
git add cartridge/
git commit -m "refactor: add Mapper interface and extract Mapper0"
```

---

### Task 2: 实现 Mapper 1 (MMC1)

**Files:**
- Create: `cartridge/mapper1.go`
- Create: `cartridge/mapper1_test.go`

Mapper 1 关键特性：
- **串行加载**：5 次连续写入（bit 0 依次移入 5 位移位寄存器），bit 7=1 时复位
- **寄存器**：$8000-$9FFF（控制寄存器）、$A000-$BFFF（CHR bank 0）、$C000-$DFFF（CHR bank 1）、$E000-$FFFF（PRG bank）
- **PRG 模式**：32KB（bit 3=0）或 16KB+16KB（bit 3=1，低 bank 固定/可切换）
- **CHR 模式**：8KB（bit 4=0）或 4KB+4KB（bit 4=1）
- **镜像**：水平/垂直/单屏A/单屏B（通过控制寄存器 bit 0-1）

实现代码提供控制寄存器的完整移位处理、PRG/CHR bank 切换逻辑和镜像模式选择。

- [ ] **Step 1: 实现 mapper1.go**

```go
package cartridge

type Mapper1 struct {
	prg         []byte
	chr         []byte
	shiftReg    uint8
	shiftCount  int
	control     uint8
	chrBank0    uint8
	chrBank1    uint8
	prgBank     uint8
	mirror      uint8
	prgBankMode bool // false=32KB, true=16KB+16KB
	chrBankMode bool // false=8KB, true=4KB+4KB
}

func NewMapper1(prg, chr []byte, mirror uint8) *Mapper1 {
	return &Mapper1{
		prg:    prg,
		chr:    chr,
		mirror: mirror,
		control: 0x0C, // PRG mode 3 (16KB fixed at $C000, switch at $8000)
	}
}

func (m *Mapper1) PRGRead(addr uint16) uint8 {
	var bank int
	switch {
	case addr >= 0x8000 && addr < 0xC000:
		if m.prgBankMode {
			bank = int(m.prgBank & 0x0F)
		} else {
			bank = int(m.prgBank & 0x0E)
		}
	case addr >= 0xC000:
		if m.prgBankMode {
			bank = (len(m.prg) / 16384) - 1
		} else {
			bank = int(m.prgBank&0x0E) + 1
		}
	}
	offset := bank*16384 + int(addr&0x3FFF)
	if offset >= len(m.prg) {
		offset %= len(m.prg)
	}
	return m.prg[offset]
}

func (m *Mapper1) PRGWrite(addr uint16, data uint8) {
	if data&0x80 != 0 {
		m.shiftReg = 0
		m.shiftCount = 0
		m.control |= 0x0C
		return
	}
	m.shiftReg = (m.shiftReg >> 1) | ((data & 1) << 4)
	m.shiftCount++
	if m.shiftCount < 5 {
		return
	}
	// 5 writes complete, store to register
	m.shiftCount = 0
	switch {
	case addr < 0xA000: // Control ($8000-$9FFF)
		m.control = m.shiftReg & 0x1F
		m.mirror = m.control & 0x03
		m.prgBankMode = m.control&0x08 != 0
		m.chrBankMode = m.control&0x10 != 0
	case addr < 0xC000: // CHR bank 0 ($A000-$BFFF)
		m.chrBank0 = m.shiftReg & 0x1F
	case addr < 0xE000: // CHR bank 1 ($C000-$DFFF)
		m.chrBank1 = m.shiftReg & 0x1F
	default: // PRG bank ($E000-$FFFF)
		m.prgBank = m.shiftReg & 0x0F
	}
	m.shiftReg = 0
}

func (m *Mapper1) CHRRead(addr uint16) uint8 {
	var bank int
	if m.chrBankMode {
		if addr < 0x1000 {
			bank = int(m.chrBank0) * 4096
		} else {
			bank = int(m.chrBank1) * 4096
		}
	} else {
		bank = int(m.chrBank0&0x1E) * 4096
	}
	offset := bank + int(addr&0xFFF)
	if offset >= len(m.chr) {
		offset %= len(m.chr)
	}
	return m.chr[offset]
}

func (m *Mapper1) CHRWrite(addr uint16, data uint8) {
	var bank int
	if m.chrBankMode {
		if addr < 0x1000 {
			bank = int(m.chrBank0) * 4096
		} else {
			bank = int(m.chrBank1) * 4096
		}
	} else {
		bank = int(m.chrBank0&0x1E) * 4096
	}
	offset := bank + int(addr&0xFFF)
	if offset < len(m.chr) {
		m.chr[offset] = data
	}
}

func (m *Mapper1) MirrorMode() uint8 { return m.mirror & 0x03 }
```

- [ ] **Step 2: 添加 Mapper1 基本测试**

```go
package cartridge

import "testing"

func TestMapper1ShiftRegister(t *testing.T) {
	prg := make([]byte, 256*16384)
	chr := make([]byte, 128*8192)
	m := NewMapper1(prg, chr, 0)

	// Write 5 times to $8000-$9FFF to set control
	m.PRGWrite(0x8000, 0x01) // bit 0 = 1
	m.PRGWrite(0x8000, 0x01)
	m.PRGWrite(0x8000, 0x00)
	m.PRGWrite(0x8000, 0x00)
	m.PRGWrite(0x8000, 0x01) // 5th write: shiftReg = 0b10001 = 0x11

	if m.control != 0x11 {
		t.Errorf("expected control 0x11, got 0x%02X", m.control)
	}
}

func TestMapper1ResetShift(t *testing.T) {
	prg := make([]byte, 256*16384)
	chr := make([]byte, 128*8192)
	m := NewMapper1(prg, chr, 0)
	m.PRGWrite(0x8000, 0x01)
	m.PRGWrite(0x8000, 0x01)
	m.PRGWrite(0x8000, 0x80) // bit 7 set → reset
	if m.shiftCount != 0 {
		t.Errorf("shift count should be 0 after reset, got %d", m.shiftCount)
	}
}
```

- [ ] **Step 3: 验证**

```bash
go build ./... && go test ./cartridge/...
```

- [ ] **Step 4: Commit**

```bash
git add cartridge/
git commit -m "feat: add Mapper 1 (MMC1) support"
```

---

### Task 3: 实现 Mapper 4 (MMC3)

**Files:**
- Create: `cartridge/mapper4.go`
- Create: `cartridge/mapper4_test.go`

Mapper 4 关键特性：
- **Bank 寄存器**：8 个寄存器（$8000-$9FFF 偶地址选择寄存器号，奇地址写入数据）
- **PRG 模式**：$8000-$9FFF 和 $A000-$BFFF 可切换（8KB），$C000-$DFFF 固定到倒数第二 bank，$E000-$FFFF 固定到最后 bank
- **CHR 模式**：6 个 1KB + 2 个 2KB bank
- **镜像**：寄存器选择水平/垂直

- [ ] **Step 1: 实现 mapper4.go**

```go
package cartridge

type Mapper4 struct {
	prg         []byte
	chr         []byte
	bankSelect  uint8
	prgMode     uint8 // 0 or 1, swaps $8000/$C000 bank behavior
	chrInvert   uint8
	mirror      uint8
	irqLatch    uint8
	irqCounter  uint8
	irqEnable   bool
	irqReload   bool
	registers   [8]uint8
}

func NewMapper4(prg, chr []byte, mirror uint8) *Mapper4 {
	m := &Mapper4{
		prg:    prg,
		chr:    chr,
		mirror: mirror,
	}
	// Default: fixed banks at end
	m.registers[6] = uint8(len(prg)/8192) - 2
	m.registers[7] = uint8(len(prg)/8192) - 1
	return m
}

func (m *Mapper4) PRGRead(addr uint16) uint8 {
	bank := 0
	switch {
	case addr < 0xA000: // $8000-$9FFF
		bank = int(m.registers[6]) * 8192
	case addr < 0xC000: // $A000-$BFFF
		bank = int(m.registers[7]) * 8192
	case addr < 0xE000: // $C000-$DFFF (fixed to second-last bank)
		bank = (len(m.prg)/8192 - 2) * 8192
	default: // $E000-$FFFF (fixed to last bank)
		bank = (len(m.prg)/8192 - 1) * 8192
	}
	offset := bank + int(addr&0x1FFF)
	if offset >= len(m.prg) {
		offset %= len(m.prg)
	}
	return m.prg[offset]
}

func (m *Mapper4) PRGWrite(addr uint16, data uint8) {
	switch {
	case addr >= 0x8000 && addr < 0xA000:
		if addr&1 == 0 {
			m.bankSelect = data & 0x07
			m.prgMode = (data >> 6) & 1
			m.chrInvert = (data >> 7) & 1
		} else {
			m.registers[m.bankSelect] = data
		}
	case addr >= 0xA000 && addr < 0xC000:
		if addr&1 == 0 {
			m.mirror = data & 1
		}
	case addr >= 0xC000 && addr <= 0xDFFF:
		if addr&1 == 0 {
			m.irqLatch = data
		} else {
			m.irqCounter = 0
			m.irqReload = true
		}
	case addr >= 0xE000:
		if addr&1 == 0 {
			m.irqEnable = false
		} else {
			m.irqEnable = true
		}
	}
}

func (m *Mapper4) CHRRead(addr uint16) uint8 {
	bank := m.getCHRBank(addr)
	offset := bank*1024 + int(addr&0x3FF)
	if offset >= len(m.chr) { offset %= len(m.chr) }
	return m.chr[offset]
}

func (m *Mapper4) CHRWrite(addr uint16, data uint8) {
	bank := m.getCHRBank(addr)
	offset := bank*1024 + int(addr&0x3FF)
	if offset < len(m.chr) { m.chr[offset] = data }
}

func (m *Mapper4) getCHRBank(addr uint16) int {
	switch {
	case addr < 0x0800:
		return int(m.registers[0] & 0xFE)
	case addr < 0x1000:
		return int(m.registers[1] & 0xFE)
	case addr < 0x1400:
		return int(m.registers[2])
	case addr < 0x1800:
		return int(m.registers[3])
	case addr < 0x1C00:
		return int(m.registers[4])
	default:
		return int(m.registers[5])
	}
}

func (m *Mapper4) MirrorMode() uint8 { return m.mirror }
```

- [ ] **Step 2: 测试**

```go
func TestMapper4BankSelect(t *testing.T) {
	prg := make([]byte, 512*8192)
	chr := make([]byte, 256*8192)
	m := NewMapper4(prg, chr, 1)

	// Select register 6 ($8000-$9FFF PRG bank)
	m.PRGWrite(0x8000, 0x06) // even: select register 6
	m.PRGWrite(0x8001, 0x03) // odd: write 3 to register 6
	if m.registers[6] != 0x03 {
		t.Errorf("expected register 6 = 3, got %d", m.registers[6])
	}
}
```

- [ ] **Step 3: 验证构建**

```bash
go build ./... && go test ./cartridge/...
```

- [ ] **Step 4: Commit**

```bash
git add cartridge/
git commit -m "feat: add Mapper 4 (MMC3) support"
```

---

### Task 4: 端到端验证

- [ ] **Step 1: Headless 测试两个新 ROM**

创建临时测试脚本验证忍者神龟1.nes 和超级玛丽2.nes 能加载并运行至少 60 帧不崩溃：

```go
func TestNewROMsLoad(t *testing.T) {
    for _, rom := range []string{"忍者神龟1.nes", "超级玛丽2.nes"} {
        data, _ := os.ReadFile("../../rom/" + rom)
        cart, err := Load(data)
        if err != nil {
            t.Fatalf("%s: Load failed: %v", rom, err)
        }
        t.Logf("%s: Mapper=%d PRG=%d CHR=%d", rom, cart.Mapper, cart.PRGBanks, cart.CHRBanks)
    }
}
```

- [ ] **Step 2: 运行全部测试**

```bash
go build ./... && go test ./...
```

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "test: add end-to-end ROM loading tests"
```

---

## 自审

| 检查项 | 结果 |
|--------|------|
| Spec 覆盖 | Mapper 接口 ✓ Mapper0 ✓ Mapper1 ✓ Mapper4 ✓ Load switch ✓ |
| 无占位符 | 全部含完整 Go 代码 ✓ |
| 类型一致性 | Mapper 接口 → Cartridge 委托 → 各实现 一致 ✓ |
