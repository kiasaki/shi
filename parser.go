package shi

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

type ReadState int

const (
	ReadStateRoot ReadState = iota
	ReadStateList
	ReadStateSingle
)

func LexerAST(l *Lexer, values []Value, state ReadState) []Value {
	for token := l.NextToken(); token.Type != TokenEOF; token = l.NextToken() {
		switch token.Type {
		case TokenIdentifier:
			switch token.Value {
			case "null":
				values = append(values, NULL)
			case "true":
				values = append(values, TRUE)
			case "false":
				values = append(values, FALSE)
			default:
				values = append(values, NewSym(token.Value))
			}
		case TokenKeyword:
			values = append(values, NewString(token.Value[1:]))
		case TokenStringLiteral:
			v, err := strconv.Unquote(token.Value)
			if err != nil {
				panic(err)
			}
			values = append(values, NewString(v))
		case TokenIntegerLiteral:
			v, err := strconv.ParseInt(token.Value, 10, 64)
			if err != nil {
				panic(err)
			}
			values = append(values, NewInt(v))
		case TokenFloatLiteral:
			v, err := strconv.ParseFloat(token.Value, 64)
			if err != nil {
				panic(err)
			}
			values = append(values, NewFloat(v))
		case TokenOpenParen:
			value := NewCell(LexerAST(l, []Value{}, ReadStateList))
			values = append(values, value)
		case TokenCloseParen:
			if state != ReadStateList {
				panic(fmt.Sprint("read: unexpected ')' token"))
			}
			return values
		case TokenOpenSquare:
			value := NewVector(LexerAST(l, []Value{}, ReadStateList))
			values = append(values, value)
		case TokenCloseSquare:
			if state != ReadStateList {
				panic(fmt.Sprint("read: unexpected ']' token"))
			}
			return values
		case TokenOpenCurly:
			value := NewMapFromList(LexerAST(l, []Value{}, ReadStateList))
			values = append(values, value)
		case TokenCloseCurly:
			if state != ReadStateList {
				panic(fmt.Sprint("read: unexpected '}' token"))
			}
			return values
		case TokenQuote:
			exprValues := []Value{NewSym("quote")}
			exprValues = append(exprValues, LexerAST(l, []Value{}, ReadStateSingle)...)
			values = append(values, NewCell(exprValues))
		case TokenQuasiquote:
			exprValues := []Value{NewSym("quasiquote")}
			exprValues = append(exprValues, LexerAST(l, []Value{}, ReadStateSingle)...)
			values = append(values, NewCell(exprValues))
		case TokenUnquote:
			exprValues := []Value{NewSym("unquote")}
			exprValues = append(exprValues, LexerAST(l, []Value{}, ReadStateSingle)...)
			values = append(values, NewCell(exprValues))
		case TokenUnquoteSplicing:
			exprValues := []Value{NewSym("unquote-splicing")}
			exprValues = append(exprValues, LexerAST(l, []Value{}, ReadStateSingle)...)
			values = append(values, NewCell(exprValues))
		case TokenError:
			panic(fmt.Errorf("tokenizer: %s", token.Value))
		default:
			panic(fmt.Errorf("read: unexpected token type: %v", token.Type))
		}

		if state == ReadStateSingle {
			return values
		}
	}

	if state != ReadStateRoot {
		// If we didn't hit a return statement it's that we didn't match
		// on a closing parens for the current list
		panic("read: unmatched delimeter, expected: ')'")
	}

	return values
}

func Parse(name string, input string) []Value {
	l := NewLexer(name, input)
	return LexerAST(l, []Value{}, ReadStateRoot)
}

func ParseFile(name string) Value {
	absPath, err := filepath.Abs(name)
	if err != nil {
		panic(fmt.Sprintf("error: %s\n", err))
	}

	file, err := os.Open(absPath)
	if err != nil {
		panic(fmt.Sprintf("error: %s\n", err))
	}

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(fmt.Sprintf("error: %s\n", err))
	}
	file.Close()

	topLevel := Parse(filepath.Base(absPath), string(contents))

	return NewCell(append([]Value{
		NewSym("do"),
		NewCell([]Value{NewSym("environment-set"), NewSym("*filename*"), NewString(filepath.Base(absPath))}),
		NewCell([]Value{NewSym("environment-set"), NewSym("*dirname*"), NewString(filepath.Dir(absPath))}),
	}, topLevel...))
}
