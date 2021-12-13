package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/image/draw"
)

// 1. 如果接受参数只有配置文件，程序不自动关闭，监听拖拽窗口文件参数

type config struct {
	Layers      []Layer `mapstructure:"layers"`
	Format      string  `mapstructure:"format"`
	Resize      string  `mapstructure:"resize"`
	ResizeScale string  `mapstructure:"resize-scale"`
	Quality     int     `mapstructure:"quality"`

	// flags
	Input             string `mapstructure:"input"`
	Watermark         string `mapstructure:"watermark"`
	WatermarkPosition string `mapstructure:"watermark-position"`
	WatermarkRepeat   string `mapstructure:"watermark-repeat"`
	Output            string `mapstructure:"output"`
}

var C config

func main() {
	// testResize()
	parseFromConfig := false
	viper.SetDefault("resize", "BiLinear")
	viper.SetDefault("resize-scale", "0.5")
	viper.SetDefault("quality", 85)
	err := viperParseConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			err = viperParseFlags()
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	} else {
		parseFromConfig = true
	}

	err = viper.Unmarshal(&C)
	if err != nil {
		log.Fatal(err)
	}

	if parseFromConfig {
	} else {
		if _, err := os.Stat(C.Input); err != nil {
			panic(err)
		}

		if C.Watermark != "" {
			if _, err := os.Stat(C.Watermark); err != nil {
				panic(err)
			}
		}

		if len(C.Layers) == 0 {
			// simplest usage
			C.Layers = append(C.Layers, &Image{
				Path: C.Input,
			})

			if C.Watermark != "" {
				C.Layers = append(C.Layers, &WaterMask{
					Image: Image{
						Path:     C.Watermark,
						Position: C.WatermarkPosition,
					},
					Repeat: C.WatermarkRepeat,
				})
			}
		}
	}

	fmt.Println(C)

	populate()
}

