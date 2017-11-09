package gdpgen

import (
	"errors"
	"fmt"
	"log"
	"os"
)

var logger = log.New(os.Stderr, "[Parser] ", 0)

type Parser struct {
	augG        *G
	lex         Lexer
	actionTable *actionTable
	gotoTable   *goToTable
}

func NewParser(g *G, lex Lexer) *Parser {
	augment(g)
	parser := &Parser{
		g,
		lex,
		newActionTable(),
		newGoToTable(),
	}
	parser.constructParsingTable()

	return parser
}

type ast struct {
	name  string
	value interface{}
	lhs   *ast
	rhs   *ast
}

var augStartElem = &ProductElem{
	false, NON_TERMINAL, "S'", nil,
}

var eot = &ProductElem{
	true, TERMINAL, "$", nil,
}

type actioncode int

const (
	shiftAction  actioncode = iota
	reduceAction
	acceptAction
	errorAction
)

type action struct {
	op    actioncode
	state int      // has set if op = shiftAction
	prod  *Product // has set if op = reduceAction
}

func (action *action) String() string {
	switch action.op {
	case shiftAction:
		return fmt.Sprintf("<<shiftAction %v>>", action.state)
	case reduceAction:
		return fmt.Sprintf("<<reduceAction %v>>", action.prod)
	case acceptAction:
		return fmt.Sprintf("<<acceptAction>>")
	case errorAction:
		return fmt.Sprintf("<<errorAction>>")
	}
	return fmt.Sprintf("%v", action)
}

func newShiftAction(state int) *action {
	return &action{
		shiftAction, state, &Product{},
	}
}

func newReduceAction(prod *Product) *action {
	return &action{
		reduceAction, 0, prod,
	}
}

func newAcceptAction() *action {
	return &action{
		acceptAction, 0, &Product{},
	}
}

func newErrorAction() *action {
	return &action{
		errorAction, 0, &Product{},
	}
}

type actionTable struct {
	states map[int]map[*ProductElem]*action
}

func newActionTable() *actionTable {
	return &actionTable{
		make(map[int]map[*ProductElem]*action),
	}
}

func (t *actionTable) set(state int, term *ProductElem, act *action) {
	// if has not register state, create
	if _, ok := t.states[state]; !ok {
		t.states[state] = make(map[*ProductElem]*action)
	}

	// shiftAction/reduceAction handling (shiftAction is higer priority)
	if val, ok := t.states[state][term]; ok {
		switch t.states[state][term].op {
		case reduceAction:
			if act.op == shiftAction {
				logger.Printf("action conflict: %v %v, %v, %v", val, state, term, act)
				t.states[state][term] = act
			}
		}
	} else {
		t.states[state][term] = act
	}
}

func (t *actionTable) get(state int, term *ProductElem) (*action, error) {
	if act, ok := t.states[state][term]; ok {
		return act, nil
	} else {
		return newErrorAction(), errors.New(fmt.Sprintf("invalid syntax"))
	}
}

type goToTable struct {
	states map[int]map[*ProductElem]int
}

func newGoToTable() *goToTable {
	return &goToTable{
		make(map[int]map[*ProductElem]int),
	}
}

func (t *goToTable) set(state int, head *ProductElem, nextState int) {
	// if has not register state, create
	if _, ok := t.states[state]; !ok {
		t.states[state] = make(map[*ProductElem]int)
	}

	t.states[state][head] = nextState
}

func (t *goToTable) get(state int, head *ProductElem) (int, error) {
	if goTo, ok := t.states[state][head]; ok {
		return goTo, nil
	} else {
		return 0, errors.New(fmt.Sprintf("invalid syntax"))
	}
}

type semaStack struct {
	stack []interface{}
}

func newSemaStack() *semaStack {
	return &semaStack{
		[]interface{}{},
	}
}

func (self *semaStack) push(e interface{}) {
	self.stack = append(self.stack, e)
}

func (self *semaStack) pop() interface{} {
	if len(self.stack) > 0 {
		top := self.stack[len(self.stack)-1]
		self.stack = self.stack[:len(self.stack)-1]
		return top
	} else {
		return nil
	}
}

func (self *semaStack) String() string {
	return fmt.Sprintf("%v", self.stack)
}

