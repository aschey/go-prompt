package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	prompt "github.com/aschey/go-prompt"
)

type RequestContext struct {
	url    *url.URL
	header http.Header
	client *http.Client
}

var ctx *RequestContext

// See https://github.com/eliangcs/http-prompt/blob/master/http_prompt/completion.py
var suggestions = []prompt.Suggest{
	// Command
	{Text: "cd", Description: "Change URL/path"},
	{Text: "exit", Description: "Exit http-prompt"},

	// HTTP Method
	{Text: "delete", Description: "DELETE request"},
	{Text: "get", Description: "GET request"},
	{Text: "patch", Description: "GET request"},
	{Text: "post", Description: "POST request"},
	{Text: "put", Description: "PUT request"},

	// HTTP Header
	{Text: "Accept", Description: "Acceptable response media type"},
	{Text: "Accept-Charset", Description: "Acceptable response charsets"},
	{Text: "Accept-Encoding", Description: "Acceptable response content codings"},
	{Text: "Accept-Language", Description: "Preferred natural languages in response"},
	{Text: "ALPN", Description: "Application-layer protocol negotiation to use"},
	{Text: "Alt-Used", Description: "Alternative host in use"},
	{Text: "Authorization", Description: "Authentication information"},
	{Text: "Cache-Control", Description: "Directives for caches"},
	{Text: "Connection", Description: "Connection options"},
	{Text: "Content-Encoding", Description: "Content codings"},
	{Text: "Content-Language", Description: "Natural languages for content"},
	{Text: "Content-Length", Description: "Anticipated size for payload body"},
	{Text: "Content-Location", Description: "Where content was obtained"},
	{Text: "Content-MD5", Description: "Base64-encoded MD5 sum of content"},
	{Text: "Content-Type", Description: "Content media type"},
	{Text: "Cookie", Description: "Stored cookies"},
	{Text: "Date", Description: "Datetime when message was originated"},
	{Text: "Depth", Description: "Applied only to resource or its members"},
	{Text: "DNT", Description: "Do not track user"},
	{Text: "Expect", Description: "Expected behaviors supported by server"},
	{Text: "Forwarded", Description: "Proxies involved"},
	{Text: "From", Description: "Sender email address"},
	{Text: "Host", Description: "Target URI"},
	{Text: "HTTP2-Settings", Description: "HTTP/2 connection parameters"},
	{Text: "If", Description: "Request condition on state tokens and ETags"},
	{Text: "If-Match", Description: "Request condition on target resource"},
	{Text: "If-Modified-Since", Description: "Request condition on modification date"},
	{Text: "If-None-Match", Description: "Request condition on target resource"},
	{Text: "If-Range", Description: "Request condition on Range"},
	{Text: "If-Schedule-Tag-Match", Description: "Request condition on Schedule-Tag"},
	{Text: "If-Unmodified-Since", Description: "Request condition on modification date"},
	{Text: "Max-Forwards", Description: "Max number of times forwarded by proxies"},
	{Text: "MIME-Version", Description: "Version of MIME protocol"},
	{Text: "Origin", Description: "Origin(s) issuing the request"},
	{Text: "Pragma", Description: "Implementation-specific directives"},
	{Text: "Prefer", Description: "Preferred server behaviors"},
	{Text: "Proxy-Authorization", Description: "Proxy authorization credentials"},
	{Text: "Proxy-Connection", Description: "Proxy connection options"},
	{Text: "Range", Description: "Request transfer of only part of data"},
	{Text: "Referer", Description: "Previous web page"},
	{Text: "TE", Description: "Transfer codings willing to accept"},
	{Text: "Transfer-Encoding", Description: "Transfer codings applied to payload body"},
	{Text: "Upgrade", Description: "Invite server to upgrade to another protocol"},
	{Text: "User-Agent", Description: "User agent string"},
	{Text: "Via", Description: "Intermediate proxies"},
	{Text: "Warning", Description: "Possible incorrectness with payload body"},
	{Text: "WWW-Authenticate", Description: "Authentication scheme"},
	{Text: "X-Csrf-Token", Description: "Prevent cross-site request forgery"},
	{Text: "X-CSRFToken", Description: "Prevent cross-site request forgery"},
	{Text: "X-Forwarded-For", Description: "Originating client IP address"},
	{Text: "X-Forwarded-Host", Description: "Original host requested by client"},
	{Text: "X-Forwarded-Proto", Description: "Originating protocol"},
	{Text: "X-Http-Method-Override", Description: "Request method override"},
	{Text: "X-Requested-With", Description: "Used to identify Ajax requests"},
	{Text: "X-XSRF-TOKEN", Description: "Prevent cross-site request forgery"},
}

func livePrefix() (string, bool) {
	if ctx.url.Path == "/" {
		return "", false
	}
	return ctx.url.String() + "> ", true
}

func executor(in string, suggest *prompt.Suggest, suggestions []prompt.Suggest) {
	in = strings.TrimSpace(in)

	var method, body string
	blocks := strings.Split(in, " ")
	switch blocks[0] {
	case "exit":
		fmt.Println("Bye!")
		os.Exit(0)
	case "cd":
		if len(blocks) < 2 {
			ctx.url.Path = "/"
		} else {
			ctx.url.Path = path.Join(ctx.url.Path, blocks[1])
		}
		return
	case "get", "delete":
		method = strings.ToUpper(blocks[0])
	case "post", "put", "patch":
		if len(blocks) < 2 {
			fmt.Println("please set request body.")
			return
		}
		body = strings.Join(blocks[1:], " ")
		method = strings.ToUpper(blocks[0])
	}
	if method != "" {
		req, err := http.NewRequest(method, ctx.url.String(), strings.NewReader(body))
		if err != nil {
			fmt.Println("err: " + err.Error())
			return
		}
		req.Header = ctx.header
		res, err := ctx.client.Do(req)
		if err != nil {
			fmt.Println("err: " + err.Error())
			return
		}
		result, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("err: " + err.Error())
			return
		}
		fmt.Printf("%s\n", result)
		ctx.header = http.Header{}
		return
	}

	if h := strings.Split(in, ":"); len(h) == 2 {
		// Handling HTTP Header
		ctx.header.Add(strings.TrimSpace(h[0]), strings.Trim(h[1], ` '"`))
	} else {
		fmt.Println("Sorry, I don't understand.")
	}
}

func completer(in prompt.Document, returnChan chan []prompt.Suggest) {
	w := in.GetWordBeforeCursor()
	if w == "" {
		returnChan <- []prompt.Suggest{}
		return
	}
	returnChan <- prompt.FilterHasPrefix(suggestions, w, true)
}

func main() {
	var baseURL = "http://localhost:8000/"
	if len(os.Args) == 2 {
		baseURL = os.Args[1]
		if strings.HasSuffix(baseURL, "/") {
			baseURL += "/"
		}
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		log.Fatal(err)
	}
	ctx = &RequestContext{
		url:    u,
		header: http.Header{},
		client: &http.Client{},
	}

	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix(u.String()+"> "),
		prompt.OptionLivePrefix(livePrefix),
		prompt.OptionTitle("http-prompt"),
	)
	p.Run()
}