func populate() {
	// input layer
	i := C.Layers[0].(*Image)
	inputLayer := i.Decode()

	// base layer
	var baseLayer = image.NewRGBA(inputLayer.Bounds())

	if C.Format == "jpeg" || C.Format == "jpg" {
		draw.Draw(baseLayer, baseLayer.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	}

	for _, clayer := range C.Layers {
		switch l := clayer.(type) {
		case *Image:
			img := l.Decode()
			draw.Draw(baseLayer, img.Bounds(), img, img.Bounds().Min, draw.Over)
		case *WaterMask:
			img := l.Decode()
			bounds := img.Bounds()

			if l.Position != "" {
				var point = image.Point{}
				rect := baseLayer.Bounds()

				p := strings.Split(l.Position, " ")

				x, err := strconv.Atoi(strings.TrimSuffix(p[0], "%"))
				if err != nil {
					panic(err)
				}
				y, err := strconv.Atoi(strings.TrimSuffix(p[1], "%"))
				if err != nil {
					panic(err)
				}

				point.X = (rect.Max.X - bounds.Max.X) * x / 100
				point.Y = (rect.Max.Y - bounds.Max.Y) * y / 100

				bounds = bounds.Add(point)
			}

			if l.Repeat != "" {
				leftRemain := bounds.Min.X
				rightRemain := baseLayer.Bounds().Max.X - leftRemain - img.Bounds().Dx()
				topRemain := bounds.Min.Y
				bottomRemain := baseLayer.Bounds().Max.Y - topRemain - img.Bounds().Dy()
				lc := int(math.Ceil(float64(leftRemain) / float64(bounds.Dx())))
				rc := int(math.Ceil(float64(rightRemain) / float64(bounds.Dx())))
				tc := int(math.Ceil(float64(topRemain) / float64(bounds.Dy())))
				bc := int(math.Ceil(float64(bottomRemain) / float64(bounds.Dy())))

				if l.Repeat == "repeat-x" {
					for i := 0; i < lc; i++ {
						draw.Draw(baseLayer, bounds.Sub(image.Point{X: bounds.Dx() * (i + 1)}), img, img.Bounds().Min, draw.Over)
					}
					for i := 0; i < rc; i++ {
						draw.Draw(baseLayer, bounds.Add(image.Point{X: bounds.Dx() * (i + 1)}), img, img.Bounds().Min, draw.Over)
					}
				}
				if l.Repeat == "repeat-y" {
					for i := 0; i < tc; i++ {
						draw.Draw(baseLayer, bounds.Sub(image.Point{Y: bounds.Dy() * (i + 1)}), img, img.Bounds().Min, draw.Over)
					}
					for i := 0; i < bc; i++ {
						draw.Draw(baseLayer, bounds.Add(image.Point{Y: bounds.Dy() * (i + 1)}), img, img.Bounds().Min, draw.Over)
					}
				}
				if l.Repeat == "repeat" {
					b := bounds
					for i := 0; i < lc; i++ {
						for j := 0; j < tc; j++ {
							draw.Draw(baseLayer, b.Sub(image.Point{Y: bounds.Dy() * (j + 1)}), img, img.Bounds().Min, draw.Over)
						}
						for k := 0; k < bc; k++ {
							draw.Draw(baseLayer, b.Add(image.Point{Y: bounds.Dy() * (k + 1)}), img, img.Bounds().Min, draw.Over)
						}

						b = b.Sub(image.Point{X: bounds.Dx()})
						draw.Draw(baseLayer, b, img, img.Bounds().Min, draw.Over)
					}
					b = bounds
					for i := 0; i < rc; i++ {
						b = b.Add(image.Point{X: bounds.Dx()})
						draw.Draw(baseLayer, b, img, img.Bounds().Min, draw.Over)
						for j := 0; j < tc; j++ {
							draw.Draw(baseLayer, b.Sub(image.Point{Y: bounds.Dy() * (j + 1)}), img, img.Bounds().Min, draw.Over)
						}
						for k := 0; k < bc; k++ {
							draw.Draw(baseLayer, b.Add(image.Point{Y: bounds.Dy() * (k + 1)}), img, img.Bounds().Min, draw.Over)
						}
					}
				}
			}

			draw.Draw(baseLayer, bounds, img, img.Bounds().Min, draw.Over)
		}
	}

	if C.Resize != "" {
		var scaler draw.Scaler

		switch C.Resize {
		case "ApproxBiLinear":
			scaler = draw.ApproxBiLinear
		case "BiLinear":
			scaler = draw.BiLinear
		case "CatmullRom":
			scaler = draw.CatmullRom
		default:
			scaler = draw.NearestNeighbor
		}

		scale, err := strconv.ParseFloat(C.ResizeScale, 64)
		if err != nil {
			panic(err)
		}
		x1, y1 := float64(baseLayer.Bounds().Max.X)*scale, float64(baseLayer.Bounds().Max.Y)*scale
		rz := image.Rect(0, 0, int(x1), int(y1))
		// resize time
		{
			now := time.Now()
			dst := image.NewRGBA(rz)
			scaler.Scale(dst, rz, baseLayer, baseLayer.Bounds(), draw.Over, nil)
			baseLayer = dst
			fmt.Printf("scaling using %q takes %v time\n", C.Resize, time.Since(now))
		}
	}

	out, err := os.Create("output." + C.Format)
	if err != nil {
		panic(err)
	}

	switch C.Format {
	case "jpeg", "jpg":
		o := jpeg.Options{
			Quality: C.Quality,
		}
		if err := jpeg.Encode(out, baseLayer, &o); err != nil {
			panic(err)
		}
	case "png":
		if err := png.Encode(out, baseLayer); err != nil {
			panic(err)
		}
	default:
		panic("unknown output format")
	}
}

func viperParseConfig() error {
	viper.AutomaticEnv()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

func viperParseFlags() error {
	pflag.StringP("input", "i", "", "input file name")
	pflag.StringP("watermark", "m", "", "watermark file name")
	pflag.StringP("watermark-position", "p", "center", "watermark position")
	pflag.StringP("watermark-repeat", "r", "no-repeat", "watermark repeat")
	pflag.StringP("output", "o", "", "output file name")
	pflag.StringP("format", "f", "", "output format")
	pflag.String("resize", "BiLinear", "resize method: BiLinear ApproxBiLinear NearestNeighbor CatmullRom")
	pflag.String("resize-scale", "0.5", "resize scale")
	pflag.IntP("quality", "q", 85, "jpeg output quality")
	pflag.Parse()

	viper.SetDefault("format", "png")
	viper.SetDefault("resize", "BiLinear")
	viper.SetDefault("resize-scale", "0.5")
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return err
	}

	return nil
}
