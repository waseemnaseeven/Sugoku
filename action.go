package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"

)

type ActionKind int

const (
	ActionNone ActionKind = iota
	ActionChoose
	ActionTry
	ActionBacktrack
	ActionSolved
	ActionFailed
)

type Action struct {
	Kind      ActionKind
	Idx       int
	Value     int
	CandCount int
	Depth     int
}

func keyForDigit(d int) ebiten.Key {
	switch d {
	case 1:
		return ebiten.Key1
	case 2:
		return ebiten.Key2
	case 3:
		return ebiten.Key3
	case 4:
		return ebiten.Key4
	case 5:
		return ebiten.Key5
	case 6:
		return ebiten.Key6
	case 7:
		return ebiten.Key7
	case 8:
		return ebiten.Key8
	case 9:
		return ebiten.Key9
	default:
		return ebiten.Key0
	}
}

func actionString(a Action) string {
	switch a.Kind {
	case ActionChoose:
		return fmt.Sprintf("Action: choose cell r%d c%d (%d candidates) depth=%d", a.Idx/9+1, a.Idx%9+1, a.CandCount, a.Depth)
	case ActionTry:
		return fmt.Sprintf("Action: try %d at r%d c%d depth=%d", a.Value, a.Idx/9+1, a.Idx%9+1, a.Depth)
	case ActionBacktrack:
		return fmt.Sprintf("Action: backtrack (clear) at r%d c%d depth=%d", a.Idx/9+1, a.Idx%9+1, a.Depth)
	case ActionSolved:
		return "Action: SOLVED"
	case ActionFailed:
		return "Action: FAILED (no solution under current constraints)"
	default:
		return "Action: (none)"
	}
}