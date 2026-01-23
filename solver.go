package main

import (
	"math/rand"
	"time"
	"math/bits"
)

type Frame struct {
	idx        int
	candidates []int
	next       int
}

type Solver struct {
	active  bool // solver mode enabled (state exists)
	running bool

	stack []Frame
	steps int

	wall Stopwatch

	compute time.Duration

	speedStepsPerTick int

	last Action
	rng  *rand.Rand
}

func (s *Solver) Reset(rng *rand.Rand) {
	s.active = false
	s.running = false
	s.stack = nil
	s.steps = 0
	s.wall.Reset()
	s.compute = 0
	s.speedStepsPerTick = 1
	s.last = Action{Kind: ActionNone, Idx: -1}
	s.rng = rng
}

func (s *Solver) Start() {
	s.active = true
	s.running = true
	s.wall.Start()
}

func (s *Solver) Stop() {
	s.running = false
	s.wall.Stop()
}

func (s *Solver) Toggle() {
	if !s.active {
		s.Start()
		return
	}
	if s.running {
		s.Stop()
	} else {
		s.running = true
		s.wall.Start()
	}
}

func (s *Solver) StepOnce(grid *[N]int) {
	s.active = true
	if !s.wall.running {
		// stepping counts as "wall time" too
		s.wall.Start()
		// but we stop immediately after one step
		defer s.wall.Stop()
	}
	s.step(grid)
}

/*
	Here, 'used' is bitmasking, guaranteed it create a number always equals to 1, it force...
	its idempotent, it doesnt change after
*/
func candidatesMask(grid [N]int, idx int) uint16 {
	if grid[idx] != 0 {          // if already full 
		return 0
	}
	row := idx / 9
	col := idx % 9

	var used uint16 
	for i := 0; i < 9; i++ {
		if v := grid[row*9+i]; v != 0 { // digit on position (r, i)
			used |= 1 << v 
		}
		if v := grid[i*9+col]; v != 0 { // digit on position (i, c)
			used |= 1 << v
		}
	}
	
	br := (row / 3) * 3 
	bc := (col / 3) * 3
	for dr := 0; dr < 3; dr++ { // go into 3x3 row
		for dc := 0; dc < 3; dc++ { // go into 3x3 col
			if v := grid[(br+dr)*9+(bc+dc)]; v != 0 { // value of subsquare
				used |= 1 << v // mark as used so equals to 1
			}
		}
	}

	/*
		all  = 0b1111111110

		used = 0b1000100100
         		 ^   ^  ^
         		 9   5  2
		all &^ used
		= all & (~used)
		= 0b1111111110
		&0b0111011011
		= 0b0111011010

		bits 2,5,9 returns to 0, the others stays to 1 => autorized candidates
	*/
	var all uint16 // every bits 1..9
	for d := 1; d <= 9; d++ { // build the entire mask
		all |= 1 << d
	}
	return all &^ used  // returns autoried bits (without used), so turn into 0 every bits that are 1 on used
}


func maskToDigits(mask uint16) []int {
	out := make([]int, 0, 9)
	for d := 1; d <= 9; d++ {
		if mask&(1<<d) != 0 {
			out = append(out, d)
		}
	}
	return out
}

func pointToCell(mx, my int) (int, bool) {
	if mx < boardX || my < boardY || mx >= boardX+cellSize*9 || my >= boardY+cellSize*9 {
		return 0, false
	}
	c := (mx - boardX) / cellSize
	r := (my - boardY) / cellSize
	return r*9 + c, true
}

// MRV: pick empty cell with minimum remaining values (candidates).
func findBestEmpty(grid *[N]int) (idx int, mask uint16, ok bool) {
	bestIdx := -1
	var bestMask uint16
	bestCount := 10

	for i := 0; i < N; i++ {
		if grid[i] != 0 {
			continue
		}
		m := candidatesMask(*grid, i)
		c := bits.OnesCount16(m)
		if c == 0 {
			return -1, 0, false
		}
		if c < bestCount {
			bestCount = c
			bestIdx = i
			bestMask = m
			if c == 1 {
				break
			}
		}
	}

	if bestIdx == -1 {
		return -1, 0, true // solved
	}
	return bestIdx, bestMask, true
}

