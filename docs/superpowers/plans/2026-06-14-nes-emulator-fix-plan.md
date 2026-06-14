# NES 模拟器全面测试与修复计划

## Context

模拟器已能运行但画面混乱。通过代码探索发现了 8 个关键 bug 和大量测试空白。本计划分 4 个阶段，从底层 CPU 修复开始，逐步向上到 PPU 渲染修复。

## 阶段 1：CPU 关键 bug 修复 + 集成测试

### Task 1.1：修复 INC/DEC abs 变体中多余的 `c.PC++`

**文件**：`cpu/cpu.go`

**Bug**：`0xEE`（INC abs）、`0xFE`（INC abs,X）、`0xCE`（DEC abs）、`0xDE`（DEC abs,X）这四条指令在执行 `c.abs()`/`c.absX()` 之前错误地写了 `c.PC++`。`abs()` 内部会自己从 PC 读取并推进 PC，多余的 `c.PC++` 导致读错操作数地址。

**修复**：删除这四条指令 case 中的 `c.PC++`：
- `case 0xEE:` 删除 `c.PC++`（`abs()` 会自己读 PC）
- `case 0xFE:` 删除 `c.PC++`（`absX()` 调用 `abs()` 读 PC）
- `case 0xCE:` 删除 `c.PC++`（同上）
- `case 0xDE:` 删除 `c.PC++`（同上）

**测试**：添加测试验证 INC abs 写入正确地址：
```go
func TestINCAbsolute(t *testing.T) {
    cpu, bus := newCPU()
    bus.ram[0x8000] = 0xEE // INC abs
    bus.ram[0x8001] = 0x00  // low byte
    bus.ram[0x8002] = 0x03  // high byte → $0300
    bus.ram[0x0300] = 0x41  // initial value
    cpu.Step()
    if bus.ram[0x0300] != 0x42 {
        t.Errorf("INC abs: expected 0x42, got 0x%02X", bus.ram[0x0300])
    }
    if cpu.PC != 0x8003 {
        t.Errorf("INC abs: PC expected 0x8003, got 0x%04X", cpu.PC)
    }
}
```

### Task 1.2：添加缺失的 SED 操作码（0xF8）

**文件**：`cpu/cpu.go` execute switch

**修复**：在 execute() 的 Flag 操作部分添加：
```go
case 0xF8: // SED - Set Decimal Flag
    c.P |= FlagD
    return 2
```

### Task 1.3：nestest.nes 集成测试

**文件**：新建 `cpu/nestest_test.go`

nestest.nes 是 CPU 正确性的金标准。测试方法：
1. 将 nestest.nes 和 nestest.log 放入 `cpu/testdata/`
2. 加载 ROM，手动设置 PC = 0xC000
3. 循环调用 `cpu.Step()`，每步后记录 CPU 状态日志
4. 与 nestest.log 逐行对比
5. 运行约 8991 条指令后停止

```go
func TestNestest(t *testing.T) {
    data, err := os.ReadFile("testdata/nestest.nes")
    require.NoError(t, err)
    cart, err := cartridge.Load(data)
    require.NoError(t, err)
    
    bus := &testBus{cart: cart, ram: make([]byte, 0x800)}
    cpu := New(bus)
    cpu.PC = 0xC000
    
    expectedLog, _ := os.ReadFile("testdata/nestest.log")
    expectedLines := strings.Split(string(expectedLog), "\n")
    
    for i := 0; i < 8991; i++ {
        cpu.Step()
        actualLine := formatCPUState(i, cpu)
        if actualLine != expectedLines[i] {
            t.Errorf("line %d mismatch:\n  got: %s\n  want: %s", i, actualLine, expectedLines[i])
            break
        }
    }
}
```

**验证**：`go test ./cpu/... -run TestNestest`

---

## 阶段 2：PPU 寄存器交互 bug 修复

### Task 2.1：修复共享锁存器（$2005/$2006）

**文件**：`ppu/ppu.go`

**Bug**：$2005（PPUSCROLL）和 $2006（PPUADDR）在真实 NES 硬件上**共享同一个**双写锁存器。当前代码用了两个独立字段 `scrollLatch` 和 `W`。写入一个会影响另一个的后续写入状态。

**修复**：
1. 删除 `scrollLatch` 和 `W`，替换为单个 `latch bool`
2. $2005 和 $2006 的写入都使用 `p.latch`：
   - `!p.latch` → 第一次写入（$2005: X scroll、$2006: 高字节）
   - `p.latch` → 第二次写入（$2005: Y scroll、$2006: 低字节 → V=T）
3. 两个寄存器写入后都执行 `p.latch = !p.latch`
4. $2002 读取后 `p.latch = false`

