package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

)

const (
	N 		 = 81

	screenW  = 820
	screenH  = 780
	cellSize = 60
	boardX   = 160
	boardY   = 160
)

type Difficulty struct {
	Name  string
	Clues int
}

var difficulties = []Difficulty{
	{Name: "Easy", Clues: 40},
	{Name: "Medium", Clues: 32},
	{Name: "Hard", Clues: 24},
}

type Game struct {
	rng *rand.Rand

	diff Difficulty

	puzzle      [N]int
	startPuzzle [N]int
	solution    [N]int
	fixed       [N]bool

	selected int 

	paused bool

	solver Solver

	pixel *ebiten.Image
}

func (g *Game) rebuildFixed() {
	for i := 0; i < N; i++ {
		g.fixed[i] = g.puzzle[i] != 0
	}
}

func (g *Game) newPuzzle(diff Difficulty) {
	g.diff = diff
	g.solver.Reset(g.rng)

	// 1) generate a full solved grid
	var full [N]int
	if !fillRandom(&full, g.rng) {
		log.Println("failed to generate a full sudoku; retrying")
		return
	}
	g.solution = full

	// 2) remove cells while preserving uniqueness
	puz := full
	perm := g.rng.Perm(N)
	clues := N
	for _, pos := range perm {
		if clues <= diff.Clues {
			break
		}
		backup := puz[pos]
		puz[pos] = 0
		if countSolutions(puz, 2) != 1 {
			puz[pos] = backup
			continue
		}
		clues--
	}

	g.puzzle = puz
	g.startPuzzle = puz
	g.rebuildFixed()
	g.selected = 0
}

func (g *Game) Layout(outsideW, outsideH int) (int, int) { return screenW, screenH }

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	// Pause toggle always works
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.paused = !g.paused
		// When paused, stop wall timer so solve time doesn't include pause
		if g.solver.active && g.solver.running {
			if g.paused {
				g.solver.wall.Stop()
			} else {
				g.solver.wall.Start()
			}
		}
	}

	if g.paused {
		return nil
	}

	// Difficulty hotkeys
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.newPuzzle(difficulties[0])
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.newPuzzle(difficulties[1])
	}
	if inpututil.IsKeyJustPressed(ebiten.Key3) {
		g.newPuzzle(difficulties[2])
	}

	// Reset current puzzle
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.puzzle = g.startPuzzle
		g.rebuildFixed()
		g.solver.Reset(g.rng)
	}

	// Start/Stop auto solver
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.solver.Toggle()
	}

	// Single step
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.solver.StepOnce(&g.puzzle)
	}

	// Speed control
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		if g.solver.speedStepsPerTick > 1 {
			g.solver.speedStepsPerTick--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		if g.solver.speedStepsPerTick < 20 {
			g.solver.speedStepsPerTick++
		}
	}

	// Mouse select
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if idx, ok := pointToCell(mx, my); ok {
			g.selected = idx
		}
	}

	// Arrow navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.selected = (g.selected/9)*9 + (g.selected%9+8)%9
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.selected = (g.selected/9)*9 + (g.selected%9+1)%9
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		r := (g.selected/9 + 8) % 9
		g.selected = r*9 + (g.selected % 9)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		r := (g.selected/9 + 1) % 9
		g.selected = r*9 + (g.selected % 9)
	}

	// Editing is disabled while solver is actively running (otherwise it corrupt the search tree)
	if !(g.solver.active && g.solver.running) && !g.fixed[g.selected] {
		// Clear
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
			g.puzzle[g.selected] = 0
		}
		// Digits 1..9 (only valid moves)
		for d := 1; d <= 9; d++ {
			if inpututil.IsKeyJustPressed(keyForDigit(d)) {
				if candidatesMask(g.puzzle, g.selected)&(1<<d) != 0 {
					g.puzzle[g.selected] = d
				}
			}
		}
	}

	// Run solver incrementally
	if g.solver.active && g.solver.running {
		t0 := time.Now()
		for i := 0; i < g.solver.speedStepsPerTick; i++ {
			g.solver.step(&g.puzzle)
			if g.solver.last.Kind == ActionSolved || g.solver.last.Kind == ActionFailed {
				g.solver.Stop()
				break
			}
		}
		g.solver.compute += time.Since(t0)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	bg := color.RGBA{R: 18, G: 18, B: 22, A: 255}
	screen.Fill(bg)

	g.drawBoard(screen)
	g.drawHUD(screen)

	if g.paused {
		g.drawPauseOverlay(screen)
	}
}

func NewGame() *Game {
	g := &Game {
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		selected: 0,
		diff:     difficulties[0],	
	}
	g.pixel = ebiten.NewImage(1, 1)
	g.pixel.Fill(color.White)

	return g
}

func main() {
	ebiten.SetWindowTitle("Sugoku Solver")
	ebiten.SetWindowSize(screenW, screenH)
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("the end")

}