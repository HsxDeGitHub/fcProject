package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct{}

func (g *Game) Update() error                 { return nil }
func (g *Game) Draw(screen *ebiten.Image)      {}
func (g *Game) Layout(w, h int) (int, int)     { return 768, 720 }

func main() {
	ebiten.SetWindowSize(768, 720)
	ebiten.SetWindowTitle("NES Emulator")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