func (parser *Parser) Parse(w string) (interface{}, error) {
	parser.lex.GetReader(w)
	var token Token
	var a *ProductElem
	stack := []int{0}
	semaStack := newSemaStack()
	terminals := parser.augG.GetTerminals()
	token = parser.lex.GetNextToken()
	a = getTerminalFrom(terminals, token.Name)
	if a == nil {
		return nil, errors.New(fmt.Sprintf("unknown token: %v\n", token))
	}
	for {
		s := stack[len(stack)-1]
		act, err := parser.actionTable.get(s, a)
		// logger.Printf("s: %v, a: %v, act: %v, state: %v\n", s, a, act, stack)
		if err != nil {
			line, column := parser.lex.GetCurrentPosition()
			expects := getCandidatesFromActionTable(parser, s)
			return nil, errors.New(fmt.Sprintf("invalid syntax at line:%v, column:%v. expects one of %v, but actual '%v'\n", line, column-len(token.Value), expects, token))
		}
		switch act.op {
		case shiftAction:
			stack = append(stack, act.state)
			semaStack.push(token)
			token = parser.lex.GetNextToken()
			a = getTerminalFrom(terminals, token.Name)
			if a == nil {
				return nil, errors.New(fmt.Sprintf("unknown token: %v\n", token))
			}
		case reduceAction:
			prod := act.prod
			poppedSemas := []interface{}{}
			for _, elem := range prod.Body {
				if elem != EmptyElem {
					_, stack = popStack(stack)

					// semantic handle
					poppedSemas = append(poppedSemas, semaStack.pop())
				} else {
					poppedSemas = append(poppedSemas, nil)
				}
			}
			t := stack[len(stack)-1]
			goTo, goToErr := parser.gotoTable.get(t, prod.Head)
			if goToErr != nil {
				fmt.Printf("%v\n", goToErr)
				line, column := parser.lex.GetCurrentPosition()
				return nil, errors.New(fmt.Sprintf("invalid syntax at line:%v, column:%v. token: %v\n", line, column-len(token.Value), token))
			}
			stack = append(stack, goTo)

			// semantic callback
			if prod.Callback != nil {
				reversed := make([]interface{}, len(poppedSemas))
				for i, sema := range poppedSemas {
					reversed[len(reversed)-1-i] = sema
				}

				reduced := prod.Callback(reversed)
				semaStack.push(reduced)
			} else {
				semaStack.push(poppedSemas)
			}
		case acceptAction:
			if 0 < len(semaStack.stack) {
				return semaStack.pop(), nil
			} else {
				return true, nil
			}
		case errorAction:
			panic("invalid syntax")
		}
	}
}

func getCandidatesFromActionTable(parser *Parser, state int) []*ProductElem {
	candidates := []*ProductElem{}
	stateMap := parser.actionTable.states[state]
	for i := range stateMap {
		candidates = append(candidates, i)
	}
	return candidates
}

func popStack(stack []int) (int, []int) {
	if len(stack) < 2 {
		return stack[0], []int{}
	}
	return stack[len(stack)-1], stack[:len(stack)-1]
}

type item struct {
	head      *ProductElem
	body      []*ProductElem
	position  int
	lookahead *ProductElem
}

func (self item) String() string {
	return fmt.Sprintf("{%v -> %v,%v %v}",
		self.head, self.body, self.lookahead.Sig, self.position)
}

type setOfItems struct {
	items []*item
}

func (itemSet setOfItems) String() string {
	return fmt.Sprintf("%v", itemSet.items)
}

func newSetOfItems() *setOfItems {
	return &setOfItems{}
}

func newSetOfItemsFrom(items ...*item) *setOfItems {
	return &setOfItems{items}
}

func (items *setOfItems) Has(item *item) bool {
	for _, val := range items.items {
		if val.head == item.head &&
			CompareProductElem(val.body, item.body) &&
			val.position == item.position &&
			val.lookahead == item.lookahead {
			return true
		}
	}
	return false
}

func (items *setOfItems) Values() []*item {
	return items.items
}

func (items *setOfItems) Add(item *item) {
	if !items.Has(item) {
		items.items = append(items.items, item)
	}
}

