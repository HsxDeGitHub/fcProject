# NES 模拟器输入修复计划

## Context

模拟器画面正常渲染，但无法控制游戏。诊断发现：
- `PC=$8150` 固定不变，CPU 在死循环
- `Ctrl=$10`（bit7=0），NMI 从未启用
- `keys=[(none)]`，键盘事件未被游戏读取

## Root Cause

**PC=$8150 的代码在轮询 `$2002` bit 6（sprite 0 hit），而非 bit 7（VBlank）**：

```asm
$8150: LDA $2002   ; 读 PPU 状态
$8153: AND #$40    ; 检查 bit 6（sprite 0 hit）
$8155: BEQ $8150   ; 未设置则死循环
```

`ppu/ppu.go` 从未设置 Status bit 6（0x40）。VBlank（bit 7）在帧开始时设置，但 sprite 0 hit（bit 6）**从未设置**。CPU 永远等不到 bit 6 → 死循环 → 游戏主循环和 NMI 永远不会启动 → 控制器读取代码永远不会执行。

## Fix Plan

### Step 1: 在帧开始时设置 sprite 0 hit（bit 6）

**文件**：`ppu/ppu.go`

在 `main.go` 的 `Update()` 中设置 VBlank 时，同时设置 sprite 0 hit：
```go
g.PPU.Status |= 0x80 | 0x40
```

### Step 2: $2002 读取时不清除 bit 6（仅清除 bit 7）

**文件**：`ppu/ppu.go` ReadRegister for $2002

当前代码清除 `0x80`（bit 7）。bit 6 不应在读取时清除（与真实 NES 行为一致：sprite 0 hit 在预渲染线清除，而非读取 $2002 时）。

### Step 3: 在 RenderFrame 末尾清除 bit 6

**文件**：`ppu/ppu.go` RenderFrame

渲染完成后清除 sprite 0 hit，为下一帧重置。

### Step 4: 清理 main.go 中的诊断日志

移除临时的 frameCount 和 log.Printf 诊断代码。

## Verification

1. `go build ./... && go test ./...` —— 构建和测试通过
2. `go run .` —— 按 Enter：标题画面响应 → 进入游戏 → Z/X 可控制角色
