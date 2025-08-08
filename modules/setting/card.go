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
	loadTextPrimary(sec.Key("TEXT_PRIMARY").MustString(""))
	loadTextSecondary(sec.Key("TEXT_SECONDARY").MustString(""))
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

func loadBackground(src string) {
	if src == "" {
		return
	}

	color, err := tryColorFromRgb(src)
	if err != nil {
		log.Error("failed to process card.BACKGROUND rgb(...): %v", err)
		return
	}

	if color != nil {
		Card.Background.Color = color
		return
	}

	log.Error("card.BACKGROUND must be 'rgb(...)'")
}

func loadTextPrimary(src string) {
	if src == "" {
		return
	}

	color, err := tryColorFromRgb(src)
	if err != nil {
		log.Error("failed to process card.TEXT_PRIMARY rgb(...): %v", err)
		return
	}

	if color != nil {
		Card.Text.PrimaryColor = color
		return
	}

	log.Error("card.TEXT_PRIMARY must be 'rgb(...)'")
}

func loadTextSecondary(src string) {
	if src == "" {
		return
	}

	color, err := tryColorFromRgb(src)
	if err != nil {
		log.Error("failed to process card.TEXT_SECONDARY rgb(...): %v", err)
		return
	}

	if color != nil {
		Card.Text.SecondaryColor = color
		return
	}

	log.Error("card.TEXT_SECONDARY must be 'rgb(...)'")
}

var rgbPattern = regexp.MustCompile(`(?:rgb\(\s*([0-9]{1,3})\s*,\s*([0-9]{1,3})\s*,\s*([0-9]{1,3}\s*)\))`)

func tryColorFromRgb(src string) (color.Color, error) {
	if rgb := rgbPattern.FindStringSubmatch(src); rgb != nil {
		r, err := strconv.ParseUint(rgb[1], 10, 8)
		if err != nil {
			return nil, err
		}
		g, err := strconv.ParseUint(rgb[2], 10, 8)
		if err != nil {
			return nil, err
		}
		b, err := strconv.ParseUint(rgb[3], 10, 8)
		if err != nil {
			return nil, err
		}

		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}, nil
	}

	return nil, nil
}