func (a *setOfItems) Equals(b *setOfItems) bool {
	if len(a.items) != len(b.items) {
		return false
	}

	equals := true
	for _, item := range b.items {
		if !a.Has(item) {
			equals = false
			break
		}
	}
	return equals
}

// Add item
func (parser *Parser) closure(items *setOfItems) {
	changed := true
	for changed {
		changed = false
		for _, e := range items.Values() {
			_, nonTerm, follow, lookAhead := match(e)
			for _, product := range parser.augG.GetProductsOf(nonTerm) {
				first := parser.firstAll(append(follow, lookAhead))
				for _, terminal := range first {
					newItem := &item{
						nonTerm,
						product.Body,
						0,
						terminal}
					if !items.Has(newItem) {
						items.Add(newItem)
						changed = true
					}
				}
			}
		}
	}
}

// Create new setOfItems
func (parser Parser) goTo(items *setOfItems, symbol *ProductElem) *setOfItems {
	itemsRight := newSetOfItems()
	for _, e := range items.Values() {
		beforeDot, afterDot, tail, lookahead := match(e)
		if afterDot == EmptyElem {
			continue
		}
		if afterDot == symbol {
			itemsRight.Add(&item{
				e.head,
				append(append(beforeDot, afterDot), tail...),
				e.position + 1,
				lookahead})
		}
	}
	parser.closure(itemsRight)
	return itemsRight
}

func (parser *Parser) items() []*setOfItems {
	item := &item{
		augStartElem,
		[]*ProductElem{parser.augG.StartSymbol},
		0,
		eot}
	var collections []*setOfItems
	items := newSetOfItemsFrom(item)
	parser.closure(items)
	collections = []*setOfItems{items}

	symbols := parser.augG.GetSymbolSet()
	var changed = true
	for changed {
		changed = false
		for _, items := range collections {
			for _, symbol := range symbols {
				goToItems := parser.goTo(items, symbol)
				if len(goToItems.items) > 0 && !has(collections, goToItems) {
					collections = append(collections, goToItems)
					changed = true
				}
			}
		}
	}

	return collections
}

func (parser *Parser) constructParsingTable() {
	// construct collection of sets of LR(1) items
	collections := parser.items()

	// construct action table
	for i, items := range collections {
		for _, item := range items.items {
			beforeDot, afterDot, _, lookahead := match(item)
			if afterDot != EmptyElem && afterDot.IsTerminal {
				nextStateItem := parser.goTo(items, afterDot)
				for j, testItems := range collections {
					if nextStateItem.Equals(testItems) {
						parser.actionTable.set(i, afterDot, newShiftAction(j))
					}
				}
			} else if item.head != augStartElem {
				if afterDot == EmptyElem {
					prod := parser.augG.GetProductOf(item.head, beforeDot)
					parser.actionTable.set(i, lookahead, newReduceAction(prod))
				}
			} else if item.head == augStartElem &&
				afterDot == EmptyElem &&
				lookahead == eot {
				parser.actionTable.set(i, eot, newAcceptAction())
			}
		}
	}

	// construct goto table
	nonTerminals := parser.augG.GetNonTerminals()
	for i, items := range collections {
		for _, nonTerm := range nonTerminals {
			for j, testItems := range collections {
				if parser.goTo(items, nonTerm).Equals(testItems) {
					parser.gotoTable.set(i, nonTerm, j)
				}
			}
		}
	}

	// dumpActionTable(parser.actionTable)
	// dumpGoToTable(parser.gotoTable)
}

func dumpActionTable(actTable *actionTable) {
	logger.Println("ACTION table ======")
	for state, actions := range actTable.states {
		for i, act := range actions {
			switch act.op {
			case shiftAction:
				logger.Printf("state: %v, term: %v(%#p) -> shiftAction %v\n",
					state, i, i, act.state)
			case reduceAction:
				logger.Printf("state: %v, term: %v(%#p) -> reduceAction %v\n",
					state, i, i, act.prod)
			case acceptAction:
				logger.Printf("state: %v, term: %v(%#p) -> acceptAction\n",
					state, i, i)
			}
		}
	}
}

func dumpGoToTable(goToTable *goToTable) {
	logger.Println("GOTO table ======")
	for state, nextStateMap := range goToTable.states {
		for i, nextState := range nextStateMap {
			logger.Printf("state: %v, nonTerm: %v -> %v\n",
				state, i, nextState)
		}
	}
}

