// icon package draws a tray icon.
package icon

import (
	"bytes"

	"github.com/fogleman/gg"
)

var (
	// segmentMasks defines which segments are "on" for each digit 0-9
	segmentMasks = [...]uint8{0x3F, 0x06, 0x5B, 0x4F, 0x66, 0x6D, 0x7D, 0x07, 0x7F, 0x6F}
)

// Icon is a struct that represents a tray icon.
type Icon struct {
	dc   *gg.Context
	size float64
	buf  *bytes.Buffer
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

// PNG draws an icon of a battery with the current level and charger status next to it.
func (i *Icon) PNG(percent int, charging bool) []byte {
	i.buf.Reset()
	i.dc.SetRGBA(0, 0, 0, 0)
	i.dc.Clear()

	digitColor := func(percent int, charging bool) (float64, float64, float64) {
		if charging {
			return 0.2, 0.8, 1.0
		}
		return 1.0, 1.0, 1.0
	}

	powerLevelColor := func(percent int, charging bool) (float64, float64, float64) {
		if charging {
			return 0.2, 0.6, 1.0 // blue when charging
		}
		if percent < 20 {
			return 0.9, 0.2, 0.2 // red when low
		}
		return 0.2, 0.8, 0.2 // green when high
	}

	// Battery Icon (Left side)
	batH := i.size * 0.2
	batW := i.size * 0.85
	xStart := i.size * 0.02
	yStart := i.size - batH - 2

	i.dc.SetRGB(0.8, 0.8, 0.8) // border
	i.dc.SetLineWidth(i.size * 0.04)
	i.dc.DrawRectangle(xStart, yStart, batW, batH)
	i.dc.Stroke()

	// Fill
	i.dc.SetRGB(powerLevelColor(percent, charging))
	i.dc.DrawRectangle(xStart+2, yStart+2, (float64(percent)/100.0)*(batW-4), batH-4)
	i.dc.Fill()

	if charging {
		i.dc.SetRGB(0.95, 0.82, 0.12)
		barY := yStart - 3.0
		i.dc.DrawRectangle(xStart, barY, batW, 3.0)
		i.dc.Fill()
	}

	textX := 0.2
	textY := 0.1

	i.dc.SetRGB(digitColor(percent, charging))
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

// drawDigit draws a retro 7 segment digit.
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
