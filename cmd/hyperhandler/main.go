// Command hyperhandler is the agent-friendly CLI for trading on the Hyperliquid DEX.
package main

import (
	"os"

	"santiment.net/san-skills/internal/app/hyperhandler"
)

func main() {
	os.Exit(hyperhandler.Execute())
}