func (parser *Parser) first(elem *ProductElem) []*ProductElem {
	firstSet := []*ProductElem{}
	if elem.IsTerminal {
		return []*ProductElem{elem}
	}
	for _, productBody := range parser.augG.GetProductBodySet(elem) {
		if len(productBody) == 1 && productBody[0] == EmptyElem {
			firstSet = append(firstSet, EmptyElem)
			continue
		}
		var recentSet []*ProductElem
		var emptyCount = 0
		for i := range productBody {
			recentSet = parser.first(productBody[i])
			firstSet = append(firstSet, recentSet...)
			if !hasElem(recentSet, EmptyElem) {
				break
			} else {
				emptyCount++
			}
		}
		if emptyCount == len(productBody) {
			firstSet = append(firstSet, EmptyElem)
		}
	}
	dupRemoved := removeDup(firstSet)

	return dupRemoved
}

func removeDup(array []*ProductElem) []*ProductElem {
	m := make(map[*ProductElem]struct{})
	for _, elem := range array {
		m[elem] = struct{}{}
	}
	array2 := []*ProductElem{}
	for key := range m {
		array2 = append(array2, key)
	}
	return array2
}

func (parser *Parser) nullable(elem *ProductElem) bool {
	if elem == EmptyElem {
		return true
	}

	productBodySet := parser.augG.GetProductBodySet(elem)
	for _, productBody := range productBodySet {
		nullable := true
		for _, productBodyElem := range productBody {
			if !parser.nullable(productBodyElem) {
				nullable = false
				break
			}
		}
		if nullable {
			return true
		}
	}

	return false
}

func (parser *Parser) firstAll(elems []*ProductElem) []*ProductElem {
	if len(elems) < 1 {
		panic("invalid argument")
	}

	allFirstSet := []*ProductElem{}
	for i := 0; i < len(elems); i++ {
		firstSet := parser.first(elems[i])
		allFirstSet = append(allFirstSet, nonEmptySet(firstSet)...)
		if !existsEmpty(firstSet) {
			break
		}
	}
	return allFirstSet
}

func existsEmpty(symbols []*ProductElem) bool {
	for _, symbol := range symbols {
		if symbol == EmptyElem {
			return true
		}
	}
	return false
}

func nonEmptySet(elems []*ProductElem) []*ProductElem {
	arr := []*ProductElem{}
	for _, elem := range elems {
		if elem == EmptyElem {
			continue
		}
		arr = append(arr, elem)
	}
	return arr
}

// match to [A → α·Bβ,a]
func match(item *item) ([]*ProductElem, *ProductElem, []*ProductElem, *ProductElem) {
	var beforeDot []*ProductElem
	if item.position == 0 {
		beforeDot = []*ProductElem{}
	} else {
		beforeDot = item.body[:item.position]
	}

	var afterDot *ProductElem
	if item.position < len(item.body) {
		afterDot = item.body[item.position]
	} else {
		afterDot = EmptyElem
	}
	var tail []*ProductElem
	if item.position+1 < len(item.body) {
		tail = item.body[item.position+1:]
	} else {
		tail = []*ProductElem{}
	}
	return beforeDot, afterDot, tail, item.lookahead
}

func has(collection []*setOfItems, setOfItems *setOfItems) bool {
	var allIn bool
	for _, val := range collection {
		allIn = true
		for _, item := range setOfItems.Values() {
			if !val.Has(item) {
				allIn = false
				break
			}
		}
		if allIn {
			return true
		}
	}
	return false
}

func hasElem(elems []*ProductElem, test *ProductElem) bool {
	for _, elem := range elems {
		if elem == test {
			return true
		}
	}

	return false
}

func augment(g *G) {
	g.AddProduct(&Product{
		augStartElem,
		[]*ProductElem{g.StartSymbol},
		func(symbols []interface{}) interface{} {
			return symbols[0]
		},
	})
}

func getTerminalFrom(terms []*ProductElem, name string) *ProductElem {
	if name == "$" {
		return eot
	}
	for _, elem := range terms {
		if elem.Sig == name {
			return elem
		}
	}
	return nil
}
