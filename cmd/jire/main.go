package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/ivab3/jire/internal/app"
)

func main() {
	p := tea.NewProgram(app.New())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "jire: %v\n", err)
		os.Exit(1)
	}
}
