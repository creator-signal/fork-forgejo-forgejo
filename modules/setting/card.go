package setting

import (
	"image/color"

	"forgejo.org/modules/log"
)

var Card = struct {
	Background struct {
		Color color.Color
	}
	Text struct {
		PrimaryColor   color.Color
		SecondaryColor color.Color
	}
}{
	Background: struct {
		Color color.Color
	}{
		Color: color.White,
	},
	Text: struct {
		PrimaryColor   color.Color
		SecondaryColor color.Color
	}{
		PrimaryColor:   color.Black,
		SecondaryColor: color.Gray{128},
	},
}

func loadCardFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("card")
	loadTheme(sec.Key("THEME").MustString("light"))
}

func loadTheme(theme string) {
	if theme == "dark" {
		Card.Background.Color = color.RGBA{23, 30, 38, 255}
		Card.Text.PrimaryColor = color.White
		Card.Text.SecondaryColor = color.Gray{164}
		return
	}

	if theme != "light" {
		log.Error("card.THEME must be either 'light' or 'dark'")
	}

	Card.Background.Color = color.White
	Card.Text.PrimaryColor = color.Black
	Card.Text.SecondaryColor = color.Gray{128}
}
