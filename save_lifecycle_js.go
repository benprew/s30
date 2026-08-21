//go:build js

package main

import (
	"fmt"
	"syscall/js"

	"github.com/benprew/s30/game"
)

var saveLifecycleCallbacks []js.Func

func registerSaveLifecycle(g *game.Game) {
	save := func() {
		if err := g.SaveGame(); err != nil {
			fmt.Printf("Error auto-saving browser game: %v\n", err)
		}
	}

	visibilityChange := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		if js.Global().Get("document").Get("hidden").Bool() {
			save()
		}
		return nil
	})
	pageHide := js.FuncOf(func(_ js.Value, _ []js.Value) any {
		save()
		return nil
	})

	js.Global().Get("document").Call("addEventListener", "visibilitychange", visibilityChange)
	js.Global().Get("window").Call("addEventListener", "pagehide", pageHide)
	saveLifecycleCallbacks = append(saveLifecycleCallbacks, visibilityChange, pageHide)
}
