package prompt

import (
	"bytes"
	"time"

	"github.com/aschey/go-prompt/internal/debug"
)

// Executor is called when user input something text.
type Executor func(string, *Suggest, []Suggest)

// ExitChecker is called after user input to check if prompt must stop and exit go-prompt Run loop.
// User input means: selecting/typing an entry, then, if said entry content matches the ExitChecker function criteria:
// - immediate exit (if breakline is false) without executor called
// - exit after typing <return> (meaning breakline is true), and the executor is called first, before exit.
// Exit means exit go-prompt (not the overall Go program)
type ExitChecker func(in string, breakline bool) bool

// Completer should return the suggest item from Document.
type Completer func(Document, chan []Suggest)

// Prompt is core struct of go-prompt.
type Prompt struct {
	in                ConsoleParser
	buf               *Buffer
	renderer          *Render
	executor          Executor
	history           *History
	completion        *CompletionManager
	keyBindings       []KeyBind
	ASCIICodeBindings []ASCIICodeBind
	keyBindMode       KeyBindMode
	completionOnDown  bool
	exitChecker       ExitChecker
	skipTearDown      bool
	statusbarChan     chan string
}

// Exec is the struct contains user input context.
type Exec struct {
	input string
}

// Run starts prompt.
func (p *Prompt) Run() int {
	p.skipTearDown = false
	defer debug.Teardown()
	debug.Log("start prompt")
	p.setUp()
	defer p.tearDown()

	if p.completion.showAtStart {
		p.completion.Update(*p.buf.Document())
	}

	p.renderer.Render(p.buf, p.completion)

	bufCh := make(chan []byte, 128)
	stopReadBufCh := make(chan struct{})
	go p.readBuffer(bufCh, stopReadBufCh)

	exitCh := make(chan int)
	winSizeCh := make(chan *WinSize)
	stopHandleSignalCh := make(chan struct{})
	go p.handleSignals(exitCh, winSizeCh, stopHandleSignalCh)

	defer func() {
		p.renderer.BreakLine(p.buf)
		stopReadBufCh <- struct{}{}
		stopHandleSignalCh <- struct{}{}
	}()

	updating := false
	pending := false

	resultsCh := make(chan []Suggest, 1)
	requestPromptUpdate := func() {
		if !updating {
			p.completion.Completer(*p.buf.Document(), resultsCh)
			updating = true
		} else {
			pending = true
		}

	}

	var lastChosen *Suggest = nil
	for {
		select {
		case b := <-bufCh:
			if shouldExit, e := p.feed(b); shouldExit {
				return 0
			} else if e != nil {
				// Stop goroutine to run readBuffer function
				stopReadBufCh <- struct{}{}
				stopHandleSignalCh <- struct{}{}
				// Unset raw mode
				// Reset to Blocking mode because returned EAGAIN when still set non-blocking mode.
				debug.AssertNoError(p.in.TearDown())

				p.executor(e.input, lastChosen, p.completion.tmp)

				requestPromptUpdate()

				p.renderer.Render(p.buf, p.completion)

				if p.exitChecker != nil && p.exitChecker(e.input, true) {
					p.skipTearDown = true
					return 0
				}
				// Set raw mode
				debug.AssertNoError(p.in.Setup())
				go p.readBuffer(bufCh, stopReadBufCh)
				go p.handleSignals(exitCh, winSizeCh, stopHandleSignalCh)
			} else {
				requestPromptUpdate()
				if p.completion.selected > -1 && p.completion.selected < len(p.completion.tmp) {
					lastChosen = &p.completion.tmp[p.completion.selected]
				} else {
					lastChosen = nil
				}
				p.renderer.Render(p.buf, p.completion)
			}
		case w := <-winSizeCh:
			p.renderer.UpdateWinSize(w)
			requestPromptUpdate()
			p.renderer.Render(p.buf, p.completion)
		case code := <-exitCh:
			return code
		case results := <-resultsCh:
			updating = false
			p.completion.SetResults(results)
			p.renderer.Render(p.buf, p.completion)
			if pending {
				requestPromptUpdate()
				pending = false
			}
		case statusBar := <-p.statusbarChan:
			p.renderer.statusBar = statusBar
			p.renderer.Render(p.buf, p.completion)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (p *Prompt) feed(b []byte) (shouldExit bool, exec *Exec) {
	key := GetKey(b)
	p.buf.lastKeyStroke = key
	// completion
	completing := p.completion.Completing()

	switch key {
	case Enter, ControlJ, ControlM:
		p.handleCompletionKeyBinding(key, completing)
		p.renderer.BreakLine(p.buf)

		exec = &Exec{input: p.buf.Text()}
		p.buf = NewBuffer()
		if exec.input != "" {
			p.history.Add(exec.input)
		}
	case ControlC:
		p.handleCompletionKeyBinding(key, completing)
		p.renderer.BreakLine(p.buf)
		p.buf = NewBuffer()
		p.history.Clear()
	case Up, ControlP:
		if !completing { // Don't use p.completion.Completing() because it takes double operation when switch to selected=-1.
			if newBuf, changed := p.history.Older(p.buf); changed {
				p.buf = newBuf
			} else {
				p.handleCompletionKeyBinding(key, completing)
			}
			return
		}
		p.handleCompletionKeyBinding(key, completing)
	case Down, ControlN:
		if !completing { // Don't use p.completion.Completing() because it takes double operation when switch to selected=-1.
			if newBuf, changed := p.history.Newer(p.buf); changed {
				p.buf = newBuf
			} else {
				p.handleCompletionKeyBinding(key, completing)
			}
			return
		}
		p.handleCompletionKeyBinding(key, completing)

	case ControlD:
		if p.buf.Text() == "" {
			shouldExit = true
			return
		}
	case NotDefined:
		p.handleCompletionKeyBinding(key, completing)
		if p.handleASCIICodeBinding(b) {
			return
		}
		p.buf.InsertText(string(b), false, true)
	default:
		p.history.Reset()
		p.handleCompletionKeyBinding(key, completing)
	}

	shouldExit = p.handleKeyBinding(key)
	return
}

func (p *Prompt) handleCompletionKeyBinding(key Key, completing bool) {
	switch key {
	case Down:
		if completing || p.completionOnDown {
			p.completion.Next()
		}
	case Tab, ControlI:
		p.completion.Next()
	case Up:
		if completing {
			p.completion.Previous()
		}
	case BackTab:
		p.completion.Previous()
	default:
		if s, ok := p.completion.GetSelectedSuggestion(); ok {
			w := p.buf.Document().GetWordBeforeCursorUntilSeparator(p.completion.wordSeparator)
			if w != "" {
				p.buf.DeleteBeforeCursor(len([]rune(w)))
			}
			p.buf.InsertText(s.Text, true, true)
		}
		p.completion.Reset()
	}
}

func (p *Prompt) handleKeyBinding(key Key) bool {
	shouldExit := false
	for i := range commonKeyBindings {
		kb := commonKeyBindings[i]
		if kb.Key == key {
			kb.Fn(p.buf)
		}
	}

	if p.keyBindMode == EmacsKeyBind {
		for i := range emacsKeyBindings {
			kb := emacsKeyBindings[i]
			if kb.Key == key {
				kb.Fn(p.buf)
			}
		}
	}

	// Custom key bindings
	for i := range p.keyBindings {
		kb := p.keyBindings[i]
		if kb.Key == key {
			kb.Fn(p.buf)
		}
	}
	if p.exitChecker != nil && p.exitChecker(p.buf.Text(), false) {
		shouldExit = true
	}
	return shouldExit
}

func (p *Prompt) handleASCIICodeBinding(b []byte) bool {
	checked := false
	for _, kb := range p.ASCIICodeBindings {
		if bytes.Equal(kb.ASCIICode, b) {
			kb.Fn(p.buf)
			checked = true
		}
	}
	return checked
}

func (p *Prompt) readBuffer(bufCh chan []byte, stopCh chan struct{}) {
	debug.Log("start reading buffer")
	for {
		select {
		case <-stopCh:
			debug.Log("stop reading buffer")
			return
		default:
			if b, err := p.in.Read(); err == nil && !(len(b) == 1 && b[0] == 0) {
				bufCh <- b
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (p *Prompt) setUp() {
	debug.AssertNoError(p.in.Setup())
	p.renderer.Setup()
	p.renderer.UpdateWinSize(p.in.GetWinSize())
}

func (p *Prompt) tearDown() {
	if !p.skipTearDown {
		debug.AssertNoError(p.in.TearDown())
	}
	p.renderer.TearDown()
}
