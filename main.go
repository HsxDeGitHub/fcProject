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

	log.Printf("Loaded ROM: %s (Mapper %d, %d PRG, %d CHR)",
		romPath, cart.Mapper, cart.PRGBanks, cart.CHRBanks)

	return g, nil
}

func (g *Game) Update() error {
	g.Input.Update()

	// Set VBlank at frame start so the CPU can detect it.
	// The NES generates VBlank during scanlines 241-261 (~20 scanlines).
	g.PPU.Status |= 0x80
	g.PPU.VblankReasserts = 10

	// Trigger NMI if VBlank + NMI output enabled (PPUCTRL bit 7)
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

func (g *Game) Draw(screen *ebiten.Image) {
	frame := g.PPU.Frame[:]
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
