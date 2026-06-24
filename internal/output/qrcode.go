package output

import (
	"bytes"

	qrterminal "github.com/mdp/qrterminal/v3"
	"rsc.io/qr"
)

func RenderQRText(text string) (string, error) {
	var buffer bytes.Buffer
	qrterminal.GenerateWithConfig(text, qrterminal.Config{
		Level:     qrterminal.L,
		Writer:    &buffer,
		BlackChar: "██",
		WhiteChar: "  ",
		QuietZone: 2,
	})
	return buffer.String(), nil
}

func RenderQRPNG(text string) ([]byte, error) {
	code, err := qr.Encode(text, qr.L)
	if err != nil {
		return nil, err
	}
	return code.PNG(), nil
}
