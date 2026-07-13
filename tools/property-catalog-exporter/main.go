package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/yeelight/yeelight-home/internal/semantic"
)

func main() {
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(semantic.PropertyCatalog()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "encode property catalog: %v\n", err)
		os.Exit(1)
	}
}
