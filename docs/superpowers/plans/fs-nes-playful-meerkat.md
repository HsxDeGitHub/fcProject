# NES 模拟器实施计划

## Context

用户希望构建一个能够在浏览器中运行的 NES/FC 模拟器，使用 Go 编写核心模拟逻辑，编译为 WebAssembly，通过 JavaScript/HTML 在浏览器中渲染画面、播放音频和处理输入。目标是分阶段实现，第一步是成功运行 `rom/SuperMary.nes`（Mapper 0 卡带）。

## 技术栈

- **Go 1.21+** — 核心模拟引擎，编译为 WASM
- **syscall/js** — Go WASM 与 JavaScript 桥接
- **HTML5 Canvas** — 画面渲染（256x240 原始分辨率）
- **Web Audio API** — 音频输出
- **Keyboard API** — 手柄输入

## 项目结构

```
fsProject/
  cmd/
    wasm/
      main.go              # WASM 入口，暴露 JS 接口
  internal/
    cpu/
      cpu.go                # 6502 CPU 模拟
      instructions.go        # 指令实现
      addressing.go          # 寻址模式
      interrupts.go          # 中断处理
    ppu/
      ppu.go                # PPU 主逻辑（扫描线、寄存器）
      rendering.go           # 背景/精灵逐像素渲染
      palette.go             # NES 调色板
    apu/
      apu.go                # APU 音频生成
      channels.go            # 5 个音频通道
    memory/
      memory.go             # 统一地址空间映射
      dma.go                # OAM DMA
    cartridge/
      cartridge.go          # iNES 解析、Mapper 接口
      mapper0.go            # Mapper 0 (NROM)
    input/
      joypad.go             # 手柄状态
  web/
    index.html              # 主页面
    wasm_exec.js            # Go WASM 胶水脚本
    emulator.js             # JS 胶水层（Canvas 渲染、音频回调）
    style.css               # 样式
  rom/
    SuperMary.nes            # 测试 ROM
  go.mod
  Makefile                   # 构建脚本
```

## 架构设计

```
┌─────────────────────────────────────────────────┐
│                    浏览器                        │
│                                                 │
│  ┌──────────────┐    ┌──────────────────────┐   │
│  │  HTML/CSS/JS  │    │    Go → WASM          │   │
│  │              │    │                       │   │
│  │  Canvas 渲染  │◄───│ PPU 帧缓冲 (61440 B)  │   │
│  │  AudioWorklet│◄───│ APU 采样缓冲           │   │
│  │  键盘事件     │───►│ Joypad 状态            │   │
│  │  ROM 上传     │───►│ Cartridge 加载         │   │
│  └──────────────┘    └──────────────────────┘   │
└─────────────────────────────────────────────────┘

数据流：
1. 主循环每帧 (≈16.67ms)：CPU 执行 → PPU 渲染 → APU 生成 → 输出帧缓冲
2. JS 层通过 requestAnimationFrame 驱动主循环
3. 每帧完成后 JS 读取帧缓冲渲染到 Canvas
4. AudioWorklet 主动从 APU 拉取采样数据
```

## 实施步骤

### 第 1 步：项目初始化
- 初始化 Go module (`go mod init`)
- 创建目录结构
- 创建 `Makefile`（wasm 编译、本地开发服务器）
- 创建基础 `index.html` + `emulator.js`

### 第 2 步：iNES ROM 解析 & Mapper 0
**文件**: `internal/cartridge/cartridge.go`, `internal/cartridge/mapper0.go`

- 解析 iNES 1.0 文件头（16 字节）
- 提取 PRG ROM、CHR ROM
- 识别 Mapper 编号、镜像模式
- 实现 Mapper 0（无 bank switching，直接映射）
- PRG ROM 映射到 $8000-$FFFF（若为 16KB 则镜像两次）
- CHR ROM 映射到 PPU $0000-$1FFF

### 第 3 步：CPU 6502 模拟
**文件**: `internal/cpu/cpu.go`, `internal/cpu/instructions.go`, `internal/cpu/addressing.go`

- 实现全部 56 条合法指令（官方指令）
- 13 种寻址模式（Implied, Accumulator, Immediate, Zero Page, Zero Page X/Y, Absolute, Absolute X/Y, Indirect, Indirect X, Indirect Y, Relative）
- 6 个寄存器：A, X, Y, SP, PC, P (Status)
- 状态标志：C, Z, I, D, B, V, N
- NMI、IRQ、RESET 中断处理
- **每条指令精确模拟所需时钟周期数**
- CPU 地址空间映射：
  - $0000-$07FF: RAM (2KB, 镜像 4 次到 $1FFF)
  - $2000-$2007: PPU 寄存器 (镜像到 $3FFF)
  - $4000-$4017: APU + Joypad 寄存器
  - $4020-$FFFF: Cartridge (PRG ROM)

