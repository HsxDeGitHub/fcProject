# NES 模拟器设计规格

## 概述

开发一个能够运行 NES ROM 文件的模拟器，目标 ROM 为 `SuperMary.nes`（Mapper 0 / NROM）。

**技术栈**：Go + Ebitengine
**精度目标**：指令级模拟，能跑就行
**功能范围**：CPU / PPU / APU / Input / Mapper 0 全实现

---

## 整体架构

模拟器核心是一个共享地址总线。CPU 通过总线读写内存，PPU 和 APU 各有独立地址空间，与 CPU 通过 MMIO 交互。

```
┌──────────────────────────────────────────┐
│                 main.go                   │
│           Ebitengine 主循环                │
│   ┌─────┐  ┌─────┐  ┌─────┐  ┌───────┐  │
│   │ CPU │  │ PPU │  │ APU │  │ Input │  │
│   └──┬──┘  └──┬──┘  └──┬──┘  └───┬───┘  │
│      └────────┼────────┼──────────┘      │
│           ┌───┴────┐    │                │
│           │  Bus   │    │                │
│           └───┬────┘    │                │
│           ┌───┴────┐    │                │
│           │Cartridge│   │                │
│           └────────┘    │                │
└──────────────────────────────────────────┘
```

### 包结构

```
fcProject/
├── main.go              # 入口，Ebitengine 主循环
├── cpu/
│   └── cpu.go           # 6502 CPU 核心
├── ppu/
│   └── ppu.go           # 图像处理单元
├── apu/
│   └── apu.go           # 音频处理单元
├── cartridge/
│   └── cartridge.go     # ROM 加载 + Mapper
├── bus/
│   └── bus.go           # 地址总线
└── input/
    └── input.go         # 键盘输入
```

### 组件通信

- CPU ↔ Bus ↔ Cartridge（PRG ROM、RAM）
- CPU ↔ PPU（MMIO：$2000-$2007、$4014）
- CPU ↔ APU（MMIO：$4000-$4017）
- CPU ↔ Input（MMIO：$4016-$4017）
- PPU ↔ Cartridge（CHR ROM，名称表）

### 时钟策略

NES 中 1 帧 = 262 扫描线 × 341 PPU 周期 ≈ 89342 PPU 周期，1 PPU 周期 = 3 CPU 周期（NTSC）。
简化实现：每帧执行约 29781 个 CPU 周期，PPU 在帧末一次性渲染，APU 每帧生成 1/60s 采样。

### 内存映射

| 地址范围 | 大小 | 映射目标 |
|----------|------|----------|
| $0000-$07FF | 2KB | CPU RAM |
| $0800-$1FFF | 镜像 | CPU RAM 镜像 |
| $2000-$2007 | 8B | PPU 寄存器 |
| $2008-$3FFF | 镜像 | PPU 寄存器镜像 |
| $4000-$4017 | 24B | APU + Input 寄存器 |
| $4020-$5FFF | - | 扩展 ROM（Mapper 0 不使用） |
| $6000-$7FFF | 8KB | PRG RAM（如有） |
| $8000-$FFFF | 32KB | PRG ROM |

---

## 各组件设计

### CPU（6502 核心）

实现所有 151 条官方指令，不含非官方指令。

**寄存器**：A、X、Y、SP（8 位）、PC（16 位）、P（状态标志）

**寻址模式（13 种）**：
Implied、Accumulator、Immediate、Zero Page、Zero Page,X、Zero Page,Y、
Absolute、Absolute,X、Absolute,Y、Indirect、(Indirect,X)、(Indirect),Y、Relative

**中断**：NMI（$FFFA）、RESET（$FFFC）、IRQ/BRK（$FFFE）

**关键指令**：所有 load/store、算术运算（ADC/SBC）、位操作（AND/ORA/EOR）、
移位（ASL/LSR/ROL/ROR）、分支（BCC/BCS/BEQ/BNE 等）、栈操作（PHA/PLA/PHL/PLP）、
跳转子程序（JSR/RTS）、中断返回（RTI）

