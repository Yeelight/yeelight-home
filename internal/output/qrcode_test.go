package output

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRenderQRTextReturnsTerminalQRCode(t *testing.T) {
	rendered, err := RenderQRText("dali&F8:24:41:00:00:01&qr-1")
	if err != nil {
		t.Fatalf("RenderQRText error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(rendered), "\n")
	if len(lines) < 10 {
		t.Fatalf("expected terminal QR to have many lines, got %d: %q", len(lines), rendered)
	}
	if !strings.Contains(rendered, "█") {
		t.Fatalf("expected rendered QR blocks, got %q", rendered)
	}
	if strings.Contains(rendered, "qr-1") {
		t.Fatalf("rendered QR should not include raw payload text: %q", rendered)
	}
}

func TestRenderQRTextDefaultsToCompactAndSupportsLargerFallbacks(t *testing.T) {
	compact, err := RenderQRTextWithSize("cli&F8:24:41:00:00:01&qr-1", QRTextCompact)
	if err != nil {
		t.Fatalf("compact QR error: %v", err)
	}
	normal, err := RenderQRTextWithSize("cli&F8:24:41:00:00:01&qr-1", QRTextNormal)
	if err != nil {
		t.Fatalf("normal QR error: %v", err)
	}
	if RenderedQRHeight(compact) >= RenderedQRHeight(normal) {
		t.Fatalf("compact QR height = %d, normal = %d", RenderedQRHeight(compact), RenderedQRHeight(normal))
	}
	if RenderedQRWidth(compact) >= RenderedQRWidth(normal) {
		t.Fatalf("compact QR width = %d, normal = %d", RenderedQRWidth(compact), RenderedQRWidth(normal))
	}
	large, err := RenderQRTextWithSize("cli&F8:24:41:00:00:01&qr-1", QRTextLarge)
	if err != nil || RenderedQRHeight(large) <= RenderedQRHeight(normal) || RenderedQRWidth(large) <= RenderedQRWidth(normal) {
		t.Fatalf("large QR was not enlarged: err=%v compact=%dx%d normal=%dx%d large=%dx%d", err, RenderedQRWidth(compact), RenderedQRHeight(compact), RenderedQRWidth(normal), RenderedQRHeight(normal), RenderedQRWidth(large), RenderedQRHeight(large))
	}
	defaultRendered, err := RenderQRText("cli&F8:24:41:00:00:01&qr-1")
	if err != nil || defaultRendered != compact {
		t.Fatalf("default QR is not compact: err=%v", err)
	}
}

func RenderedQRHeight(rendered string) int {
	return len(strings.Split(strings.TrimSpace(rendered), "\n"))
}

func RenderedQRWidth(rendered string) int {
	width := 0
	for _, line := range strings.Split(strings.TrimSpace(rendered), "\n") {
		width = max(width, utf8.RuneCountInString(line))
	}
	return width
}

func TestRenderQRPNGReturnsPNGBytes(t *testing.T) {
	data, err := RenderQRPNG("cli&F8:24:41:00:00:01&qr-1")
	if err != nil {
		t.Fatalf("RenderQRPNG error: %v", err)
	}
	if len(data) < 100 {
		t.Fatalf("png is too small: %d bytes", len(data))
	}
	if string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("not a png header: %q", data[:8])
	}
}
