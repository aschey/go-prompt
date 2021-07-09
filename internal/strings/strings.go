package strings

import (
	"strings"
)

// IndexNotByte is similar with strings.IndexByte but showing the opposite behavior.
func IndexNotByte(s string, c byte) int {
	n := len(s)
	for i := 0; i < n; i++ {
		if s[i] != c {
			return i
		}
	}
	return -1
}

// LastIndexNotByte is similar with strings.LastIndexByte but showing the opposite behavior.
func LastIndexNotByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != c {
			return i
		}
	}
	return -1
}

func IndexAny(s string, seps []string) int {
	first := -1
	for _, sep := range seps {
		curFirst := strings.Index(s, sep) + len(sep) - 1
		if curFirst > -1 && (curFirst < first || first == -1) {
			first = curFirst
		}
	}

	return first
}

// IndexNotAny is similar with strings.IndexAny but showing the opposite behavior.
func IndexNotAny(s string, seps []string) int {
	if len(strings.Join(seps, "")) == 0 {
		return -1
	}
	i := 0
	for i < len(s) {
		found := false
		for _, sep := range seps {
			sub := s[i:]
			if strings.Index(sub, sep) == 0 {
				i += len(sep)
				found = true
				break
			}
		}
		if !found {
			return i
		}

	}
	return -1
}

func LastIndexAny(s string, seps []string) int {
	last := -1
	for _, sep := range seps {
		if sep == "" {
			continue
		}
		curLast := strings.LastIndex(s, sep) + len(sep) - 1
		if curLast > last {
			last = curLast
		}
	}

	return last
}

// LastIndexNotAny is similar with strings.LastIndexAny but showing the opposite behavior.
func LastIndexNotAny(s string, seps []string) int {
	if len(strings.Join(seps, "")) == 0 {
		return -1
	}
	i := 0
	last := -1
	for i < len(s) {
		found := false
		for _, sep := range seps {
			sub := s[i:]
			if strings.Index(sub, sep) == 0 {
				i += len(sep)
				found = true
				break
			}
		}
		if !found {
			last = i
			i += 1
		}

	}
	return last
}
