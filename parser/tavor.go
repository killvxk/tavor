package parser

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"text/scanner"

	"github.com/zimmski/tavor/token"
	"github.com/zimmski/tavor/token/aggregates"
	"github.com/zimmski/tavor/token/constraints"
	"github.com/zimmski/tavor/token/lists"
	"github.com/zimmski/tavor/token/primitives"
	"github.com/zimmski/tavor/token/sequences"
)

//TODO remove this
var DEBUG = false

const zeroRune = 0

const (
	MaxRepeat = 2
)

type tavorParser struct {
	scan scanner.Scanner

	err string

	lookup map[string]token.Token
	used   map[string]struct{}
}

func (p *tavorParser) expectRune(expect rune, got rune) (rune, error) {
	if got != expect {
		return got, &ParserError{
			Message: fmt.Sprintf("Expected \"%c\" but got \"%c\"", expect, got),
			Type:    ParseErrorExpectRune,
		}
	}

	return got, nil
}

func (p *tavorParser) expectScanRune(expect rune) (rune, error) {
	got := p.scan.Scan()
	if DEBUG {
		fmt.Printf("%d:%v -> %v\n", p.scan.Line, scanner.TokenString(got), p.scan.TokenText())
	}

	return p.expectRune(expect, got)
}

func (p *tavorParser) parseGlobalScope() error {
	var err error

	c := p.scan.Scan()
	if DEBUG {
		fmt.Printf("%d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
	}

	for c != scanner.EOF {
		switch c {
		case '\n':
			// ignore new lines in the global scope
		case scanner.Ident:
			c, err = p.parseTokenDefinition()
			if err != nil {
				return err
			}

			continue
		case '$':
			c, err = p.parseSpecialTokenDefinition()
			if err != nil {
				return err
			}

			continue
		case scanner.Int:
			return &ParserError{
				Message: "Token names have to start with a letter",
				Type:    ParseErrorInvalidTokenName,
			}
		default:
			panic("what am i to do now") // TODO remove this
		}

		c = p.scan.Scan()
		if DEBUG {
			fmt.Printf("%d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
		}
	}

	return nil
}

func (p *tavorParser) parseTerm(c rune) (rune, []token.Token, error) {
	var tokens []token.Token

OUT:
	for {
		switch c {
		case scanner.Ident:
			n := p.scan.TokenText()

			if _, ok := p.lookup[n]; !ok {
				return zeroRune, nil, &ParserError{
					Message: fmt.Sprintf("Token %s is not defined", n),
					Type:    ParseErrorTokenNotDefined,
				}
			}

			p.used[n] = struct{}{}

			tokens = append(tokens, p.lookup[n])
		case scanner.Int:
			v, _ := strconv.Atoi(p.scan.TokenText())

			tokens = append(tokens, primitives.NewConstantInt(v))
		case scanner.String:
			s := p.scan.TokenText()

			if s[0] != '"' {
				panic("unknown " + s) // TODO remove this
			}

			if s[len(s)-1] != '"' {
				return zeroRune, nil, &ParserError{
					Message: "String is not terminated",
					Type:    ParseErrorNonTerminatedString,
				}
			}

			tokens = append(tokens, primitives.NewConstantString(s[1:len(s)-1]))
		case '(':
			if DEBUG {
				fmt.Println("NEW group")
			}
			c = p.scan.Scan()
			if DEBUG {
				fmt.Printf("parseTerm Group %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
			}

			c, toks, err := p.parseScope(c)
			if err != nil {
				return zeroRune, nil, err
			}

			p.expectRune(')', c)

			switch len(toks) {
			case 0:
				// ignore
			case 1:
				tokens = append(tokens, toks[0])
			default:
				tokens = append(tokens, lists.NewAll(toks...))
			}

			if DEBUG {
				fmt.Println("END group")
			}
		case '?':
			if DEBUG {
				fmt.Println("NEW optional")
			}
			p.expectScanRune('(')

			c = p.scan.Scan()
			if DEBUG {
				fmt.Printf("parseTerm optional after ( %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
			}

			c, toks, err := p.parseScope(c)
			if err != nil {
				return zeroRune, nil, err
			}

			p.expectRune(')', c)

			switch len(toks) {
			case 0:
				// ignore
			case 1:
				tokens = append(tokens, constraints.NewOptional(toks[0]))
			default:
				tokens = append(tokens, constraints.NewOptional(lists.NewAll(toks...)))
			}

			if DEBUG {
				fmt.Println("END optional")
			}
		case '+', '*':
			if DEBUG {
				fmt.Println("NEW repeat")
			}

			sym := c

			c = p.scan.Scan()
			if DEBUG {
				fmt.Printf("parseTerm repeat before ( %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
			}

			var from, to int

			if sym == '*' {
				from, to = 0, MaxRepeat
			} else {
				if c == scanner.Int {
					from, _ = strconv.Atoi(p.scan.TokenText())

					c = p.scan.Scan()
					if DEBUG {
						fmt.Printf("parseTerm repeat after from ( %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
					}

					// until there is an explicit "to" we can assume to==from
					to = from
				} else {
					from, to = 1, MaxRepeat
				}

				if c == ',' {
					c = p.scan.Scan()
					if DEBUG {
						fmt.Printf("parseTerm repeat after , ( %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
					}

					if c == scanner.Int {
						to, _ = strconv.Atoi(p.scan.TokenText())

						c = p.scan.Scan()
						if DEBUG {
							fmt.Printf("parseTerm repeat after to ( %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
						}
					} else {
						to = MaxRepeat
					}
				}
			}

			p.expectRune('(', c)

			if DEBUG {
				fmt.Printf("repeat from %v to %v\n", from, to)
			}

			c = p.scan.Scan()
			if DEBUG {
				fmt.Printf("parseTerm repeat after ( %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
			}

			c, toks, err := p.parseScope(c)
			if err != nil {
				return zeroRune, nil, err
			}

			p.expectRune(')', c)

			switch len(toks) {
			case 0:
				// ignore
			case 1:
				tokens = append(tokens, lists.NewRepeat(toks[0], int64(from), int64(to)))
			default:
				tokens = append(tokens, lists.NewRepeat(lists.NewAll(toks...), int64(from), int64(to)))
			}

			if DEBUG {
				fmt.Println("END repeat")
			}
		case '$':
			if DEBUG {
				fmt.Println("START token attribute")
			}

			if tok, err := p.parseTokenAttribute(); err != nil {
				return zeroRune, nil, err
			} else {
				tokens = append(tokens, tok)
			}

			if DEBUG {
				fmt.Println("END token attribute")
			}
		case ',': // multi line token
			if _, err := p.expectScanRune('\n'); err != nil {
				return zeroRune, nil, err
			}

			c = p.scan.Scan()
			if DEBUG {
				fmt.Printf("parseTerm multiline %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
			}

			if c == '\n' {
				return zeroRune, nil, &ParserError{
					Message: "Multi line token definition unexpectedly terminated",
					Type:    ParseErrorUnexpectedTokenDefinitionTermination,
				}
			}

			continue
		default:
			if DEBUG {
				fmt.Println("break out parseTerm")
			}
			break OUT
		}

		c = p.scan.Scan()
		if DEBUG {
			fmt.Printf("parseTerm %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
		}
	}

	return c, tokens, nil
}

func (p *tavorParser) parseTokenAttribute() (token.Token, error) {
	_, err := p.expectScanRune(scanner.Ident)
	if err != nil {
		return nil, err
	}

	name := p.scan.TokenText()

	_, err = p.expectScanRune('.')
	if err != nil {
		return nil, err
	}

	_, err = p.expectScanRune(scanner.Ident)
	if err != nil {
		return nil, err
	}

	attribute := p.scan.TokenText()

	tok, ok := p.lookup[name]
	if !ok {
		return nil, &ParserError{
			Message: fmt.Sprintf("Token %s is not defined", name),
			Type:    ParseErrorTokenNotDefined,
		}
	}

	p.used[name] = struct{}{}

	switch i := tok.(type) {
	case lists.List:
		switch attribute {
		case "Count":
			return aggregates.NewLen(i), nil
		}
	case *sequences.Sequence:
		switch attribute {
		case "Existing":
			return i.ExistingItem(), nil
		case "Next":
			return i.Item(), nil
		}
	}

	return nil, &ParserError{
		Message: fmt.Sprintf("Unknown token attribute %s for token type %s", attribute, reflect.TypeOf(tok)),
		Type:    ParseErrorUnknownTokenAttribute,
	}
}

func (p *tavorParser) parseScope(c rune) (rune, []token.Token, error) {
	var err error

	var tokens []token.Token

OUT:
	for {
		// identifier and literals
		var toks []token.Token
		c, toks, err = p.parseTerm(c)
		if err != nil {
			return zeroRune, nil, err
		} else if toks != nil {
			tokens = append(tokens, toks...)
		}

		// alternations
		switch c {
		case '|':
			if DEBUG {
				fmt.Println("NEW or")
			}
			var orTerms []token.Token
			optional := false

			toks = tokens

		OR:
			for {
				switch len(toks) {
				case 0:
					optional = true
				case 1:
					orTerms = append(orTerms, toks[0])
				default:
					orTerms = append(orTerms, lists.NewAll(toks...))
				}

				if c == '|' {
					c = p.scan.Scan()
					if DEBUG {
						fmt.Printf("parseScope Or %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
					}
				} else {
					if DEBUG {
						fmt.Println("parseScope break out or")
					}
					break OR
				}

				c, toks, err = p.parseTerm(c)
				if err != nil {
					return zeroRune, nil, err
				}
			}

			or := lists.NewOne(orTerms...)

			if optional {
				tokens = []token.Token{constraints.NewOptional(or)}
			} else {
				tokens = []token.Token{or}
			}

			if DEBUG {
				fmt.Println("END or")
			}

			continue
		default:
			break OUT
		}

		c = p.scan.Scan()
		if DEBUG {
			fmt.Printf("parseScope %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
		}
	}

	return c, tokens, nil
}

func (p *tavorParser) parseTokenDefinition() (rune, error) {
	var c rune
	var err error

	name := p.scan.TokenText()

	if _, ok := p.lookup[name]; ok {
		return zeroRune, &ParserError{
			Message: "Token already defined",
			Type:    ParseErrorTokenAlreadyDefined,
		}
	}

	// do an empty definition to allow loops
	p.lookup[name] = nil

	if c, err = p.expectScanRune('='); err != nil {
		// unexpected new line?
		if c == '\n' {
			return zeroRune, &ParserError{
				Message: "New line inside single line token definitions is not allowed",
				Type:    ParseErrorEarlyNewLine,
			}
		}

		return zeroRune, err
	}

	c = p.scan.Scan()
	if DEBUG {
		fmt.Printf("parseTokenDefinition after = %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
	}

	c, tokens, err := p.parseScope(c)
	if err != nil {
		return zeroRune, err
	}

	if DEBUG {
		fmt.Printf("back to token definition with c=%c\n", c)
	}

	// we always want a new line at the end of the file
	if c == scanner.EOF {
		return zeroRune, &ParserError{
			Message: "New line at end of token definition needed",
			Type:    ParseErrorNewLineNeeded,
		}
	}

	if c, err = p.expectRune('\n', c); err != nil {
		return zeroRune, err
	}

	switch len(tokens) {
	case 0:
		return zeroRune, &ParserError{
			Message: "Empty token definition",
			Type:    ParseErrorEmptyTokenDefinition,
		}
	case 1:
		p.lookup[name] = tokens[0]
	default:
		p.lookup[name] = lists.NewAll(tokens...)
	}

	c = p.scan.Scan()
	if DEBUG {
		fmt.Printf("parseTokenDefinition after newline %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
	}

	return c, nil
}

func (p *tavorParser) parseSpecialTokenDefinition() (rune, error) {
	var c rune
	var err error

	if DEBUG {
		fmt.Println("START special token")
	}

	c = p.scan.Scan()
	if DEBUG {
		fmt.Printf("parseSpecialTokenDefinition after $ %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
	}

	name := p.scan.TokenText()
	if _, ok := p.lookup[name]; ok {
		return zeroRune, &ParserError{
			Message: "Token already defined",
			Type:    ParseErrorTokenAlreadyDefined,
		}
	}

	if c, err = p.expectScanRune('='); err != nil {
		return zeroRune, err
	}

	arguments := make(map[string]string)

	for {
		c, err = p.expectScanRune(scanner.Ident)
		if err != nil {
			return zeroRune, err
		}

		arg := p.scan.TokenText()

		_, err = p.expectScanRune(':')
		if err != nil {
			return zeroRune, err
		}

		c = p.scan.Scan()
		if DEBUG {
			fmt.Printf("parseSpecialTokenDefinition argument value %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
		}

		switch c {
		case scanner.Ident, scanner.String, scanner.Int:
			arguments[arg] = p.scan.TokenText()
		default:
			return zeroRune, &ParserError{
				Message: fmt.Sprintf("Invalid argument value %v", c),
				Type:    ParseErrorInvalidArgumentValue,
			}
		}

		c = p.scan.Scan()
		if DEBUG {
			fmt.Printf("parseSpecialTokenDefinition after argument value %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
		}

		if c != ',' {
			break
		}

		if c, err = p.expectScanRune('\n'); err != nil {
			return zeroRune, err
		}
	}

	// we always want a new line at the end of the file
	if c == scanner.EOF {
		return zeroRune, &ParserError{
			Message: "New line at end of token definition needed",
			Type:    ParseErrorNewLineNeeded,
		}
	}

	if c, err = p.expectRune('\n', c); err != nil {
		return zeroRune, err
	}

	typ, ok := arguments["type"]
	if !ok {
		return zeroRune, &ParserError{
			Message: "Special token has no type argument",
			Type:    ParseErrorUnknownTypeForSpecialToken,
		}
	}

	var tok token.Token
	usedArguments := map[string]struct{}{
		"type": struct{}{},
	}

	switch typ {
	case "Int":
		rawFrom, okFrom := arguments["from"]
		rawTo, okTo := arguments["to"]

		if okFrom || okTo {
			if okFrom && !okTo {
				return zeroRune, &ParserError{
					Message: "Argument \"to\" is missing",
					Type:    ParseErrorMissingSpecialTokenArgument,
				}
			} else if !okFrom && okTo {
				return zeroRune, &ParserError{
					Message: "Argument \"from\" is missing",
					Type:    ParseErrorMissingSpecialTokenArgument,
				}
			}

			from, err := strconv.Atoi(rawFrom)
			if err != nil {
				return zeroRune, &ParserError{
					Message: "\"from\" needs an integer value",
					Type:    ParseErrorInvalidArgumentValue,
				}
			}

			to, err := strconv.Atoi(rawTo)
			if err != nil {
				return zeroRune, &ParserError{
					Message: "\"to\" needs an integer value",
					Type:    ParseErrorInvalidArgumentValue,
				}
			}

			usedArguments["from"] = struct{}{}
			usedArguments["to"] = struct{}{}

			tok = primitives.NewRangeInt(from, to)
		} else {
			tok = primitives.NewRandomInt()
		}
	case "Sequence":
		start := 1
		step := 1

		if raw, ok := arguments["start"]; ok {
			start, err = strconv.Atoi(raw)
			if err != nil {
				return zeroRune, &ParserError{
					Message: "\"start\" needs an integer value",
					Type:    ParseErrorInvalidArgumentValue,
				}
			}
		}

		if raw, ok := arguments["step"]; ok {
			step, err = strconv.Atoi(raw)
			if err != nil {
				return zeroRune, &ParserError{
					Message: "\"step\" needs an integer value",
					Type:    ParseErrorInvalidArgumentValue,
				}
			}
		}

		usedArguments["start"] = struct{}{}
		usedArguments["step"] = struct{}{}

		tok = sequences.NewSequence(start, step)
	default:
		return zeroRune, &ParserError{
			Message: fmt.Sprintf("Unknown special token type %s", typ),
			Type:    ParseErrorUnknownSpecialTokenType,
		}
	}

	for arg, _ := range arguments {
		if _, ok := usedArguments[arg]; !ok {
			return zeroRune, &ParserError{
				Message: fmt.Sprintf("Unknown special token argument %s", arg),
				Type:    ParseErrorUnknownSpecialTokenArgument,
			}
		}
	}

	p.lookup[name] = tok

	c = p.scan.Scan()
	if DEBUG {
		fmt.Printf("parseSpecialTokenDefinition after newline %d:%v -> %v\n", p.scan.Line, scanner.TokenString(c), p.scan.TokenText())
	}

	if DEBUG {
		fmt.Println("END special token")
	}

	return c, nil
}

func ParseTavor(src io.Reader) (token.Token, error) {
	p := &tavorParser{
		lookup: make(map[string]token.Token),
		used:   make(map[string]struct{}),
	}

	if DEBUG {
		fmt.Println("INIT")
	}

	p.scan.Init(src)

	p.scan.Error = func(s *scanner.Scanner, msg string) {
		p.err = msg
	}
	p.scan.Whitespace = 1<<'\t' | 1<<' ' | 1<<'\r'

	if err := p.parseGlobalScope(); err != nil {
		return nil, err
	}

	if _, ok := p.lookup["START"]; !ok {
		return nil, &ParserError{
			Message: "No START token defined",
			Type:    ParseErrorNoStart,
		}
	}

	p.used["START"] = struct{}{}

	for key := range p.lookup {
		if _, ok := p.used[key]; !ok {
			return nil, &ParserError{
				Message: fmt.Sprintf("Token %s declared but not used", key),
				Type:    ParseErrorUnusedToken,
			}
		}
	}

	return p.lookup["START"], nil
}
