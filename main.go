package main

import (
	sdk "github.com/dusthoff/hashpoint/plugin/sdk"

	"github.com/DustHoff/hashpoint-plugin-personio-dayoff/internal/plugin"
)

func main() {
	sdk.Serve(plugin.New())
}
