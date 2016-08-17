package parsing

import (
	"fmt"
	"reflect"
)

func ParseErr(s Stream, p Parser) (_s Stream, err SyntaxError) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if se, ok := r.(SyntaxError); ok {
			err = se
			return
		}
		panic(r)
	}()
	_s = Parse(s, p)
	return
}

func ParserName(p Parser) string {
	t := reflect.ValueOf(p).Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Name() == "" {
		panic(p)
	}
	return t.Name()
}

func recoverSyntaxError(f func(SyntaxError)) {
	r := recover()
	if r == nil {
		return
	}
	se, ok := r.(SyntaxError)
	if !ok {
		panic(r)
	}
	f(se)
}

func Parse(s Stream, p Parser) Stream {
	defer recoverSyntaxError(func(se SyntaxError) {
		se.AddContext(SyntaxErrorContext{
			Parser: p,
			Stream: s,
		})
		panic(se)
	})
	return p.Parse(s)
}

type Value interface{}

type errNoMatch struct {
	ps []Parser
}

func (me errNoMatch) Error() string {
	return fmt.Sprintf("couldn't match on of %s", me.ps)
}

func OneOf(s Stream, ps ...Parser) (Stream, int) {
	for i, p := range ps {
		s1, err := ParseErr(s, p)
		if err == nil {
			return s1, i
		}
	}
	panic(NewSyntaxError(SyntaxErrorContext{Err: errNoMatch{ps}}))
}

type ParseFunc func(Stream) Stream

func (pf ParseFunc) Parse(s Stream) Stream {
	return pf(s)
}

func Maybe(s Stream, p Parser) Stream {
	_s, err := ParseErr(s, p)
	if err != nil {
		return s
	}
	return _s
}
