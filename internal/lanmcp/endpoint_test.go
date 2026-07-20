package lanmcp

import "testing"

func TestEndpointForGateway(t *testing.T) {
	endpoint, err := EndpointForGateway("192.168.1.2")
	if err != nil {
		t.Fatalf("EndpointForGateway error: %v", err)
	}
	if endpoint != "http://192.168.1.2:18080/mcp" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default port and path", input: "http://10.0.0.8", want: "http://10.0.0.8:18080/mcp"},
		{name: "custom local endpoint", input: "https://[fd00::8]:18443/mcp/", want: "https://[fd00::8]:18443/mcp"},
		{name: "localhost mock", input: "http://localhost:18080/mcp", want: "http://localhost:18080/mcp"},
		{name: "public address", input: "https://8.8.8.8/mcp", wantErr: true},
		{name: "hostname", input: "http://gateway.local/mcp", wantErr: true},
		{name: "credentials", input: "http://" + "user:pass@" + "127.0.0.1/mcp", wantErr: true},
		{name: "wrong path", input: "http://127.0.0.1/admin", wantErr: true},
		{name: "query", input: "http://127.0.0.1/mcp?token=x", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeEndpoint(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("NormalizeEndpoint error = %v", err)
			}
			if got != test.want {
				t.Fatalf("NormalizeEndpoint = %q, want %q", got, test.want)
			}
		})
	}
}
