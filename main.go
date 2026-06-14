package main

import (
	"image/color"
	"log"
	"os"
	"sort"
	"strings"

	"fcProject/apu"
	"fcProject/bus"
	"fcProject/cartridge"
	"fcProject/cpu"
	"fcProject/input"
	"fcProject/ppu"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	cyclesPerFrame = 29781
	screenWidth    = 256 * 3
	screenHeight   = 240 * 3
)

type MenuState struct {
	romFiles    []string
	cursor      int
	prevUp      bool
	prevDown    bool
	errMsg      string
	errTimer    int
	menuFont    font.Face
}

func loadChineseFont() font.Face {
	fontPaths := []string{
		"/System/Library/Fonts/STHeiti Medium.ttc",
		"/System/Library/Fonts/PingFang.ttc",
	}
	for _, p := range fontPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		// Try as TTC first, then as single font
		var tt *opentype.Font
		col, cerr := opentype.ParseCollection(data)
		if cerr == nil && col.NumFonts() > 0 {
			tt, cerr = col.Font(0)
			if cerr != nil {
				continue
			}
		} else {
			tt, err = opentype.Parse(data)
			if err != nil {
				continue
			}
		}
		if tt != nil {
			face, err := opentype.NewFace(tt, &opentype.FaceOptions{
				Size: 14, DPI: 72, Hinting: font.HintingFull,
			})
			if err == nil && face != nil {
				return face
			}
		}
	}
	return nil
}

func NewMenu() *MenuState {
	entries, err := os.ReadDir("rom")
	if err != nil {
		log.Fatal(err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".nes") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return &MenuState{
		romFiles: files,
		menuFont: loadChineseFont(),
	}
}

func (m *MenuState) Update() error {
	up := ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	down := ebiten.IsKeyPressed(ebiten.KeyArrowDown)

	if up && !m.prevUp && m.cursor > 0 {
		m.cursor--
	}
	if down && !m.prevDown && m.cursor < len(m.romFiles)-1 {
		m.cursor++
	}
	m.prevUp = up
	m.prevDown = down
	if m.errTimer > 0 {
		m.errTimer--
	}
	return nil
}

func (m *MenuState) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	face := m.menuFont
	if face == nil {
		return // Can't render without font
	}

	// Title
	title := "=== NES Emulator ==="
	text.Draw(screen, title, face, 280, 200, color.White)

	// Game list
	for i, name := range m.romFiles {
		y := 260 + i*24
		if i == m.cursor {
			// Highlight: draw white rectangle behind text
			w := len(name) * 7 // approximate text width
			hl := ebiten.NewImage(w+8, 16)
			hl.Fill(color.White)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(280, float64(y-12))
			screen.DrawImage(hl, op)
			text.Draw(screen, name, face, 284, y, color.Black)
		} else {
			text.Draw(screen, name, face, 280, y, color.White)
		}
	}

	// Error message
	if m.errTimer > 0 {
		text.Draw(screen, m.errMsg, face, 220, 560, color.RGBA{255, 80, 80, 255})
	}
	// Footer
	text.Draw(screen, "Arrow Keys: Select  Enter: Start", face, 250, 680, color.RGBA{128, 128, 128, 255})
}

func (m *MenuState) Layout(w, h int) (int, int) { return 768, 720 }

func (m *MenuState) Selected() string {
	if m.cursor < len(m.romFiles) {
		return m.romFiles[m.cursor]
	}
	return ""
}

func (m *MenuState) SetError(msg string) {
	m.errMsg = msg
	m.errTimer = 180 // Show error for ~3 seconds at 60fps
}

type GameState struct {
	CPU       *cpu.CPU
	PPU       *ppu.PPU
	APU       *apu.APU
	Bus       *bus.Bus
	Cartridge *cartridge.Cartridge
	Input     *input.Controller

	frameImage *ebiten.Image
}

func NewGameState(romPath string) (*GameState, error) {
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

	g := &GameState{
		CPU:       c,
		PPU:       p,
		APU:       a,
		Bus:       b,
		Cartridge: cart,
		Input:     inp,
		frameImage: ebiten.NewImage(ppu.ScreenWidth, ppu.ScreenHeight),
	}

	g.CPU.Reset()

	log.Printf("Loaded ROM: %s (Mapper %d, %d PRG, %d CHR)",
		romPath, cart.Mapper, cart.PRGBanks, cart.CHRBanks)

	return g, nil
}

func (g *GameState) Update() error {
	g.Input.Update()

	// Set VBlank (bit 7) and sprite 0 hit (bit 6) at frame start.
	// Bit 7 is required for VBlank wait loops during init.
	// Bit 6 is required for SMB scroll split timing ($8150 and $813D checks).
	g.PPU.Status |= 0x80 | 0x40
	g.PPU.VblankReasserts = 10

	// Trigger NMI if NMI output enabled (PPUCTRL bit 7)
	if g.PPU.Ctrl&0x80 != 0 {
		g.CPU.NMI = true
	}

	// Run CPU for one frame worth of cycles
	g.CPU.RunCycles(g.CPU.Cycles + cyclesPerFrame)

	// Render the frame using current PPU state
	g.PPU.RenderFrame()

	// Audio: generate samples (playback skipped for now)
	_ = g.APU.GenerateSamples()

	return nil
}

func (g *GameState) Draw(screen *ebiten.Image) {
	frame := g.PPU.Frame[:]
	g.frameImage.WritePixels(frame)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(3, 3)
	screen.DrawImage(g.frameImage, op)
}

func (g *GameState) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

type AppState int

const (
	StateMenu AppState = iota
	StateGame
)

type App struct {
	state     AppState
	menu      *MenuState
	game      *GameState
	prevEnter bool
	prevESC   bool
}

func NewApp() *App {
	return &App{
		state: StateMenu,
		menu:  NewMenu(),
	}
}

func (a *App) Update() error {
	enter := ebiten.IsKeyPressed(ebiten.KeyEnter)
	esc := ebiten.IsKeyPressed(ebiten.KeyEscape)

	switch a.state {
	case StateMenu:
		a.menu.Update()
		if enter && !a.prevEnter {
			sel := a.menu.Selected()
			if sel != "" {
				g, err := NewGameState("rom/" + sel)
				if err != nil {
					a.menu.SetError("Cannot load: " + err.Error())
				} else {
					a.game = g
					a.state = StateGame
				}
			}
		}
	case StateGame:
		a.game.Update()
		if esc && !a.prevESC {
			a.state = StateMenu
		}
	}
	a.prevEnter = enter
	a.prevESC = esc
	return nil
}

func (a *App) Draw(screen *ebiten.Image) {
	switch a.state {
	case StateMenu:
		a.menu.Draw(screen)
	case StateGame:
		a.game.Draw(screen)
	}
}

func (a *App) Layout(w, h int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("NES Emulator")
	ebiten.SetTPS(60)

	if err := ebiten.RunGame(NewApp()); err != nil {
		log.Fatal(err)
	}
}
