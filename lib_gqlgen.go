/*
Copyright (c) 2020 gqlgen authors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package main

import (
	"strings"
	"unicode"
)

func ToGo(name string) string {
	if name == "_" {
		return "_"
	}
	runes := make([]rune, 0, len(name))

	wordWalker(name, func(info *wordInfo) {
		word := info.Word
		if info.MatchCommonInitial {
			word = strings.ToUpper(word)
		} else if !info.HasCommonInitial {
			if strings.ToUpper(word) == word || strings.ToLower(word) == word {
				// FOO or foo → Foo
				// FOo → FOo
				word = UcFirst(strings.ToLower(word))
			}
		}
		runes = append(runes, []rune(word)...)
	})

	return string(runes)
}

func UcFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func isDelimiter(c rune) bool {
	return c == '-' || c == '_' || unicode.IsSpace(c)
}

type wordInfo struct {
	Word               string
	MatchCommonInitial bool
	HasCommonInitial   bool
}

// This function is based on the following code.
// https://github.com/golang/lint/blob/06c8688daad7faa9da5a0c2f163a3d14aac986ca/lint.go#L679
func wordWalker(str string, f func(*wordInfo)) {
	runes := []rune(strings.TrimFunc(str, isDelimiter))
	w, i := 0, 0 // index of start of word, scan
	hasCommonInitial := false
	for i+1 <= len(runes) {
		eow := false // whether we hit the end of a word
		switch {
		case i+1 == len(runes):
			eow = true
		case isDelimiter(runes[i+1]):
			// underscore; shift the remainder forward over any run of underscores
			eow = true
			n := 1
			for i+n+1 < len(runes) && isDelimiter(runes[i+n+1]) {
				n++
			}

			// Leave at most one underscore if the underscore is between two digits
			if i+n+1 < len(runes) && unicode.IsDigit(runes[i]) && unicode.IsDigit(runes[i+n+1]) {
				n--
			}

			copy(runes[i+1:], runes[i+n+1:])
			runes = runes[:len(runes)-n]
		case unicode.IsLower(runes[i]) && !unicode.IsLower(runes[i+1]):
			// lower->non-lower
			eow = true
		}
		i++

		// [w,i) is a word.
		word := string(runes[w:i])
		if !eow && commonInitialisms[word] && !unicode.IsLower(runes[i]) {
			// through
			// split IDFoo → ID, Foo
			// but URLs → URLs
		} else if !eow {
			if commonInitialisms[word] {
				hasCommonInitial = true
			}
			continue
		}

		matchCommonInitial := false
		if commonInitialisms[strings.ToUpper(word)] {
			hasCommonInitial = true
			matchCommonInitial = true
		}

		f(&wordInfo{
			Word:               word,
			MatchCommonInitial: matchCommonInitial,
			HasCommonInitial:   hasCommonInitial,
		})
		hasCommonInitial = false
		w = i
	}
}

// commonInitialisms is a set of common initialisms.
// Only add entries that are highly unlikely to be non-initialisms.
// For instance, "ID" is fine (Freudian code is rare), but "AND" is not.
var commonInitialisms = map[string]bool{
	"ACL":   true,
	"API":   true,
	"ASCII": true,
	"CPU":   true,
	"CSS":   true,
	"CSV":   true,
	"DNS":   true,
	"EOF":   true,
	"GUID":  true,
	"HTML":  true,
	"HTTP":  true,
	"HTTPS": true,
	"ICMP":  true,
	"ID":    true,
	"IP":    true,
	"JSON":  true,
	"KVK":   true,
	"LHS":   true,
	"PDF":   true,
	"PGP":   true,
	"QPS":   true,
	"QR":    true,
	"RAM":   true,
	"RHS":   true,
	"RPC":   true,
	"SLA":   true,
	"SMTP":  true,
	"SQL":   true,
	"SSH":   true,
	"SVG":   true,
	"TCP":   true,
	"TLS":   true,
	"TTL":   true,
	"UDP":   true,
	"UI":    true,
	"UID":   true,
	"URI":   true,
	"URL":   true,
	"UTF8":  true,
	"UUID":  true,
	"VM":    true,
	"XML":   true,
	"XMPP":  true,
	"XSRF":  true,
	"XSS":   true,
}
