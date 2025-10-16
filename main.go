package main

import (
	"image"
	_ "image/png"
	"log"
	"math"

	"github.com/demouth/ebitencp"
	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/jakecoffman/cp/v2"
)

const (
	screenWidth         = 640
	screenHeight        = 480
	easeOutFactor       = 0.025
	beamRadius          = 5.0
	beamSpeed           = 1000.0
	ballRadius          = 30.0
	bouncyBallRadius    = 80.0
	minBouncyBallRadius = 10.0
	beamCooldownFrames  = 2   // Fire beam every 2 frames
	playerScale         = 0.3 // Scale factor for player images
	rotationOffset      = 0   // 90 degrees to fix orientation
)

var (
	body                *cp.Body
	bouncyBalls         []*BouncyBall
	space               *cp.Space
	drawer              *ebitencp.Drawer
	isButtonDown        bool
	beams               []*Beam
	beamCooldownCounter int
	headImg             *ebiten.Image
	bodyImg1            *ebiten.Image
	bodyImg2            *ebiten.Image
	bodyImg3            *ebiten.Image
	armImg              *ebiten.Image
	currentAngle        float64
	targetAngle         float64
	smoothingFactor     = 0.15
	accumulatedDistance float64
	bodyCyclePattern    = []int{0, 1, 2, 1} // Cycle pattern for body images
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

func init() {
	var err error
	headImg, _, err = ebitenutil.NewImageFromFile("assets/頭部.png")
	if err != nil {
		log.Fatal(err)
	}
	bodyImg1, _, err = ebitenutil.NewImageFromFile("assets/胴体部1.png")
	if err != nil {
		log.Fatal(err)
	}
	bodyImg2, _, err = ebitenutil.NewImageFromFile("assets/胴体部2.png")
	if err != nil {
		log.Fatal(err)
	}
	bodyImg3, _, err = ebitenutil.NewImageFromFile("assets/胴体部3.png")
	if err != nil {
		log.Fatal(err)
	}
	armImg, _, err = ebitenutil.NewImageFromFile("assets/腕部+銃.png")
	if err != nil {
		log.Fatal(err)
	}
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

	// Calculate target angle (mouse position relative to player)
	mousePos := cp.Vector{
		X: float64(x) - screenWidth/2,
		Y: -(float64(y) - screenHeight/2), // Flip Y coordinate
	}
	dx := mousePos.X - currentPos.X
	dy := mousePos.Y - currentPos.Y
	targetAngle = -math.Atan2(dy, dx) // Negate to reverse rotation direction

	// Normalize angle difference to [-π, π] for shortest rotation path
	angleDiff := targetAngle - currentAngle
	if angleDiff > math.Pi {
		angleDiff -= 2 * math.Pi
	} else if angleDiff < -math.Pi {
		angleDiff += 2 * math.Pi
	}

	// Apply smoothing to current angle
	currentAngle += angleDiff * smoothingFactor

	newX := currentPos.X + (targetX-currentPos.X)*easeOutFactor
	if newX < -screenWidth/2+ballRadius {
		newX = -screenWidth/2 + ballRadius
	}
	if newX > screenWidth/2-ballRadius {
		newX = screenWidth/2 - ballRadius
	}

	// Track accumulated movement distance for body animation
	distanceMoved := math.Abs(newX - currentPos.X)
	accumulatedDistance += distanceMoved

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

func drawPlayer(screen *ebiten.Image) {
	if body == nil {
		return
	}

	pos := body.Position()
	// Convert physics coordinates to screen coordinates
	screenX := pos.X + screenWidth/2
	screenY := screenHeight/2 - pos.Y

	// Get image dimensions
	bodyBounds := bodyImg1.Bounds()
	bodyW := float64(bodyBounds.Dx())
	bodyH := float64(bodyBounds.Dy())

	headBounds := headImg.Bounds()
	headW := float64(headBounds.Dx())
	headH := float64(headBounds.Dy())

	armBounds := armImg.Bounds()
	armW := float64(armBounds.Dx())
	armH := float64(armBounds.Dy())

	// Shoulder pivot point (measured from arm image)
	// Assuming shoulder is at approximately 1/5 from left and middle of height
	shoulderPivotX := armW / 5
	shoulderPivotY := armH / 2

	// Head and arm offset (slightly above body)
	headOffsetY := -150.0
	armOffsetY := -75.0

	// Determine drawing order based on angle
	// Apply rotation offset to fix orientation
	displayAngle := currentAngle + rotationOffset
	// Check if facing left (left half of the circle)
	isFacingLeft := displayAngle < -math.Pi/2 || displayAngle > math.Pi/2

	drawBody := func() {
		// Select body image based on accumulated distance
		bodyImages := []*ebiten.Image{bodyImg1, bodyImg2, bodyImg3}
		distancePerFrame := 20.0 // Distance threshold for changing frames
		cycleIndex := int(accumulatedDistance/distancePerFrame) % len(bodyCyclePattern)
		imageIndex := bodyCyclePattern[cycleIndex]
		currentBodyImg := bodyImages[imageIndex]

		// Draw body (no rotation)
		bodyOp := &ebiten.DrawImageOptions{}
		// 1. Pivot: move center to origin
		bodyOp.GeoM.Translate(-bodyW/2, -bodyH/2)
		// 2. Scale with flip if facing left
		if isFacingLeft {
			bodyOp.GeoM.Scale(-playerScale, playerScale)
		} else {
			bodyOp.GeoM.Scale(playerScale, playerScale)
		}
		// 3. Position
		bodyOp.GeoM.Translate(screenX, screenY)
		screen.DrawImage(currentBodyImg, bodyOp)
	}

	drawHead := func() {
		// Draw head with rotation
		headOp := &ebiten.DrawImageOptions{}
		// 1. Pivot: move center to origin
		headOp.GeoM.Translate(-headW/2, -headH/2)
		// 2. Scale with flip if facing left
		if isFacingLeft {
			headOp.GeoM.Scale(-playerScale, playerScale)
		} else {
			headOp.GeoM.Scale(playerScale, playerScale)
		}
		// 3. Rotate with offset (reverse rotation and add 90 degrees when flipped)
		if isFacingLeft {
			headOp.GeoM.Rotate(displayAngle + math.Pi)
		} else {
			headOp.GeoM.Rotate(displayAngle)
		}
		// 4. Position: move to player position with offset
		headOp.GeoM.Translate(screenX, screenY+headOffsetY*playerScale)
		screen.DrawImage(headImg, headOp)
	}

	drawArm := func() {
		// Draw arm with rotation (pivot at shoulder)
		armOp := &ebiten.DrawImageOptions{}
		// 1. Pivot: move shoulder to origin
		armOp.GeoM.Translate(-shoulderPivotX, -shoulderPivotY)
		// 2. Scale with flip if facing left
		if isFacingLeft {
			armOp.GeoM.Scale(-playerScale, playerScale)
		} else {
			armOp.GeoM.Scale(playerScale, playerScale)
		}
		// 3. Rotate with offset (reverse rotation and add 90 degrees when flipped)
		if isFacingLeft {
			armOp.GeoM.Rotate(displayAngle + math.Pi)
		} else {
			armOp.GeoM.Rotate(displayAngle)
		}
		// 4. Position: move to player position with offset
		armOp.GeoM.Translate(screenX, screenY+armOffsetY*playerScale)
		screen.DrawImage(armImg, armOp)
	}

	// Draw in fixed order: head -> arm -> body
	drawBody()
	drawHead()
	drawArm()
}

func (g *Game) Draw(screen *ebiten.Image) {
	cp.DrawSpace(space, drawer.WithScreen(screen))
	drawPlayer(screen)
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

	addBall(space, -50, -180+ballRadius, 10)
	// addBall(space, -50, -180+ballRadius, ballRadius)
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
