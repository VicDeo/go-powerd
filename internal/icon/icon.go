// Package icon draws a tray icon depending on the battery level and charging status.
package icon

import (
	"bytes"
	"fmt"

	"github.com/VicDeo/go-powerd/internal/config"
	"github.com/fogleman/gg"
)

var (
	// segmentMasks defines which segments are "on" for each digit 0-9
	segmentMasks = [...]uint8{0x3F, 0x06, 0x5B, 0x4F, 0x66, 0x6D, 0x7D, 0x07, 0x7F, 0x6F}
)

// Icon is a struct that represents a tray icon.
type Icon struct {
	dc     *gg.Context
	size   float64
	colors IconColors
	buf    *bytes.Buffer
}

type IconColors struct {
	SegmentsOk       RGBA
	SegmentsLow      RGBA
	SegmentsCharging RGBA
	BarOK            RGBA
	BarLow           RGBA
	BarCharging      RGBA
	Border           RGBA
	Charger          RGBA
}
type RGBA struct {
	R, G, B, A float64
}

// New creates a new Icon instance.
func New(size float64) *Icon {
	n := int(size)
	return &Icon{
		dc:   gg.NewContext(n, n),
		buf:  &bytes.Buffer{},
		size: size,
	}
}

func (i *Icon) SetColors(cfg *config.Colors) {
	i.colors = IconColors{
		SegmentsOk:       hexToRGBA(cfg.SegmentsOk),
		SegmentsLow:      hexToRGBA(cfg.SegmentsLow),
		SegmentsCharging: hexToRGBA(cfg.SegmentsCharging),
		BarOK:            hexToRGBA(cfg.BarOk),
		BarLow:           hexToRGBA(cfg.BarLow),
		BarCharging:      hexToRGBA(cfg.BarCharging),
		Border:           hexToRGBA(cfg.Border),
		Charger:          hexToRGBA(cfg.Charger),
	}
}

// PNG draws an icon of a battery with the current level and charger status.
func (i *Icon) PNG(percent int, charging bool) []byte {
	i.buf.Reset()
	i.dc.SetRGBA(0, 0, 0, 0)
	i.dc.Clear()

	// Battery Icon (Left side)
	batH := i.size * 0.2
	batW := i.size * 0.85
	xStart := i.size * 0.02
	yStart := i.size - batH - 2

	i.dc.SetRGBA(i.colors.Border.R, i.colors.Border.G, i.colors.Border.B, i.colors.Border.A) // border
	i.dc.SetLineWidth(i.size * 0.04)
	i.dc.DrawRectangle(xStart, yStart, batW, batH)
	i.dc.Stroke()

	// Fill
	fillColor := i.powerLevelColor(percent, charging)
	i.dc.SetRGBA(fillColor.R, fillColor.G, fillColor.B, fillColor.A)
	i.dc.DrawRectangle(xStart+2, yStart+2, (float64(percent)/100.0)*(batW-4), batH-4)
	i.dc.Fill()

	if charging {
		i.dc.SetRGBA(i.colors.Charger.R, i.colors.Charger.G, i.colors.Charger.B, i.colors.Charger.A)
		barY := yStart - 3.0
		i.dc.DrawRectangle(xStart, barY, batW, 3.0)
		i.dc.Fill()
	}

	textX := 0.2
	textY := 0.1

	segmentColor := i.digitColor(percent, charging)
	i.dc.SetRGBA(segmentColor.R, segmentColor.G, segmentColor.B, segmentColor.A)
	if percent >= 100 {
		textX = -0.5
		i.drawDigit(1, textX, textY, i.size*0.5)
		i.drawDigit(0, textX+i.size*0.34, textY, i.size*0.5)
		i.drawDigit(0, textX+i.size*0.66, textY, i.size*0.5)
	} else {
		i.drawDigit(percent/10, textX, textY, i.size*0.65)
		i.drawDigit(percent%10, textX+i.size*0.5, textY, i.size*0.65)
	}

	i.dc.EncodePNG(i.buf)
	return i.buf.Bytes()
}

func (i *Icon) digitColor(percent int, charging bool) RGBA {
	if charging {
		return i.colors.SegmentsCharging
	}
	if percent < 20 {
		return i.colors.SegmentsLow
	}
	return i.colors.SegmentsOk
}

func (i *Icon) powerLevelColor(percent int, charging bool) RGBA {
	if charging {
		return i.colors.BarCharging
	}
	if percent < 20 {
		return i.colors.BarLow
	}
	return i.colors.BarOK
}

// drawDigit draws a retro 7-segment digit.
func (i *Icon) drawDigit(val int, x, y, size float64) {
	// Segment definitions (A-G)
	// A: top, B: top-right, C: bottom-right, D: bottom, E: bottom-left, F: top-left, G: middle

	if val < 0 || val > 9 {
		return
	}
	mask := segmentMasks[val]

	// Helper to draw segment based on mask position
	drawSeg := func(bit uint8, dx, dy, width, height float64) {
		if (mask & bit) != 0 {
			i.dc.DrawRectangle(x+dx*size, y+dy*size, width*size, height*size)
		}
	}

	// We use a small gap (0.02) to create that "disconnected" watch look
	gap := 0.02
	drawSeg(0x01, 0.1+gap, 0, 0.4, 0.1)       // A
	drawSeg(0x02, 0.5+gap, 0.1+gap, 0.1, 0.3) // B
	drawSeg(0x04, 0.5+gap, 0.5+gap, 0.1, 0.3) // C
	drawSeg(0x08, 0.1+gap, 0.8+gap, 0.4, 0.1) // D
	drawSeg(0x10, 0, 0.5+gap, 0.1, 0.3)       // E
	drawSeg(0x20, 0, 0.1+gap, 0.1, 0.3)       // F
	drawSeg(0x40, 0.1+gap, 0.4+gap, 0.4, 0.1) // G
	i.dc.Fill()
}

func hexToRGBA(hex string) RGBA {
	var r, g, b, a int
	fmt.Sscanf(hex, "#%02x%02x%02x%02x", &r, &g, &b, &a)
	return RGBA{
		R: float64(r) / 255.0,
		G: float64(g) / 255.0,
		B: float64(b) / 255.0,
		A: float64(a) / 255.0,
	}
}
