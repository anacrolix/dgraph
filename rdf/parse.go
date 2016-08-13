package rdf

import (
	"fmt"
	"strconv"
	"unicode"

	p "github.com/dgraph-io/dgraph/parser"
)

var pSubject = p.ParseFunc(func(c p.Context) p.Context {
	var i int
	i, c = p.OneOf{pIriRef, pBNLabel}.ParseIndex(c)
	var v string
	switch i {
	case 0:
		v = c.Value().(string)
	case 1:
		v = "_:" + c.Value().(string)
	}
	return c.WithValue(v)
})

var pObject = p.ParseFunc(func(c p.Context) p.Context {
	var i int
	i, c = p.OneOf{pIriRef, pBNLabel, pLiteral}.ParseIndex(c)
	switch i {
	case 1:
		c = c.WithValue("_:" + c.Value().(string))
	}
	return c
})

type literal struct {
	Value   string
	LangTag string
}

func pStringUntilByte(b byte) p.Parser {
	return p.ParseFunc(func(c p.Context) p.Context {
		v := ""
		for c.Stream().Good() {
			_b := c.Stream().Token().Value().(byte)
			if _b == b {
				break
			}
			v += string(_b)
			c = c.NextToken()
		}
		return c.WithValue(v)
	})
}

func pNotByteIn(s string) p.Parser {
	return pPred(func(b byte) bool {
		for _, _b := range []byte(s) {
			if b == _b {
				return false
			}
		}
		return true
	})
}

var pEChar = p.ParseFunc(func(c p.Context) p.Context {
	c = c.Parse(pByte('\\'))
	if c.Stream().Err() != nil {
		return c.WithError(fmt.Errorf("incomplete echar: %s", c.Stream().Err()))
	}
	s, err := strconv.Unquote(`"\` + string(c.Stream().Token().Value().(byte)) + `"`)
	if err == nil && len(s) != 1 {
		panic(s)
	}
	return c.NextToken().WithValue(s[0]).WithError(err)
})

var pQuotedStringLiteral = p.ParseFunc(func(c p.Context) p.Context {
	c = c.Parse(pByte('"'))
	c, vs := p.MinTimes{0, p.OneOf{pNotByteIn(`\"`), pEChar}}.Parse(c)
	return c.Parse(pByte('"')).WithValue(catBytes(vs))
})

func catBytes(vs []p.Value) (ret string) {
	for _, v := range vs {
		ret += string(v.(byte))
	}
	return
}

var pLiteral = p.ParseFunc(func(c p.Context) p.Context {
	c = pQuotedStringLiteral.Parse(c)
	l := literal{
		Value: c.Value().(string),
	}
	i, _c := p.OneOf{pLangTag, p.ParseFunc(func(c p.Context) p.Context {
		c = c.Parse(pBytes("^^"))
		return c.Parse(pIriRef)
	})}.ParseIndex(c)
	if _c.Good() {
		switch i {
		case 0:
			l.LangTag = _c.Value().(string)
		case 1:
			l.Value += "@@" + _c.Value().(string)
		}
		c = _c
	}
	return c.WithValue(l)
})

var pLangTag = p.ParseFunc(func(c p.Context) p.Context {
	c = c.Parse(pByte('@'))
	s := ""
	c, vs := p.MinTimes{1, pPred(func(b byte) bool {
		return unicode.IsLetter(rune(b))
	})}.Parse(c)
	for _, v := range vs {
		s += string(v.(byte))
	}
	c, vs = p.MinTimes{0, pPred(func(b byte) bool {
		return b == '-' || unicode.IsLetter(rune(b)) || unicode.IsNumber(rune(b))
	})}.Parse(c)
	for _, v := range vs {
		s += string(v.(byte))
	}
	return c.WithValue(s)
})

func pPred(pred func(byte) bool) p.Parser {
	return p.ParseFunc(func(c p.Context) p.Context {
		if c.Stream().Err() != nil {
			return c.WithError(c.Stream().Err())
		}
		_b := c.Stream().Token().Value().(byte)
		if pred(_b) {
			return c.WithValue(_b).NextToken()
		}
		return c.WithError(fmt.Errorf("%q does not satisfy %s", _b, pred))
	})
}

func pByte(b byte) p.Parser {
	return p.ParseFunc(func(c p.Context) p.Context {
		if c.Stream().Err() != nil {
			return c.WithError(c.Stream().Err())
		}
		_b := c.Stream().Token().Value().(byte)
		if _b != b {
			return c.WithError(fmt.Errorf("expected %q but got %q", b, _b))
		}
		return c.WithValue(b).NextToken()
	})
}

var pIriRef = p.ParseFunc(func(c p.Context) p.Context {
	c = c.Parse(pByte('<'))
	c = c.Parse(pStringUntilByte('>'))
	v := c.Value()
	c = c.Parse(pByte('>'))
	return c.WithValue(v)
})

var pBNLabel = p.ParseFunc(func(c p.Context) p.Context {
	c = c.Parse(pBytes("_:"))
	c = c.Parse(notWS)
	return c
})

func predStar(pred func(b byte) bool) p.Parser {
	return p.ParseFunc(func(c p.Context) p.Context {
		v := ""
		for c.Stream().Err() == nil {
			_b := c.Stream().Token().Value().(byte)
			if !pred(_b) {
				break
			}
			v += string(_b)
			c = c.NextToken()
		}
		return c.WithValue(v)
	})
}

var pPredicate = pIriRef

var notWS = predStar(func(b byte) bool {
	return !unicode.IsSpace(rune(b))
})

func pBytes(bs string) p.Parser {
	return p.ParseFunc(func(c p.Context) p.Context {
		for _, b := range []byte(bs) {
			if err := c.Stream().Err(); err != nil {
				return c.WithError(fmt.Errorf("expected %q but got %s", b, err))
			}
			_b := c.Stream().Token().Value().(byte)
			if _b != b {
				return c.WithError(fmt.Errorf("expected %q but saw %q", b, _b))
			}
			c = c.NextToken()
		}
		return c
	})
}

var pWS = predStar(func(b byte) bool {
	return unicode.IsSpace(rune(b))
})

var pLabel = pSubject

var pNQuadStatement = p.ParseFunc(func(c p.Context) p.Context {
	var ret NQuad
	c = c.Parse(pSubject)
	ret.Subject = c.Value().(string)
	c = c.Parse(pWS)
	c = c.Parse(pPredicate)
	ret.Predicate = c.Value().(string)
	c = c.Parse(pWS)
	c = c.Parse(pObject)
	switch v := c.Value().(type) {
	case string:
		ret.ObjectId = v
	case literal:
		ret.ObjectValue = []byte(v.Value)
		if v.LangTag != "" {
			ret.Predicate += "." + v.LangTag
		}
	}
	c = c.Parse(pWS)
	_c := c.Parse(pLabel)
	if _c.Good() {
		c = _c
		ret.Label = c.Value().(string)
	}
	c = c.Parse(pWS)
	c = c.Parse(pByte('.'))
	return c.WithValue(ret)
})
