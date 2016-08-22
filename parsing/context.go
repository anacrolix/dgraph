package parsing

import (
	"fmt"

	"github.com/joeshaw/gengen/generic"
)

func NewContext(s Stream) *Context {
	return &Context{
		s: s,
	}
}

type Context struct {
	errs []Error
	s    Stream
}

func (me *Context) Stream() Stream {
	return me.s
}

func (me *Context) Token() generic.T {
	if me.s.Err() != nil {
		me.Fatal(fmt.Errorf("stream error: %s", me.s.Err()))
	}
	return me.s.Token()
}

func (me *Context) Advance() {
	me.s = me.s.Next()
}

func (me *Context) Good() bool {
	return me.s.Good()
}

func (me *Context) parse(p Parser, c *Context) {
	p.Parse(c)
	me.s = c.s
}

func (me *Context) newChild() *Context {
	return &Context{
		s: me.s,
	}
}

func (me *Context) TryParse(p Parser) bool {
	child := me.newChild()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		se, ok := r.(Error)
		if !ok {
			panic(r)
		}
		me.errs = append(me.errs, se)
	}()
	me.parse(p, child)
	return true
}

func (me *Context) Parse(p Parser) {
	me.parse(p, me.newChild())
}

func (me *Context) Fatal(err error) {
	panic(Error{
		Context: me,
		Err:     err,
	})
}

func (me *Context) FailNow() {
	panic(Error{
		Context: me,
	})
}
