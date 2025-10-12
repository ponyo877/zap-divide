package main

import (
	_ "image/png"

	"github.com/demouth/ebitencp"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/jakecoffman/cp/v2"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	body   *cp.Body
	space  *cp.Space
	drawer *ebitencp.Drawer
)

type Game struct{}

func (g *Game) Update() error {
	x, _ := ebiten.CursorPosition()
	body.SetPosition(cp.Vector{X: float64(x) - screenWidth/2, Y: body.Position().Y})
	space.Step(1 / 60.0)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	cp.DrawSpace(space, drawer.WithScreen(screen))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Initialising Chipmunk
	space = cp.NewSpace()
	space.SetGravity(cp.Vector{X: 0, Y: -100})
	addWall(space, cp.Vector{X: -200, Y: 0}, cp.Vector{X: 200, Y: 0}, 10)
	addWall(space, cp.Vector{X: -200, Y: 200}, cp.Vector{X: -200, Y: 0}, 10)
	addWall(space, cp.Vector{X: 200, Y: 200}, cp.Vector{X: 200, Y: 0}, 10)
	addBall(space, -50, 200, 30)

	// Initialising Ebitengine/v2
	game := &Game{}
	drawer = ebitencp.NewDrawer(screenWidth, screenHeight)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.RunGame(game)
}

func addWall(space *cp.Space, pos1 cp.Vector, pos2 cp.Vector, radius float64) {
	shape := space.AddShape(cp.NewSegment(space.StaticBody, pos1, pos2, radius))
	shape.SetElasticity(0)
	shape.SetFriction(0.5)
}

func addBall(space *cp.Space, x, y, radius float64) *cp.Body {
	mass := radius * radius / 100.0
	body = space.AddBody(
		cp.NewBody(
			mass,
			cp.MomentForCircle(mass, 0, radius, cp.Vector{}),
		),
	)
	body.SetPosition(cp.Vector{X: x, Y: y})

	shape := space.AddShape(
		cp.NewCircle(
			body,
			radius,
			cp.Vector{},
		),
	)
	shape.SetElasticity(0)
	shape.SetFriction(0.5)
	return body
}
