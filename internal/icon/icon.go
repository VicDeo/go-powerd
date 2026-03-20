// icon package draws a tray icon.
package icon

import (
	"bytes"

	"github.com/fogleman/gg"
)

// DrawIcon draws an icon of a battery with the current level and charger status next to it.
func DrawIcon(percent int, charging bool, size float64, buf *bytes.Buffer) []byte {

	digitColor := func(percent int, charging bool) (float64, float64, float64) {
		if charging {
			return 0.2, 0.8, 1.0
		}
		return 1.0, 1.0, 1.0
	}

	powerLevelColor := func(percent int, charging bool) []float64 {
		if charging {
			return []float64{0.2, 0.6, 1.0} // blue when charging
		}
		if percent < 20 {
			return []float64{0.9, 0.2, 0.2} // red when low
		}
		return []float64{0.2, 0.8, 0.2} // green when high
	}

	if buf == nil {
		buf = &bytes.Buffer{}
	}
	buf.Reset()

	dc := gg.NewContext(int(size), int(size))

	// Battery Icon (Left side)
	batH := size * 0.2
	batW := size * 0.85
	xStart := size * 0.02
	yStart := size - batH - 2

	dc.SetRGB(0.8, 0.8, 0.8) // border
	dc.SetLineWidth(size * 0.04)
	dc.DrawRectangle(xStart, yStart, batW, batH)
	dc.Stroke()

	// Fill
	fillColor := powerLevelColor(percent, charging)

	dc.SetRGB(fillColor[0], fillColor[1], fillColor[2])
	dc.DrawRectangle(xStart+2, yStart+2, (float64(percent)/100.0)*(batW-4), batH-4)
	dc.Fill()

	if charging {
		dc.SetRGB(0.95, 0.82, 0.12)
		barY := yStart - 3.0
		dc.DrawRectangle(xStart, barY, batW, 3.0)
		dc.Fill()
	}

	textX := 0.2
	textY := 0.1

	dc.SetRGB(digitColor(percent, charging))
	if percent >= 100 {
		textX = -0.5
		DrawDigit(dc, 1, textX, textY, size*0.5)
		DrawDigit(dc, 0, textX+size*0.34, textY, size*0.5)
		DrawDigit(dc, 0, textX+size*0.66, textY, size*0.5)
	} else {
		DrawDigit(dc, percent/10, textX, textY, size*0.65)
		DrawDigit(dc, percent%10, textX+size*0.5, textY, size*0.65)
	}

	dc.EncodePNG(buf)
	return buf.Bytes()
}

// DrawDigit draws a retro 7 segment digit.
func DrawDigit(dc *gg.Context, val int, x, y, size float64) {
	// Segment definitions (A-G)
	// A: top, B: top-right, C: bottom-right, D: bottom, E: bottom-left, F: top-left, G: middle

	// segmentMask defines which segments are "on" for each digit 0-9
	masks := []uint8{0x3F, 0x06, 0x5B, 0x4F, 0x66, 0x6D, 0x7D, 0x07, 0x7F, 0x6F}
	if val < 0 || val > 9 {
		return
	}
	mask := masks[val]

	// Helper to draw segment based on mask position
	drawSeg := func(bit uint8, dx, dy, width, height float64) {
		if (mask & bit) != 0 {
			dc.DrawRectangle(x+dx*size, y+dy*size, width*size, height*size)
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
	dc.Fill()
}
