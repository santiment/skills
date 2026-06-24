// Command score is the agent-friendly CLI for the Santiment Score APIs.
package main

import (
	"os"

	"santiment.net/san-skills/internal/app/score"
)

func main() {
	os.Exit(score.Execute())
}
