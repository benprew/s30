package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

func TouchPosition() (X, Y int) {
	var touches []ebiten.TouchID
	touches = ebiten.AppendTouchIDs(touches)
	if len(touches) > 0 {
		fmt.Println("Got touch")
		X, Y = ebiten.TouchPosition(touches[0])
		return X, Y
	}
	return
}
