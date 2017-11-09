package gdpgen

import (
	"fmt"
)

const (
	NON_TERMINAL SymbolType = iota
	TERMINAL
	ID
	EMPTY
)

type SymbolType int

type ProductElem struct {
	IsTerminal bool
	SymbolType SymbolType
	Sig        string
	Value      interface{}
}

func (p ProductElem) String() string {
	var typeOf string
	switch p.SymbolType {
	case NON_TERMINAL:
		typeOf = "NonTerminal"
	case TERMINAL:
		typeOf = "Terminal"
	case EMPTY:
		typeOf = "Empty"
	}
	return fmt.Sprintf("%v(%v)", typeOf, p.Sig)
}

func NewProductElem(isTerminal bool, symbolType SymbolType, sig string) *ProductElem {
	return &ProductElem{
		isTerminal,
		symbolType,
		sig,
		nil,
	}
}

func NewNonTerminal(sig string) *ProductElem {
	return &ProductElem{
		false,
		NON_TERMINAL,
		sig,
		nil,
	}
}

func NewTerminal(sig string) *ProductElem {
	return &ProductElem{
		true,
		TERMINAL,
		sig,
		nil,
	}
}

var EmptyElem = &ProductElem{
	true,
	EMPTY,
	"",
	nil,
}

type Product struct {
	Head     *ProductElem
	Body     []*ProductElem
	Callback func([]interface{}) interface{}
}

func NewProduct(head *ProductElem) *Product {
	return &Product{head, []*ProductElem{}, nil}
}

func (p Product) String() string {
	return fmt.Sprintf("%v -> %v", p.Head.Sig, p.Body)
}

type G struct {
	StartSymbol *ProductElem
	Products    []*Product
}

func NewGrammar(startSymbol *ProductElem) *G {
	return &G{startSymbol, []*Product{}}
}

func (g *G) GetSymbolSet() []*ProductElem {
	notIn := func(elems []*ProductElem, test *ProductElem) bool {
		for _, elem := range elems {
			if elem == test {
				return false
			}
		}
		return true
	}
	set := []*ProductElem{}
	for _, product := range g.Products {
		head := product.Head
		if notIn(set, head) {
			set = append(set, head)
		}
		for _, elem := range product.Body {
			if notIn(set, elem) {
				set = append(set, elem)
			}
		}
	}
	return set
}

func (g *G) GetNonTerminals() []*ProductElem {
	nonTerminals := []*ProductElem{}
	for _, symbol := range g.GetSymbolSet() {
		if !symbol.IsTerminal {
			nonTerminals = append(nonTerminals, symbol)
		}
	}
	return nonTerminals
}

func (g *G) GetTerminals() []*ProductElem {
	terminals := []*ProductElem{}
	for _, symbol := range g.GetSymbolSet() {
		if symbol.IsTerminal {
			terminals = append(terminals, symbol)
		}
	}
	return terminals
}

func (g *G) AddProduct(p *Product) {
	g.Products = append(g.Products, p)
}

func (g *G) GetProductsOf(head *ProductElem) []*Product {
	var products []*Product
	for _, product := range g.Products {
		if product.Head == head {
			products = append(products, product)
		}
	}
	return products
}

func (g *G) GetProductOf(head *ProductElem, body []*ProductElem) *Product {
	if len(body) == 0 {
		body = []*ProductElem{EmptyElem}
	}
	for _, p := range g.Products {
		if p.Head == head && CompareProductElem(p.Body, body) {
			return p
		}
	}

	return nil
}

func (g *G) GetProductBodySet(head *ProductElem) [][]*ProductElem {
	productBodySet := [][]*ProductElem{}
	for _, product := range g.Products {
		if product.Head == head {
			productBodySet = append(productBodySet, product.Body)
		}
	}
	return productBodySet
}

func CompareProductElem(a, b []*ProductElem) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
