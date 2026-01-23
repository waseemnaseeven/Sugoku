package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/basicfont"
)



func (g *Game) drawRect(screen *ebiten.Image, x, y, w, h int, clr color.Color) {
	limits := math.MaxUint16 // 65535 always returns 16-bit channel values in the range 
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))

	red, green, blue, a := clr.RGBA()
	op.ColorScale.Scale(float32(red)/float32(limits), float32(green)/float32(limits), float32(blue)/float32(limits), float32(a)/float32(limits))
	screen.DrawImage(g.pixel, op)
}

func drawCandidates(screen *ebiten.Image, cellX, cellY int, mask uint16, col color.Color) {
	face := basicfont.Face7x13
	// 3x3 mini-grid positions inside a 60x60 cell
	for d := 1; d <= 9; d++ {
		if mask&(1<<d) == 0 {
			continue
		}
		rr := (d - 1) / 3
		cc := (d - 1) % 3
		x := cellX + 8 + cc*18
		y := cellY + 18 + rr*18
		text.Draw(screen, fmt.Sprintf("%d", d), face, x, y, col)
	}
}

func (g *Game) drawOutline(screen *ebiten.Image, x, y, w, h, t int, col color.Color) {
	// top
	g.drawRect(screen, x, y, w, t, col)
	// bottom
	g.drawRect(screen, x, y+h-t, w, t, col)
	// left
	g.drawRect(screen, x, y, t, h, col)
	// right
	g.drawRect(screen, x+w-t, y, t, h, col)
}

func (g *Game) drawBoard(screen *ebiten.Image) {
	// board background
	g.drawRect(screen, boardX-6, boardY-6, cellSize*9+12, cellSize*9+12, color.RGBA{R: 30, G: 30, B: 40, A: 255})

	focusIdx := -1
	if g.solver.last.Kind == ActionChoose || g.solver.last.Kind == ActionTry || g.solver.last.Kind == ActionBacktrack {
		focusIdx = g.solver.last.Idx
	}

	// Cells background
	for i := 0; i < N; i++ {
		row, col := i/9, i%9 // give lign and column
		x := boardX + col*cellSize
		y := boardY + row*cellSize

		// base cell background
		base := color.RGBA{R: 24, G: 24, B: 32, A: 255}
		if i == g.selected {
			base = color.RGBA{R: 52, G: 52, B: 78, A: 255}
		}
		g.drawRect(screen, x, y, cellSize, cellSize, base)
	}

	// Grid lines
	thin := color.RGBA{R: 80, G: 80, B: 100, A: 255}
	thick := color.RGBA{R: 160, G: 160, B: 200, A: 255}
	for i := 0; i <= 9; i++ {
		w := 2 // arbitrary ??? 
		col := thin
		if i%3 == 0 {
			w = 5
			col = thick
		}
		g.drawRect(screen, boardX+i*cellSize-w/2, boardY, w, cellSize*9, col)
		g.drawRect(screen, boardX, boardY+i*cellSize-w/2, cellSize*9, w, col)
	}

	// Cell contents (values and candidates) drawn above grid lines
	for i := 0; i < N; i++ {
		r, c := i/9, i%9
		x := boardX + c*cellSize
		y := boardY + r*cellSize

		v := g.puzzle[i]
		if v != 0 {
			col := color.RGBA{R: 230, G: 230, B: 240, A: 255} // fixed
			if !g.fixed[i] {
				col = color.RGBA{R: 180, G: 220, B: 255, A: 255} // user
			}
			text.Draw(screen, fmt.Sprintf("%d", v), basicfont.Face7x13,
				x+cellSize/2-4, y+cellSize/2+6, col)
		}

		if v == 0 && (i == g.selected || i == focusIdx) {
			m := candidatesMask(g.puzzle, i)
			cCol := color.RGBA{R: 120, G: 120, B: 140, A: 255}
			if i == focusIdx {
				cCol = color.RGBA{R: 200, G: 200, B: 120, A: 255}
			}
			drawCandidates(screen, x, y, m, cCol)
		}
	}

	// Solver action outline 
	if focusIdx != -1 {
		row, col := focusIdx/9, focusIdx%9
		x := boardX + col*cellSize
		y := boardY + row*cellSize
		out := color.RGBA{R: 210, G: 210, B: 130, A: 255}
		if g.solver.last.Kind == ActionTry {
			out = color.RGBA{R: 130, G: 220, B: 160, A: 255}
		}
		if g.solver.last.Kind == ActionBacktrack {
			out = color.RGBA{R: 230, G: 120, B: 120, A: 255}
		}
		g.drawOutline(screen, x+2, y+2, cellSize-4, cellSize-4, 3, out)
	}
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	face := basicfont.Face7x13
	y := 18

	text.Draw(screen,
		"Sudoku |  1/2/3 new puzzle  |  R reset  |  SPACE run/stop  |  N step  |  -/+ speed  |  P pause  |  ESC quit",
		face, 10, y, color.White)

	y += 22
	status := "idle"
	if g.solver.active && g.solver.running {
		status = "running"
	} else if g.solver.last.Kind == ActionSolved {
		status = "solved"
	} else if g.solver.last.Kind == ActionFailed {
		status = "failed"
	}
	text.Draw(screen,
		fmt.Sprintf("Difficulty: %s   |   Solver: %s   |   speed: %d steps/tick",
			g.diff.Name, status, g.solver.speedStepsPerTick),
		face, 10, y, color.RGBA{R: 200, G: 200, B: 255, A: 255})

	y += 22
	text.Draw(screen,
		fmt.Sprintf("Steps: %d   |   Wall time: %s   |   Compute time: %s",
			g.solver.steps, g.solver.wall.Elapsed(), g.solver.compute),
		face, 10, y, color.RGBA{R: 170, G: 220, B: 255, A: 255})

	y += 22
	text.Draw(screen,
		actionString(g.solver.last),
		face, 10, y, color.RGBA{R: 240, G: 220, B: 180, A: 255})

	// Show candidates for selected cell
	y += 22
	if g.puzzle[g.selected] == 0 {
		m := candidatesMask(g.puzzle, g.selected)
		text.Draw(screen,
			fmt.Sprintf("Selected cell r%d c%d candidates: %v", g.selected/9+1, g.selected%9+1, maskToDigits(m)),
			face, 10, y, color.RGBA{R: 160, G: 160, B: 180, A: 255})
	} else {
		text.Draw(screen,
			fmt.Sprintf("Selected cell r%d c%d value: %d", g.selected/9+1, g.selected%9+1, g.puzzle[g.selected]),
			face, 10, y, color.RGBA{R: 160, G: 160, B: 180, A: 255})
	}
}

func (g *Game) drawPauseOverlay(screen *ebiten.Image) {
	overlay := color.RGBA{R: 0, G: 0, B: 0, A: 140}
	g.drawRect(screen, 0, 0, screenW, screenH, overlay)
	text.Draw(screen, "PAUSED (press P to resume)", basicfont.Face7x13, screenW/2-90, screenH/2, color.White)
}

