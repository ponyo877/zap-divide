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
	screenWidth         = 640
	screenHeight        = 480
	easOutFactor        = 0.05
	beamRadius          = 5.0
	beamSpeed           = 1000.0
	ballRadius          = 30.0
	bouncyBallRadius    = 80.0
	minBouncyBallRadius = 10.0
	beamCooldownFrames  = 2 // Fire beam every 2 frames
)

var (
	body                *cp.Body
	bouncyBalls         []*BouncyBall
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

type BouncyBall struct {
	body     *cp.Body
	shape    *cp.Shape
	radius   float64
	hitCount int
}

type Game struct {
	debugui debugui.DebugUI
}

func randomDiagonalUpVelocity(num int, height float64) cp.Vector {
	// Random X velocity: -0.5 to 0.5
	vx := -150.0
	vy := -200.0 + height
	if num > 0 {
		vx = 150.0
	}
	return cp.Vector{X: vx, Y: vy}
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

	// Check collisions between beams and bouncyBalls
	remainingBeams := make([]*Beam, 0, len(beams))
	ballsToSplit := make(map[int]bool) // Track which balls need to split

	for _, beam := range beams {
		beamPos := beam.body.Position()
		shouldRemoveBeam := false

		// Check if beam is off-screen
		if math.Abs(beamPos.X) > screenWidth || math.Abs(beamPos.Y) > screenHeight {
			shouldRemoveBeam = true
		}

		// Check collision with all bouncyBalls
		if !shouldRemoveBeam {
			for i, ball := range bouncyBalls {
				ballPos := ball.body.Position()
				dx := beamPos.X - ballPos.X
				dy := beamPos.Y - ballPos.Y
				distance := math.Sqrt(dx*dx + dy*dy)

				if distance < ball.radius+beamRadius {
					// Collision detected
					ball.hitCount++
					shouldRemoveBeam = true

					// Mark ball for splitting if hit 3 times
					if ball.hitCount >= 10 {
						ballsToSplit[i] = true
					}
					break
				}
			}
		}

		if shouldRemoveBeam {
			space.RemoveShape(beam.shape)
			space.RemoveBody(beam.body)
		} else {
			remainingBeams = append(remainingBeams, beam)
		}
	}
	beams = remainingBeams

	// Process ball splits
	remainingBalls := make([]*BouncyBall, 0, len(bouncyBalls))
	for i, ball := range bouncyBalls {
		if ballsToSplit[i] {
			// Remove original ball
			space.RemoveShape(ball.shape)
			space.RemoveBody(ball.body)

			// Split if large enough
			newRadius := ball.radius / 2
			if newRadius >= minBouncyBallRadius {
				ballPos := ball.body.Position()
				// Create two new balls
				for j := 0; j < 2; j++ {
					newBall := addBouncyBall(space, ballPos.X, ballPos.Y, newRadius)
					vel := randomDiagonalUpVelocity(j, ballPos.Y)
					newBall.body.SetVelocity(vel.X, vel.Y)
					remainingBalls = append(remainingBalls, newBall)
				}
			}
		} else {
			remainingBalls = append(remainingBalls, ball)
		}
	}
	bouncyBalls = remainingBalls

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
	// Bottom wall
	addWall(space, cp.Vector{X: -screenWidth / 2, Y: -screenHeight / 2}, cp.Vector{X: screenWidth / 2, Y: -screenHeight / 2}, 40)
	// Left wall
	addWall(space, cp.Vector{X: -screenWidth / 2, Y: -screenHeight / 2}, cp.Vector{X: -screenWidth / 2, Y: screenHeight / 2}, 40)
	// Right wall
	addWall(space, cp.Vector{X: screenWidth / 2, Y: -screenHeight / 2}, cp.Vector{X: screenWidth / 2, Y: screenHeight / 2}, 40)

	addBall(space, -50, -180+ballRadius, ballRadius)
	// Add larger bouncy ball with diagonal downward velocity
	initialBouncyBall := addBouncyBall(space, 100, 100, bouncyBallRadius)
	initialBouncyBall.body.SetVelocity(150, -200)
	bouncyBalls = append(bouncyBalls, initialBouncyBall)

	// Initialising Ebitengine/v2
	game := &Game{}
	drawer = ebitencp.NewDrawer(screenWidth, screenHeight)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.RunGame(game)
}

func addWall(space *cp.Space, pos1 cp.Vector, pos2 cp.Vector, radius float64) {
	shape := space.AddShape(cp.NewSegment(space.StaticBody, pos1, pos2, radius))
	shape.SetElasticity(1.0)
	shape.SetFriction(0.5)
	// Set collision filter for walls
	shape.SetFilter(cp.ShapeFilter{
		Group:      0,
		Categories: 0b1,    // wall category
		Mask:       0xFFFF, // collide with everything
	})
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

func addBouncyBall(space *cp.Space, x, y, radius float64) *BouncyBall {
	// Use lighter mass for less gravity effect
	mass := radius * radius / 500.0
	ballBody := space.AddBody(
		cp.NewBody(
			mass,
			cp.MomentForCircle(mass, 0, radius, cp.Vector{}),
		),
	)
	ballBody.SetPosition(cp.Vector{X: x, Y: y})

	shape := space.AddShape(
		cp.NewCircle(
			ballBody,
			radius,
			cp.Vector{},
		),
	)
	// Perfect elasticity for bouncing
	shape.SetElasticity(1.0)
	shape.SetFriction(0.1)
	// Set collision filter to prevent bouncyBalls from colliding with each other
	shape.SetFilter(cp.ShapeFilter{
		Group:      0,
		Categories: 0b100, // bouncyBall category
		Mask:       0b1,   // only collide with walls (category 0b1)
	})

	return &BouncyBall{
		body:     ballBody,
		shape:    shape,
		radius:   radius,
		hitCount: 0,
	}
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
	// Set collision filter to prevent beam from physically colliding
	shape.SetFilter(cp.ShapeFilter{
		Group:      0,
		Categories: 0b10, // beam category
		Mask:       0,    // don't collide with anything
	})

	return &Beam{
		body:  beamBody,
		shape: shape,
	}
}
