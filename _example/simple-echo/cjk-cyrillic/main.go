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
		{Text: "こんにちは", Description: "'こんにちは' means 'Hello' in Japanese"},
		{Text: "감사합니다", Description: "'안녕하세요' means 'Hello' in Korean."},
		{Text: "您好", Description: "'您好' means 'Hello' in Chinese."},
		{Text: "Добрый день", Description: "'Добрый день' means 'Hello' in Russian."},
	}
	returnChan <- prompt.FilterHasPrefix(s, in.GetWordBeforeCursor(), true)
}

func main() {
	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix(">>> "),
		prompt.OptionTitle("sql-prompt for multi width characters"),
	)
	p.Run()
}
