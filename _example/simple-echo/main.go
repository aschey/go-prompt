package main

import (
	"fmt"

	prompt "github.com/aschey/go-prompt"
)

func executor(in string, suggest *prompt.Suggest, suggestions []prompt.Suggest) {
	fmt.Println("Your input: " + in)
}

func completer(in prompt.Document, returnChan chan []prompt.Suggest) {
	s := []prompt.Suggest{
		{Text: "users", Description: "Store the username and age"},
		{Text: "articles", Description: "Store the article text posted by user"},
		{Text: "comments", Description: "Store the text commented to articles"},
		{Text: "groups", Description: "Combine users with specific rules"},
	}
	returnChan <- prompt.FilterHasPrefix(s, in.GetWordBeforeCursor(), true)
}

func main() {
	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix(">>> "),
		prompt.OptionTitle("sql-prompt"),
	)
	p.Run()
}
