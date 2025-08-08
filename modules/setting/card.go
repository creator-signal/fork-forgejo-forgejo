package setting

import (
	"image/color"
	"regexp"
	"strconv"

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
	loadBackground(sec.Key("BACKGROUND").MustString(""))
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

var rgbPattern = regexp.MustCompile(`(?:rgb\(\s*([0-9]{1,3})\s*,\s*([0-9]{1,3})\s*,\s*([0-9]{1,3}\s*)\))`)

func loadBackground(src string) {
	if src == "" {
		return
	}

	if rgb := rgbPattern.FindStringSubmatch(src); rgb != nil {
		r, errR := strconv.ParseUint(rgb[1], 10, 8)
		if errR != nil {
			log.Error("r parameter to rgb(...) must be in range [0, 255]")
		}
		g, errG := strconv.ParseUint(rgb[2], 10, 8)
		if errG != nil {
			log.Error("g parameter to rgb(...) must be in range [0, 255]")
		}
		b, errB := strconv.ParseUint(rgb[3], 10, 8)
		if errB != nil {
			log.Error("b parameter to rgb(...) must be in range [0, 255]")
		}

		if errR != nil || errG != nil || errB != nil {
			return
		}

		Card.Background.Color = color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}

	log.Error("card.BACKGROUND must be 'rgb(...)'")
}
