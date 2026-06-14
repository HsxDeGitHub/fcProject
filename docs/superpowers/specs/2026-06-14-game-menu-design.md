# NES 模拟器游戏选择菜单设计规格

## 概述

启动模拟器时不直接加载游戏，而是显示游戏选择菜单。扫描 `rom/` 目录下的 `.nes` 文件，用户通过键盘选择后进入游戏。

## 架构

`main.go` 使用状态机管理两个界面：

- **MenuState**：扫描 ROM 文件、显示列表、键盘选择
- **GameState**：加载选中 ROM、运行模拟器循环

```
MenuState ←──────┐
  │ Enter        │ ESC
  ▼              │
GameState ───────┘
```

## MenuState 设计

- **启动**：扫描 `rom/` 目录，列出所有 `.nes` 文件
- **渲染**：黑色背景 + 白色文字列表，当前选中项反色高亮（白色背景 + 黑色文字）
- **交互**：
  - ↑↓ 移动光标
  - Enter 确认选择，启动 GameState
- **布局**：文字居中排列，文件名显示在屏幕中央，上下留白
- **窗口**：768×720（与游戏画面一致，无需调整）

## GameState 设计

- **启动**：加载选中的 ROM 文件，初始化 CPU/PPU/APU/Bus/Input
- **运行**：现有的模拟器循环（Ebitengine Update/Draw/Layout）
- **退出**：ESC 返回 MenuState

## 键盘映射

| 菜单界面 | 游戏界面 |
|----------|----------|
| ↑↓ 选择 | ↑↓ 方向键 |
| Enter 确认 | Enter/Space = Start |
| - | Z/X = A/B |
| - | ESC 返回菜单 |

## 实现方式

在 `main.go` 中定义 `MenuState` 和 `GameState`，`App` 结构体持有当前状态和切换逻辑。

```go
type AppState int
const (
    StateMenu AppState = iota
    StateGame
)

type App struct {
    state AppState
    menu  *MenuState
    game  *GameState
}
```

## 测试

- 启动后确认显示菜单
- 确认 `rom/` 下所有 `.nes` 文件出现在列表中
- ↑↓ 键移动光标验证
- Enter 选择游戏验证进入模拟器
- ESC 从游戏返回菜单验证
- 选择另一个游戏验证正常切换