func countSolutions(grid [N]int, limit int) int {
	var rec func(*[N]int) int
	rec = func(g *[N]int) int {
		idx, mask, ok := findBestEmpty(g)
		if !ok {
			return 0
		}
		if idx == -1 {
			return 1
		}
		total := 0
		for d := 1; d <= 9; d++ {
			if mask&(1<<d) == 0 {
				continue
			}
			g[idx] = d
			total += rec(g)
			g[idx] = 0
			if total >= limit {
				return total
			}
		}
		return total
	}
	tmp := grid
	return rec(&tmp)
}

func fillRandom(grid *[N]int, rng *rand.Rand) bool {
	idx, mask, ok := findBestEmpty(grid)
	if !ok {
		return false
	}
	if idx == -1 {
		return true
	}
	digs := maskToDigits(mask)
	rng.Shuffle(len(digs), func(i, j int) { digs[i], digs[j] = digs[j], digs[i] })
	for _, d := range digs {
		grid[idx] = d
		if fillRandom(grid, rng) {
			return true
		}
		grid[idx] = 0
	}
	return false
}

// Depth-first backtracking search tree algorithm
func (s *Solver) step(grid *[N]int) {
	// If already terminal, do nothing
	if s.last.Kind == ActionSolved || s.last.Kind == ActionFailed {
		return
	}

	if len(s.stack) == 0 {
		idx, mask, ok := findBestEmpty(grid)
		if !ok {
			s.last = Action{Kind: ActionFailed, Idx: -1}
			return
		}
		if idx == -1 {
			s.last = Action{Kind: ActionSolved, Idx: -1}
			return
		}
		cands := maskToDigits(mask)
		s.rng.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })
		s.stack = append(s.stack, Frame{idx: idx, candidates: cands, next: 0})
		s.last = Action{Kind: ActionChoose, Idx: idx, CandCount: len(cands), Depth: len(s.stack)}
		return
	}

	top := &s.stack[len(s.stack)-1]

	// If this frame's cell is empty, we must try a candidate (or backtrack if exhausted).
	if grid[top.idx] == 0 {
		if top.next >= len(top.candidates) {
			// pop and force previous cell to clear next tick (visual backtrack).
			s.stack = s.stack[:len(s.stack)-1]
			if len(s.stack) == 0 {
				s.last = Action{Kind: ActionFailed, Idx: -1}
				return
			}
			prev := &s.stack[len(s.stack)-1]
			grid[prev.idx] = 0
			s.last = Action{Kind: ActionBacktrack, Idx: prev.idx, Depth: len(s.stack)}
			return
		}
		v := top.candidates[top.next]
		top.next++
		grid[top.idx] = v
		s.steps++
		s.last = Action{Kind: ActionTry, Idx: top.idx, Value: v, Depth: len(s.stack)}
		return
	}

	// This frame currently has a value; attempt to go deeper.
	idx, mask, ok := findBestEmpty(grid)
	if !ok {
		// Contradiction downstream => clear current top and try next candidate later.
		grid[top.idx] = 0
		s.last = Action{Kind: ActionBacktrack, Idx: top.idx, Depth: len(s.stack)}
		return
	}
	if idx == -1 {
		s.last = Action{Kind: ActionSolved, Idx: -1}
		return
	}
	cands := maskToDigits(mask)
	s.rng.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })
	s.stack = append(s.stack, Frame{idx: idx, candidates: cands, next: 0})
	s.last = Action{Kind: ActionChoose, Idx: idx, CandCount: len(cands), Depth: len(s.stack)}
}