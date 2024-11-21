package main

import (
	"github.com/sago35/koebiten"
	"github.com/sago35/koebiten/games/flappygopher/flappygopher"
	"log"
)

func main() {
	koebiten.SetWindowSize(128, 64)
	koebiten.SetWindowTitle("Flappy Gopher")
	koebiten.SetRotate(koebiten.Rotation180)

	game := flappygopher.NewGame()

	if err := koebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
