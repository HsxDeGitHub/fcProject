# fcProject — Go 语言 NES 模拟器

一个使用 Go + [Ebitengine](https://ebitengine.org/) 编写的 NES/Famicom 模拟器，支持多种 Mapper 和中文游戏。

## 特性

- **6502 CPU 完整实现** — 151 个官方指令 + 71 个非官方指令，通过 nestest 全部验证（8991/8991）
- **PPU 图形渲染** — 背景 + 精灵渲染，支持 8×16 精灵、水平/垂直镜像、滚动分割、精灵 0 碰撞
- **APU 音频** — 方波 1/2、三角波、噪声通道，44100Hz PCM 输出
- **Mapper 系统** — 可扩展的 Mapper 接口，当前支持 Mapper 0/1/4
- **游戏选择菜单** — 中文字体渲染，键盘导航
- **手柄模拟** — 键盘映射，支持连发

## 支持的 Mapper

| Mapper | 类型 | 示例游戏 |
|--------|------|---------|
| Mapper 0 | NROM | 超级玛丽 |
| Mapper 1 | MMC1 | 忍者神龟 |
| Mapper 4 | MMC3 | 超级玛丽 2 |

## 运行

### 依赖

- Go 1.21+
- macOS / Windows / Linux

Ebitengine 在 Linux 上需要额外的系统依赖，详见 [官方文档](https://ebitengine.org/en/documents/install.html)。

### 启动

```bash
go run .
```

启动后进入游戏选择菜单，用方向键选择游戏，回车确认。

## 键位映射

| NES 按键 | 键盘 |
|----------|------|
| A | Z |
| B | X |
| Start | Enter / Space |
| Select | Shift |
| 方向键 | 方向键 |

## 项目结构

```
.
├── main.go          # 入口 + Ebitengine 主循环 + 菜单/游戏状态机
├── cpu/
│   ├── cpu.go       # 6502 CPU 模拟
│   └── nestest_test.go  # nestest 集成测试
├── ppu/
│   └── ppu.go       # PPU 图形渲染
├── apu/
│   └── apu.go       # APU 音频输出
├── bus/
│   └── bus.go       # 地址总线 + 内存映射
├── cartridge/
│   ├── cartridge.go # iNES 解析 + Mapper 接口
│   ├── mapper0.go   # Mapper 0 (NROM)
│   ├── mapper1.go   # Mapper 1 (MMC1)
│   └── mapper4.go   # Mapper 4 (MMC3)
├── input/
│   └── input.go     # 键盘控制器
└── rom/             # ROM 文件目录
```

## 参考资料

- [NESDev Wiki](https://www.nesdev.org/wiki/Nesdev_Wiki)
- [6502 Instruction Reference](https://www.masswerk.at/6502/6502_instruction_set.html)
- [nestest](https://github.com/christopherpow/nes-test-roms)