### 第 4 步：PPU 图形处理
**文件**: `internal/ppu/ppu.go`, `internal/ppu/rendering.go`, `internal/ppu/palette.go`

- 逐扫描线渲染（共 262 条扫描线，240 条可见）
- 每个扫描线 341 个 PPU 周期
- 背景渲染：
  - 2 个 Name Table（各 960 bytes，32x30 tiles）
  - Pattern Table（tile 数据，8x8 像素）
  - 属性表（每 2x2 tile 组使用 2-bit 调色板选择）
  - 8 像素移位寄存器（Pattern 低位 + 高位）
- 精灵渲染（OAM）：
  - 64 个精灵，每个 4 bytes（Y, Tile, Attr, X）
  - Sprite 0 Hit 检测
  - 每扫描线最多 8 个精灵
- NES 调色板（64 种颜色，含 $0D 黑色洞）
- PPU 寄存器：
  - $2000 PPUCTRL, $2001 PPUMASK, $2002 PPUSTATUS
  - $2003 OAMADDR, $2004 OAMDATA, $2005 PPUSCROLL, $2006 PPUADDR, $2007 PPUDATA
- 输出帧缓冲：256x240 RGBA，供 JS 层渲染

### 第 5 步：输入系统
**文件**: `internal/input/joypad.go`

- 8 个按键：A, B, Select, Start, Up, Down, Left, Right
- 按钮状态通过移位寄存器读取（$4016/$4017）
- JS 键盘事件映射到手柄按键

### 第 6 步：WASM 集成
**文件**: `cmd/wasm/main.go`, `web/emulator.js`

- Go WASM 导出函数：
  - `loadROM(data []byte)` — 加载 ROM
  - `reset()` — 复位模拟器
  - `step()` — 执行一帧
  - `getFrameBuffer()` — 获取帧缓冲指针
  - `setKeyState(player, key, pressed)` — 设置按键状态
- JS 胶水层：
  - 文件拖拽/选择加载 ROM
  - requestAnimationFrame 驱动主循环
  - Canvas 2D 渲染（ImageData 方式）
  - 键盘事件 → Joypad 状态
  - AudioWorklet 音频管线（第二步实现）

### 第 7 步：音频（APU）— 第二步
**文件**: `internal/apu/apu.go`, `internal/apu/channels.go`

- 待 CPU/PPU 核心稳定后再实现
- 5 个通道：Pulse 1, Pulse 2, Triangle, Noise, DMC
- 44.1kHz 采样率，推送到音频缓冲区

## CPU 指令实现参考

所有指令需按 6502 手册精确实现。核心指令分类：

| 类别 | 指令 | 数量 |
|------|------|------|
| 加载/存储 | LDA, LDX, LDY, STA, STX, STY | 6 |
| 寄存器传输 | TAX, TXA, TAY, TYA, TSX, TXS | 6 |
| 算术 | ADC, SBC, INC, INX, INY, DEC, DEX, DEY | 8 |
| 逻辑 | AND, ORA, EOR, BIT | 4 |
| 移位 | ASL, LSR, ROL, ROR | 4 |
| 跳转 | JMP, JSR, RTS, RTI | 4 |
| 分支 | BCC, BCS, BEQ, BNE, BMI, BPL, BVC, BVS | 8 |
| 标志 | CLC, SEC, CLD, SED, CLI, SEI, CLV | 7 |
| 栈 | PHA, PHP, PLA, PLP | 4 |
| 比较 | CMP, CPX, CPY | 3 |
| 其他 | NOP, BRK | 2 |

## 验证方案

1. **单元测试**：每个 CPU 指令独立测试，验证寄存器状态和周期数是否符合预期（使用 Nestest ROM 验证）
2. **集成测试**：加载 SuperMary.nes，验证能否通过 Logo 画面
3. **手动验证**：
   - `make serve` 启动本地服务器
   - 浏览器打开页面，拖入 ROM 文件
   - 确认画面渲染正常、按键响应正常
4. **Nestest ROM**：运行社区标准的 Nestest.nes 自动化测试 ROM，验证 CPU 指令正确性

## 构建命令

```makefile
build:
	GOOS=js GOARCH=wasm go build -o web/main.wasm ./cmd/wasm

serve:
	cd web && python3 -m http.server 8080
```
