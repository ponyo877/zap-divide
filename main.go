package main

import (
	"image"
	_ "image/png"
	"math"

	"github.com/demouth/ebitencp"
	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/jakecoffman/cp/v2"
)

const (
	screenWidth        = 640
	screenHeight       = 480
	easOutFactor       = 0.05
	beamRadius         = 5.0
	beamSpeed          = 1000.0
	ballRadius         = 30.0
	beamCooldownFrames = 2 // Fire beam every 2 frames
)

var (
	body                *cp.Body
	space               *cp.Space
	drawer              *ebitencp.Drawer
	isButtonDown        bool
	beams               []*Beam
	beamCooldownCounter int
)

type Beam struct {
	body  *cp.Body
	shape *cp.Shape
}

type Game struct {
	debugui debugui.DebugUI
}

func (g *Game) Update() error {

	x, y := ebiten.CursorPosition()
	targetX := float64(x) - screenWidth/2
	currentPos := body.Position()

	newX := currentPos.X + (targetX-currentPos.X)*easOutFactor
	if newX < -screenWidth/2+ballRadius {
		newX = -screenWidth/2 + ballRadius
	}
	if newX > screenWidth/2-ballRadius {
		newX = screenWidth/2 - ballRadius
	}

	body.SetPosition(cp.Vector{X: newX, Y: body.Position().Y})

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		isButtonDown = true
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		isButtonDown = false
	}
	targetPos := cp.Vector{
		X: float64(x) - screenWidth/2,
		Y: -(float64(y) - screenHeight/2), // Flip Y coordinate
	}

	// Decrement cooldown counter
	if beamCooldownCounter > 0 {
		beamCooldownCounter--
	}

	if isButtonDown && beamCooldownCounter == 0 {
		// Calculate direction vector
		direction := targetPos.Sub(currentPos)
		length := math.Sqrt(direction.X*direction.X + direction.Y*direction.Y)
		if length > 0 {
			// Normalize direction
			direction.X /= length
			direction.Y /= length
			// Calculate velocity
			velocityX := direction.X * beamSpeed
			velocityY := direction.Y * beamSpeed
			// Calculate start position offset from ball center
			offset := ballRadius + beamRadius
			startX := currentPos.X + direction.X*offset
			startY := currentPos.Y + direction.Y*offset
			// Fire beam
			beam := addBeam(space, startX, startY, velocityX, velocityY, beamRadius)
			beams = append(beams, beam)
			// Reset cooldown
			beamCooldownCounter = beamCooldownFrames
		}
	}

	// Remove off-screen beams
	remainingBeams := make([]*Beam, 0, len(beams))
	for _, beam := range beams {
		pos := beam.body.Position()
		// Check if beam is off-screen (with margin)
		if math.Abs(pos.X) > screenWidth || math.Abs(pos.Y) > screenHeight {
			// Remove from space
			space.RemoveShape(beam.shape)
			space.RemoveBody(beam.body)
		} else {
			remainingBeams = append(remainingBeams, beam)
		}
	}
	beams = remainingBeams

	space.Step(1 / 60.0)
	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("HSV", image.Rect(10, 10, 260, 160), func(layout debugui.ContainerLayout) {
			ctx.SetGridLayout([]int{-1, -2}, nil)
			ctx.Text("currentPosX")
			ctx.NumberFieldF(&currentPos.X, 1, 3)
			ctx.Text("currentPosY")
			ctx.NumberFieldF(&currentPos.Y, 1, 3)
			ctx.Text("targetPosX")
			ctx.NumberFieldF(&targetPos.X, 1, 3)
			ctx.Text("targetPosY")
			ctx.NumberFieldF(&targetPos.Y, 1, 3)
		})
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	cp.DrawSpace(space, drawer.WithScreen(screen))
	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Initialising Chipmunk
	space = cp.NewSpace()
	space.SetGravity(cp.Vector{X: 0, Y: -100})
	addWall(space, cp.Vector{X: -screenWidth / 2, Y: -200}, cp.Vector{X: screenWidth / 2, Y: -200}, 10)
	addBall(space, -50, -200+ballRadius, ballRadius)

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

func addBeam(space *cp.Space, x, y, vx, vy, radius float64) *Beam {
	// Use kinematic body to ignore gravity
	beamBody := space.AddBody(cp.NewKinematicBody())
	beamBody.SetPosition(cp.Vector{X: x, Y: y})
	beamBody.SetVelocity(vx, vy)

	shape := space.AddShape(
		cp.NewCircle(
			beamBody,
			radius,
			cp.Vector{},
		),
	)
	shape.SetElasticity(0)
	shape.SetFriction(0)

	return &Beam{
		body:  beamBody,
		shape: shape,
	}
}
