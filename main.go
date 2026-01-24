// Copyright (C) 2026 Murilo Gomes Julio
// SPDX-License-Identifier: GPL-2.0-only

// Site: https://mugomes.github.io

package main

import (
	"image/color"
	"log"
	"math"

	"github.com/gordonklaus/portaudio"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mjibson/go-dsp/fft"
)

const (
	sampleRate = 44100
	bufferSize = 1024

	screenW = 200
	screenH = 50
)

var (
	audioBuffer = make([]float32, bufferSize) // ✅ float32 (CORRETO)
	spectrum    = make([]float64, bufferSize/2)
	smoothed    = make([]float64, bufferSize/2)
	stream      *portaudio.Stream
)

// ================= GAME =================

type Game struct{}

var timeTick float64

func (g *Game) Update() error {
	timeTick += 0.02
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 0}) // importante p/ transparência
	drawLinearSpectrum(screen)
	calcFFT()
}

func (g *Game) Layout(_, _ int) (int, int) {
	return screenW, screenH
}

// ================= MAIN =================
func main() {

	// -------- PortAudio --------
	portaudio.Initialize()
	defer portaudio.Terminate()

	var err error
	// 	params := portaudio.StreamParameters{
	//     Input: portaudio.StreamDeviceParameters{
	//         Device:   inputDevice,
	//         Channels: 1,
	//         Latency:  inputDevice.DefaultLowInputLatency,
	//     },
	//     SampleRate:      sampleRate,
	//     FramesPerBuffer: len(audioBuffer),
	// }

	// stream, err = portaudio.OpenStream(params, audioBuffer)
	//
	stream, err = portaudio.OpenDefaultStream(
		1, 0,
		sampleRate,
		len(audioBuffer),
		audioBuffer, // ✅ float32
	)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	stream.Start()
	defer stream.Stop()

	go func() {
		for {
			stream.Read()
		}
	}()

	// -------- Ebiten --------
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Espectro Circular de Áudio")

	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

// ================= AUDIO =================

func calcFFT() {
	// converte para float64 só para FFT
	windowed := make([]float64, len(audioBuffer))

	for i := range audioBuffer {
		w := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(len(audioBuffer)))
		windowed[i] = float64(audioBuffer[i]) * w
	}

	result := fft.FFTReal(windowed)

	for i := 0; i < len(spectrum); i++ {
		re := real(result[i])
		im := imag(result[i])
		mag := math.Sqrt(re*re + im*im)

		// noise floor
		if mag < 2 {
			mag *= 0.05
		}

		// compressão logarítmica
		mag = math.Log10(mag + 1)

		spectrum[i] = mag
	}

	// suavização (ataque rápido, queda lenta)
	for i := range spectrum {
		if spectrum[i] > smoothed[i] {
			smoothed[i] += (spectrum[i] - smoothed[i]) * 0.35
		} else {
			smoothed[i] *= 0.92
		}
	}
}

// ================= RENDER =================
func drawLinearSpectrum(screen *ebiten.Image) {
	numBars := len(smoothed)
	maxHeight := float64(screenH)
	barSpacing := 1.0
	barWidth := math.Max(float64(screenW)/float64(numBars)-barSpacing, 1)

	// cria 1x1 pixel branco
	barImg := ebiten.NewImage(1, 1)
	barImg.Fill(color.White)

	for i := 0; i < numBars; i++ {
		mag := smoothed[i]
		mag = math.Min(mag, 1.0)
		mag = math.Pow(mag, 0.6)

		height := math.Max(mag*maxHeight, 1)
		x := float64(i)*(barWidth+barSpacing)
		y := float64(screenH) - height

		hue := float64(i)/float64(numBars)*360 + timeTick*40
		col := hsv(hue, 0.9, 1)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(barWidth, height)
		op.GeoM.Translate(x, y)

		// ✅ Forma correta com colorm.ColorM
		op.ColorM.Scale(
            float64(col.R)/255, 
            float64(col.G)/255, 
            float64(col.B)/255, 
            float64(col.A)/255,
        )

		screen.DrawImage(barImg, op)
	}
}



func hsv(h, s, v float64) color.RGBA {
	h = math.Mod(h, 360)
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c

	var r, g, b float64

	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 220,
	}
}
