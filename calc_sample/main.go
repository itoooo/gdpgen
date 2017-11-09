package main

import "fmt"
import . "gdpgen"
import "strconv"

func main() {
	expr := NewNonTerminal("expr")
	term := NewNonTerminal("term")
	fact := NewNonTerminal("fact")

	g := NewGrammar(expr)
	g.AddProduct(&Product{expr, []*ProductElem{term},
		func(tokens []interface{}) interface{} {
			val := tokens[0]
			return val
		}})
	g.AddProduct(&Product{expr, []*ProductElem{expr, NewTerminal("+"), term},
		func(tokens []interface{}) interface{} {
			leftVal := tokens[0]
			rightVal := tokens[2]
			return leftVal.(int) + rightVal.(int)
		}})
	g.AddProduct(&Product{expr, []*ProductElem{expr, NewTerminal("-"), term},
		func(tokens []interface{}) interface{} {
			leftVal := tokens[0]
			rightVal := tokens[2]
			return leftVal.(int) - rightVal.(int)
		}})
	g.AddProduct(&Product{term, []*ProductElem{fact},
		func(tokens []interface{}) interface{} {
			val := tokens[0]
			return val
		}})
	g.AddProduct(&Product{term, []*ProductElem{term, NewTerminal("*"), fact},
		func(tokens []interface{}) interface{} {
			leftVal := tokens[0]
			rightVal := tokens[2]
			return leftVal.(int) * rightVal.(int)
		}})
	g.AddProduct(&Product{term, []*ProductElem{term, NewTerminal("/"), fact},
		func(tokens []interface{}) interface{} {
			leftVal := tokens[0]
			rightVal := tokens[2]
			return leftVal.(int) / rightVal.(int)
		}})
	g.AddProduct(&Product{fact, []*ProductElem{NewTerminal("("), expr, NewTerminal(")")},
		func(tokens []interface{}) interface{} {
			expr := tokens[1]
			return expr
		}})
	g.AddProduct(&Product{fact, []*ProductElem{NewTerminal("number")},
		func(tokens []interface{}) interface{} {
			numToken := tokens[0]
			num, _ := strconv.Atoi(numToken.(Token).Value)
			return num
		}})

	lex := NewRegexLexer()
	lex.AddPattern("+", "\\+")
	lex.AddPattern("-", "-")
	lex.AddPattern("*", "\\*")
	lex.AddPattern("/", "/")
	lex.AddPattern("(", "\\(")
	lex.AddPattern(")", "\\)")
	lex.AddPattern("number", "\\d+")

	p := NewParser(g, lex)
	var result interface{}

	result, _ = p.Parse("1 + 1")
	fmt.Printf("\nparse 1 + 1: %v\n", result)

	result, _ = p.Parse("1 + 100")
	fmt.Printf("\nparse 1 + 100: %v\n", result)

	result, _ = p.Parse("8*(1+100)")
	fmt.Printf("\nparse 8*(1+100): %v\n", result)
}