**测试**：添加测试验证写入顺序 $2005 → $2006 共享锁存器：
```go
func TestSharedLatch(t *testing.T) {
    ppu := newPPU()
    // 第一次写入 $2005 → latch flips
    ppu.WriteRegister(0x2005, 0x42)
    if !ppu.Latch {  // 导出 latch
        t.Error("latch should be true after first $2005 write")
    }
    // 写入 $2006 → 应该被当作第二次写入（因为 latch = true）
    ppu.WriteRegister(0x2006, 0x23)
    // latch 应该又翻回 false
    if ppu.Latch {
        t.Error("latch should be false after $2006 write")
    }
    // 验证 V 被设置（因为第二次 $2006 写入时 latch=true，V=T）
    if ppu.V != 0x2300 { // T 高字节是 0x23，低字节还未设置
        t.Errorf("V expected 0x2300, got 0x%04X", ppu.V)
    }
}
```

### Task 2.2：修复镜像模式来源

**文件**：`ppu/ppu.go`、`cartridge/cartridge.go`

**Bug**：`mirrorNameTable` 使用 `Ctrl&0x01` 判断镜像模式，但真实硬件上镜像模式由卡带硬件决定（iNES header flags6 bit 0）。

**修复**：
1. 在 `CartridgeInterface` 中添加 `MirrorMode() uint8` 方法（或直接使用已有的 `Mirror` 字段）
2. 在 `PPU` 中存储镜像模式：初始化时从 Cartridge 读取
3. `mirrorNameTable` 中使用存储的镜像模式，而非 `Ctrl&0x01`
4. 注意：`Ctrl&0x01` 仍需用于选择基础命名表（但这是渲染时而非镜像时的工作）

具体方案：在 PPU 中添加 `mirrorMode uint8` 字段，初始化时设置：
```go
func New(cart CartridgeInterface) *PPU {
    return &PPU{
        Cart:       cart,
        mirrorMode: cart.Mirror(), // 0=水平, 1=垂直
    }
}
```

---

## 阶段 3：PPU 渲染 bug 修复

### Task 3.1：修复精灵背景优先级

**文件**：`ppu/ppu.go`

**Bug**：精灵的"背景后"优先级检查（`attr&0x20`）通过检查 `Frame[offset]` 的 RGBA 值是否为 0 来判断背景是否"透明"。但所有背景像素都有非零 RGBA 值（包括背景色索引 0 对应 NESPalette[0] = 0x7C7C7C），导致所有"背景后"精灵都被错误隐藏。

**修复**：
1. 添加 `tileBuffer [256*240]uint8` 字段存储每像素的原始颜色索引
2. `renderBackground` 中：写入 RGBA 之前，先存储 `tileBuffer[y*256+x] = colorIdx`
3. `renderSprites` 中：检查优先级时用 `tileBuffer[pos] != 0` 代替 `Frame[offset] != 0`

### Task 3.2：OAM Y=0 精灵隐藏

**文件**：`ppu/ppu.go` renderSprites

**修复**：在 Y 坐标 +1 调整之前，检查 `p.OAM[i*4] == 0`，若为 0 则跳过该精灵。

### Task 3.3：修复 mirrorNameTable $3000-$3EFF 范围

**文件**：`ppu/ppu.go` mirrorNameTable

**修复**：将 `addr = addr & 0x2FFF` 改为正确处理 $3000-$3EFF：
```go
func (p *PPU) mirrorNameTable(addr uint16) uint16 {
    addr &= 0x3FFF
    if addr >= 0x3000 && addr < 0x3F00 {
        addr -= 0x1000 // $3000-$3EFF → $2000-$2EFF
    }
    table := (addr - 0x2000) / 0x400
    // ... mirroring logic
}
```

### Task 3.4：移除强制测试调色板

**文件**：`main.go`

删除 NewGame 中的强制调色板写入代码（之前是诊断用的临时方案）。

---

## 阶段 4：全面测试验证

### Task 4.1：nestest 集成测试（阶段 1 完成）

### Task 4.2：PPU 渲染基础测试

**文件**：`ppu/ppu_test.go`

添加测试：创建一个已知图案表/名称表/调色板，渲染一帧，验证特定像素颜色。

### Task 4.3：端到端验证

运行无头诊断 60 帧，验证：
- 调色板写入数 > 0（PalWrites > 0）
- 非零像素数 > 0（Nz > 0）
- 帧缓冲中有多种颜色（不仅是背景色）

---

## 验证步骤

每阶段完成后：
1. `go build ./...` — 确保编译通过
2. `go test ./...` — 确保测试通过
3. `go run .` — 用 Ebitengine 运行 SuperMary.nes，确认画面改善
4. `git commit` — 提交该阶段的修改

## 关键文件清单

| 文件 | 阶段 | 操作 |
|------|------|------|
| `cpu/cpu.go` | 1 | 修复 PC++ bug + SED |
| `cpu/cpu_test.go` | 1 | 添加 INC abs 测试 |
| `cpu/nestest_test.go` | 1 | 新建 nestest 集成测试 |
| `cpu/testdata/nestest.nes` | 1 | 新建测试 ROM |
| `cpu/testdata/nestest.log` | 1 | 新建已知正确日志 |
| `ppu/ppu.go` | 2, 3 | 锁存器、镜像、渲染修复 |
| `ppu/ppu_test.go` | 2, 4 | 添加锁存器和渲染测试 |
| `cartridge/cartridge.go` | 2 | 添加 Mirror() 方法 |
| `main.go` | 3 | 移除强制测试调色板 |