### PPU（图像处理单元）

**渲染模式**：逐扫描线渲染，262 行 × 256 像素，V-Blank 在第 241-261 行。

**背景渲染**：名称表（Name Table）索引图案表（Pattern Table）获取 tile 数据，
属性表（Attribute Table）获取颜色组。每个 tile 8×8 像素。

**精灵渲染**：OAM（64 字节，最多 64 个精灵），支持 8×8 和 8×16 尺寸。

**调色板**：NES 64 色调色板，背景 4 组 × 3 色 + 背景色，精灵 4 组 × 3 色 + 透明。

**寄存器**：PPUCTRL($2000)、PPUMASK($2001)、PPUSTATUS($2002)、OAMADDR($2003)、
OAMDATA($2004)、PPUSCROLL($2005)、PPUADDR($2006)、PPUDATA($2007)

**OAM DMA**：$4014 写入触发，256 字节从 CPU 内存复制到 OAM。

### APU（音频处理单元）

5 个通道：方波 1、方波 2、三角波、噪声、DMC。

每帧生成 1/60s 的 PCM 采样（44100Hz → 约 735 个采样点/帧）。
所有采样值归一化到 [-1.0, 1.0] 的 float32。

**寄存器**：
- $4000-$4003：方波 1
- $4004-$4007：方波 2
- $4008-$400B：三角波
- $400C-$400F：噪声
- $4010-$4013：DMC
- $4015：通道使能/状态
- $4017：帧计数器

### Cartridge

**iNES 文件头解析**（16 字节）：
- 字节 0-3：魔术字 "NES\x1A"
- 字节 4：PRG ROM 大小（16KB 单位）
- 字节 5：CHR ROM 大小（8KB 单位）
- 字节 6：Mapper 编号 + 镜像模式

**Mapper 0（NROM）**：
- PRG ROM 直接映射到 $8000-$FFFF（16KB 时在 $C000 处镜像）
- CHR ROM 直接映射到 PPU $0000-$1FFF
- 无 Bank Switching

### Bus（地址总线）

接口定义：

```go
type Bus interface {
    Read(addr uint16) uint8
    Write(addr uint16, data uint8)
}
```

根据地址范围路由到对应组件，处理 RAM 镜像（$0800-$1FFF → $0000-$07FF）。

### Input

键盘映射：
- 方向键 ↑↓←→ → D-Pad
- Z → A，X → B
- Enter → Start，右 Shift → Select

通过 $4016 读取控制器状态（8 次移位读出每个按键），通过 Ebitengine 的 `ebiten.IsKeyPressed` 实时检测。

---

## 帧循环

```
每个 Ebitengine 帧（1/60s）：

1. 读取键盘状态
2. CPU 执行指令直到累计 ≥ 29781 个 CPU 周期
3. PPU 渲染完整一帧（256×240 像素）
4. APU 生成本帧音频采样（~735 个 float32）
5. Ebitengine 绘制像素缓冲到窗口（3x 缩放：768×720）
6. Ebitengine 播放音频采样
```

### 画面缩放

256×240 像素以 3x 整数倍缩放显示 → 768×720 窗口。

---

## 测试策略

### 单元测试
- CPU：每条指令独立测试，覆盖所有寻址模式组合
- PPU：寄存器读写、OAM DMA、扫描线渲染单步验证
- APU：各通道频率/波形输出验证
- Cartridge：iNES 文件头解析、Mapper 0 地址映射

### 集成测试
- CPU + Bus + Cartridge 联动，加载 ROM 验证前 N 条指令
- 使用 **nestest.nes**（官方 CPU 测试 ROM）的运行日志逐行对比

### 验收标准
- `SuperMary.nes` 能加载并显示标题画面
- 键盘输入可控制游戏
- 音频正常播放
