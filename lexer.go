package gdpgen

import (
	"regexp"
)

type Lexer interface {
	GetReader(string)
	GetNextToken() Token
	AddPattern(string, string)
	GetCurrentPosition() (int, int)
}

type Token struct {
	Name  string
	Value string
}

type pattern struct {
	name  string
	regex *regexp.Regexp
}

func newPattern(name string, regexPattern string) pattern {
	r, err := regexp.Compile("^" + regexPattern)
	if err != nil {
		panic(err)
	}
	return pattern{name, r}
}

type RegexLexer struct {
	chars    []rune
	patterns []pattern
	pos      int
	line     int
	column   int
}

func (l *RegexLexer) GetReader(s string) {
	l.chars = []rune(s)
}

func (l *RegexLexer) AddPattern(name, pattern string) {
	for _, p := range l.patterns {
		if p.name == name {
			return
		}
	}

	l.patterns = append(l.patterns, newPattern(name, pattern))
}

func (l *RegexLexer) GetNextToken() Token {
	for i, c := range l.chars {
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			if c == '\n' {
				l.line++
				l.column = 1
			} else {
				l.column++
			}
			continue
		}
		for _, p := range l.patterns {
			mRange := p.regex.FindStringIndex(string(l.chars[i:]))
			if mRange != nil {
				l.column = l.column + (mRange[1] - mRange[0])
				value := l.chars[i+mRange[0] : i+mRange[1]]
				if mRange[1] < len(l.chars) {
					l.pos = l.pos + i + mRange[1]
					l.chars = l.chars[i+mRange[1]:]
				} else {
					l.pos = l.pos + len(l.chars)
					l.chars = []rune{}
				}
				return Token{p.name, string(value)}
			}
		}
	}
	l.pos = l.pos + len(l.chars)
	l.chars = []rune{}
	return Token{"$", ""}
}

func (l *RegexLexer) GetCurrentPosition() (int, int) {
	return l.line, l.column
}

func NewRegexLexer() Lexer {
	return &RegexLexer{
		[]rune{},
		[]pattern{},
		0, 1, 0,
	}
}

func dumpPatterns(patterns []pattern) {
	for _, p := range patterns {
		logger.Printf("regex: %v\n", p)
	}
}
