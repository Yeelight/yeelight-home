package output

import (
	"strings"
	"testing"
)

func TestRenderQRTextReturnsTerminalQRCode(t *testing.T) {
	rendered, err := RenderQRText("dali&F8:24:41:00:00:01&qr-1")
	if err != nil {
		t.Fatalf("RenderQRText error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(rendered), "\n")
	if len(lines) < 20 {
		t.Fatalf("expected terminal QR to have many lines, got %d: %q", len(lines), rendered)
	}
	if !strings.Contains(rendered, "██") {
		t.Fatalf("expected rendered QR blocks, got %q", rendered)
	}
	if strings.Contains(rendered, "qr-1") {
		t.Fatalf("rendered QR should not include raw payload text: %q", rendered)
	}
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
