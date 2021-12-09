package main

import (
	"os"
	"os/exec"

	prompt "github.com/aschey/go-prompt"
)

func executor(t string, suggest *prompt.Suggest, suggestions []prompt.Suggest) {
	if t == "bash" {
		cmd := exec.Command("bash")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}

func completer(t prompt.Document, returnChan chan []prompt.Suggest) {
	returnChan <- []prompt.Suggest{
		{Text: "bash"},
	}
}

func main() {
	p := prompt.New(
		executor,
		completer,
	)
	p.Run()
}
