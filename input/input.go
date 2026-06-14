package input

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Controller struct {
	buttons [8]bool
	strobe  bool
	index   int
}

func New() *Controller {
	return &Controller{}
}

func (c *Controller) Update() {
	c.buttons[0] = ebiten.IsKeyPressed(ebiten.KeyZ)                                                          // A
	c.buttons[1] = ebiten.IsKeyPressed(ebiten.KeyX)                                                          // B
	c.buttons[2] = ebiten.IsKeyPressed(ebiten.KeyShiftRight) || ebiten.IsKeyPressed(ebiten.KeyShiftLeft)      // Select
	c.buttons[3] = ebiten.IsKeyPressed(ebiten.KeyEnter) || ebiten.IsKeyPressed(ebiten.KeySpace)               // Start
	c.buttons[4] = ebiten.IsKeyPressed(ebiten.KeyArrowUp)                                                    // Up
	c.buttons[5] = ebiten.IsKeyPressed(ebiten.KeyArrowDown)                                                  // Down
	c.buttons[6] = ebiten.IsKeyPressed(ebiten.KeyArrowLeft)                                                  // Left
	c.buttons[7] = ebiten.IsKeyPressed(ebiten.KeyArrowRight)                                                 // Right
}

func (c *Controller) Read() uint8 {
	var val uint8 = 0
	if c.index < 8 && c.buttons[c.index] {
		val = 1
	}
	c.index++
	if c.strobe {
		c.index = 0
	}
	return val
}

func (c *Controller) Write(data uint8) {
	c.strobe = data&1 != 0
	if c.strobe {
		c.index = 0
	}
}
