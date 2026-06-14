# NES 模拟器 Mapper 系统设计

## 概述

当前仅支持 Mapper 0 (NROM)。为运行 忍者神龟1.nes (Mapper 1/MMC1) 和 超级玛丽2.nes (Mapper 4/MMC3)，需要可扩展的 mapper 系统。

## 架构

`cartridge/` 包重构为接口 + 多实现模式：

```
cartridge/
├── cartridge.go    # Mapper 接口 + Cartridge 结构 + Load
├── mapper0.go      # Mapper 0 (NROM) — 已有逻辑
├── mapper1.go      # Mapper 1 (MMC1)
└── mapper4.go      # Mapper 4 (MMC3)
```

### Mapper 接口

```go
type Mapper interface {
    PRGRead(addr uint16) uint8
    PRGWrite(addr uint16, data uint8)
    CHRRead(addr uint16) uint8
    CHRWrite(addr uint16, data uint8)
    MirrorMode() uint8
}
```

Cartridge 结构体持有 Mapper 实例，所有 PRG/CHR 读写委托给 Mapper。

### Mapper 1 (MMC1)

- **PRG 模式**：16KB 固定 + 16KB 切换，或 32KB 整体切换
- **CHR 模式**：4KB 或 8KB 切换
- **镜像控制**：水平/垂直/单屏
- **串行端口**：5 次写入（bit 0 依次移入），bit 7=1 复位移位寄存器

### Mapper 4 (MMC3)

- **PRG 模式**：$8000-$9FFF 和 $A000-$BFFF 可切换（8KB），$C000-$FFFF 固定到倒数第二个 bank，$E000-$FFFF 固定到最后一个 bank
- **CHR 切换**：2KB + 4KB bank
- **IRQ 计数器**：扫描线计数器，用于 mid-frame 效果
- **镜像控制**：寄存器选择

### Cartridge 改动

- 移除 mapper==0 检查，改为 switch 创建对应 Mapper
- Cartridge 的 PRGRead/PRGWrite/CHRRead/CHRWrite/MirrorMode 全部委托给 Mapper

### Load 函数

```go
func Load(data []byte) (*Cartridge, error) {
    // 解析 header（不变）
    var m Mapper
    switch mapper {
    case 0: m = NewMapper0(prgROM, chrROM)
    case 1: m = NewMapper1(prgROM, chrROM, mirror)
    case 4: m = NewMapper4(prgROM, chrROM, mirror)
    default: return nil, errors.New("unsupported mapper")
    }
    return &Cartridge{mapper: m, PRG: prgROM, CHR: chrROM, Mapper: mapper}, nil
}
```

## 测试

- Cartridge 现有 13 项测试继续通过
- 添加 Mapper0 单元测试（验证重构后行为不变）
- 添加 Mapper1 寄存器写入 + PRG/CHR bank 切换测试
- 添加 Mapper4 寄存器写入 + bank 切换测试
- 端到端：忍者神龟1.nes、超级玛丽2.nes 能加载并运行
