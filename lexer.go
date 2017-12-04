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

type Pattern struct {
	Name  string
	Regex *regexp.Regexp
}

func NewPattern(name string, regexPattern string) Pattern {
	r, err := regexp.Compile("^" + regexPattern)
	if err != nil {
		panic(err)
	}
	return Pattern{name, r}
}

type RegexLexer struct {
	chars    []rune
	patterns []Pattern
	pos      int
	line     int
	column   int
}

func (l *RegexLexer) GetReader(s string) {
	l.chars = []rune(s)
}

func (l *RegexLexer) AddPattern(name, pattern string) {
	for _, p := range l.patterns {
		if p.Name == name {
			return
		}
	}

	l.patterns = append(l.patterns, NewPattern(name, pattern))
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
			mRange := p.Regex.FindStringIndex(string(l.chars[i:]))
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
				return Token{p.Name, string(value)}
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
		[]Pattern{},
		0, 1, 0,
	}
}

func dumpPatterns(patterns []Pattern) {
	for _, p := range patterns {
		logger.Printf("Regex: %v\n", p)
	}
}
