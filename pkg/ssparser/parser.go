package ssparser

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type tokenizer struct {
	tokenMatchers  []*tokenMatcher
	ignoreMatchers []*tokenMatcher
}

type tokenMatcher struct {
	pattern  string
	re       *regexp.Regexp
	tokenTyp int
	verifier func(string) error
}

type token struct {
	typ   int
	value string
}

func (t *tokenizer) addNil(pattern string, token int) {
	rxp := regexp.MustCompile(pattern)
	matcher := &tokenMatcher{pattern, rxp, token, nil}
	t.tokenMatchers = append(t.tokenMatchers, matcher)
}

func (t *tokenizer) add(pattern string, token int, verifier func(string) error) {
	rxp := regexp.MustCompile(pattern)
	matcher := &tokenMatcher{pattern, rxp, token, verifier}
	t.tokenMatchers = append(t.tokenMatchers, matcher)
}

func (t *tokenizer) ignore(pattern string, token int) {
	rxp := regexp.MustCompile(pattern)
	matcher := &tokenMatcher{pattern, rxp, token, nil}
	t.ignoreMatchers = append(t.ignoreMatchers, matcher)
}

func (t *tokenizer) tokenizeBytes(target []byte) ([]*token, error) {
	result := make([]*token, 0)
	match := true // false when no match is found
	for len(target) > 0 && match {
		match = false
		for _, m := range t.tokenMatchers {
			tok := m.re.Find(target)
			if len(tok) > 0 {
				if m.verifier != nil {
					if err := m.verifier(string(tok)); err != nil {
						return nil, fmt.Errorf("found invalid token: %s with type: %v, err: %v", string(tok), m.tokenTyp, err)
					}
				}
				parsed := token{value: string(tok), typ: m.tokenTyp}
				result = append(result, &parsed)
				target = target[len(tok):] // remove the token from the input
				match = true
				break
			}
		}
		for _, m := range t.ignoreMatchers {
			tok := m.re.Find(target)
			if len(tok) > 0 {
				match = true
				target = target[len(tok):] // remove the token from the input
				break
			}
		}
	}

	if len(target) > 0 && !match {
		return result, fmt.Errorf("no matching token for %s", string(target))
	}

	return result, nil
}

func (t *tokenizer) tokenize(target string) ([]*token, error) {
	return t.tokenizeBytes([]byte(target))
}

const (
	tokenWhitespace int = iota
	tokenString
	tokenLiteral
	tokenVariable
)

func shellStringTokenizer() *tokenizer {
	t := tokenizer{}
	t.addNil("^'(''|[^'])*'", tokenString)
	t.addNil("^[a-zA-Z0-9!@#%^&*()_+\\-=\\[\\];:'\"\\\\|,.<>\\/?]*", tokenLiteral)
	t.add("^\\${?[a-zA-Z_][a-zA-Z0-9_]*}?", tokenVariable, func(s string) error {
		if s[:2] == "${" {
			if s[len(s)-1:] != "}" {
				return fmt.Errorf("expected }")
			}
		}
		return nil
	})
	t.ignore("^ ", tokenWhitespace)

	return &t
}

// Parse the shell string that contains os env.
func Parse(s string) (string, error) {
	tokens, err := shellStringTokenizer().tokenize(s)
	if err != nil {
		return "", err
	}
	var values []string
	for _, tok := range tokens {
		switch tok.typ {
		case tokenVariable:
			val := tok.value[1:] // remove $
			if val[:1] == "{" {
				val = val[1 : len(val)-1]
			}
			values = append(values, os.Getenv(val))
		case tokenLiteral:
			values = append(values, tok.value)
		case tokenString:
			// remove ' from head and tail
			val := tok.value[1 : len(tok.value)-1]
			values = append(values, val)
		default:
			values = append(values, tok.value)
		}
	}
	return strings.Join(values, ""), nil
}
