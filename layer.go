package main

import (
	"image"
	"net/http"
	"os"

	_ "image/jpeg"
	_ "image/png"
)

type Layer interface {
	XPos() int
	YPos() int
}

type Image struct {
	Path     string
	URL      string
	Position string

	image image.Image
	xPos  int
	yPos  int
}

func (x *Image) XPos() int { return x.xPos }
func (x *Image) YPos() int { return x.yPos }

func (l *Image) Decode() image.Image {
	if l.image != nil {
		return l.image
	}

	// local
	if l.Path != "" {
		file, err := os.Open(l.Path)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			panic(err)
		}
		l.image = img
	}

	// remote
	if l.URL != "" {
		res, err := http.Get(l.URL)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()
		img, _, err := image.Decode(res.Body)
		if err != nil {
			panic(err)
		}
		l.image = img
	}

	return l.image
}

type WaterMask struct {
	Image

	Repeat string
}

type Text struct {
	DPI      float64
	FontPath string
	FontName string
}
