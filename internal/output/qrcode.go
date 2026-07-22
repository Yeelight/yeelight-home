package output

import (
	"bytes"
	"fmt"
	"strings"

	qrterminal "github.com/mdp/qrterminal/v3"
	"rsc.io/qr"
)

type QRTextSize string

const (
	QRTextCompact QRTextSize = "compact"
	QRTextNormal  QRTextSize = "normal"
	QRTextLarge   QRTextSize = "large"
)

func RenderQRText(text string) (string, error) {
	return RenderQRTextWithSize(text, QRTextCompact)
}

func RenderQRTextWithSize(text string, size QRTextSize) (string, error) {
	var buffer bytes.Buffer
	config := qrterminal.Config{
		Level:     qrterminal.L,
		Writer:    &buffer,
		BlackChar: "██",
		WhiteChar: "  ",
		QuietZone: 2,
	}
	switch size {
	case QRTextCompact:
		config.HalfBlocks = true
		config.BlackChar = qrterminal.BLACK_BLACK
		config.BlackWhiteChar = qrterminal.BLACK_WHITE
		config.WhiteChar = qrterminal.WHITE_WHITE
		config.WhiteBlackChar = qrterminal.WHITE_BLACK
	case QRTextNormal:
	case QRTextLarge:
	default:
		return "", fmt.Errorf("QR size must be compact, normal, or large")
	}
	qrterminal.GenerateWithConfig(text, config)
	if size == QRTextLarge {
		return scaleQRText(buffer.String(), 2), nil
	}
	return buffer.String(), nil
}

func scaleQRText(rendered string, factor int) string {
	if factor <= 1 {
		return rendered
	}
	var result strings.Builder
	for _, line := range strings.SplitAfter(rendered, "\n") {
		hasNewline := strings.HasSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\n")
		var expanded strings.Builder
		for _, character := range line {
			expanded.WriteString(strings.Repeat(string(character), factor))
		}
		repetitions := factor
		if !hasNewline {
			repetitions = 1
		}
		for index := 0; index < repetitions; index++ {
			result.WriteString(expanded.String())
			if hasNewline {
				result.WriteByte('\n')
			}
		}
	}
	return result.String()
}

func RenderQRPNG(text string) ([]byte, error) {
	code, err := qr.Encode(text, qr.L)
	if err != nil {
		return nil, err
	}
	return code.PNG(), nil
}
