package gql

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	p "github.com/dgraph-io/dgraph/parsing"
)

const (
	query    = "query"
	mutation = "mutation"
	first    = "first"
	after    = "after"
	offset   = "offset"
)

type comment struct{}

func (comment) Parse(c *p.Context) {
	c.Parse(p.Regexp("#[^\n]*"))
}

var newline = p.Byte('\n')

var inlineSpace = p.Pred(func(b byte) bool {
	return unicode.IsSpace(rune(b)) && b != '\n'
})

type whitespace struct{}

func (whitespace) Parse(c *p.Context) {
	c.Parse(p.Star(p.OneOf(comment{}, inlineSpace, newline)))
}

type Document struct {
	Query    *Query
	Mutation *Mutation
}

func (d *Document) Parse(c *p.Context) {
	for {
		c.Parse(wsnl)
		if !c.Good() {
			break
		}
		r := p.Regexp("(|query|mutation)")
		c.Parse(r)
		switch r.Submatches[0] {
		case "", "query":
			if d.Query != nil {
				c.Fatal(errors.New("more than one query in document"))
			}
			d.Query = new(Query)
			c.Parse(d.Query)
		case "mutation":
			if d.Mutation != nil {
				c.Fatal(errors.New("more than one mutation in document"))
			}
			d.Mutation = new(Mutation)
			c.Parse(d.Mutation)
		default:
			panic(r.Submatches)
		}
	}
}

type Query struct {
	Operation
}

type Operation struct {
	Type       string
	Name       string
	Variables  struct{}
	Directives struct{}
	Selection  Selection
}

func (op *Operation) Parse(c *p.Context) {
	oo := p.OneOf(p.Bytes(query), p.Bytes(mutation))
	if c.TryParse(oo) {
		switch oo.Index {
		case 0:
			op.Type = query
		case 1:
			op.Type = mutation
		}
	}
	c.Parse(whitespace{})
	c.Parse(p.Byte('{'))
	c.Parse(whitespace{})
	c.Parse(&op.Selection)
	c.Parse(whitespace{})
	c.Parse(p.Byte('}'))
}

// Leaves the stream at the terminating newline, or stream error.
func discardComment(_s *p.Stream) {
	s := *_s
	for s.Good() {
		b := s.Token().(byte)
		if b == '\n' {
			break
		}
		s = s.Next()
	}
	*_s = s
}

type SelectionSet []Selection

func (ss *SelectionSet) Parse(c *p.Context) {
	*ss = nil
	c.Parse(p.Byte('{'))
	for {
		c.Parse(p.Maybe(wsnl))
		var sel Selection
		if !c.TryParse(&sel) {
			break
		}
		*ss = append(*ss, sel)
		c.Parse(p.Maybe(ws))
		if !c.TryParse(p.OneOf(p.Byte(','), p.Byte('\n'))) {
			break
		}
	}
	c.Parse(p.Maybe(wsnl))
	c.Parse(p.Byte('}'))
}

type Selection struct {
	Field
}

type Field struct {
	Name       Name
	Args       Arguments
	Selections SelectionSet
}

var (
	ws   = p.Star(p.OneOf(inlineSpace, comment{}))
	wsnl = p.Star(p.OneOf(inlineSpace, comment{}, p.Byte('\n')))
)

func (f *Field) Parse(c *p.Context) {
	c.Parse(&f.Name, p.Maybe(ws, &f.Args), p.Maybe(ws, &f.Selections))
}

type Arguments struct {
	Normal []Argument
	First  int
	Offset int
	After  uint64
}

func (args *Arguments) Parse(c *p.Context) {
	args.Normal = nil
	c.Parse(p.Byte('('))
	c.ParseErr(whitespace{})
	for {
		var arg Argument
		if !c.TryParse(&arg) {
			break
		}
		switch arg.Name {
		case first:
			i, err := strconv.ParseInt(string(arg.Value), 0, 0)
			if err != nil {
				c.Fatal(err)
			}
			args.First = int(i)
		case offset:
			i, err := strconv.ParseUint(string(arg.Value), 0, 0)
			if err != nil {
				c.Fatal(err)
			}
			args.Offset = int(i)
		case after:
			i, err := strconv.ParseUint(string(arg.Value), 0, 0)
			if err != nil {
				c.Fatal(err)
			}
			args.After = uint64(i)
		default:
			args.Normal = append(args.Normal, arg)
		}
		c.ParseErr(whitespace{})
		if !c.TryParse(p.Byte(',')) {
			break
		}
		c.Parse(p.Maybe(ws))
	}
	c.Parse(p.Byte(')'))
}

type Argument struct {
	Name  Name
	Value Value
}

type Value string

func (v *Value) Parse(c *p.Context) {
	bw := p.BytesWhile{
		Pred: func(b byte) bool {
			r := rune(b)
			return !unicode.IsSpace(r) && !strings.ContainsRune("(),", r)
		},
	}
	c.Parse(&bw)
	*v = Value(bw.B)
}

func (arg *Argument) Parse(c *p.Context) {
	c.Parse(&arg.Name, p.Maybe(ws), p.Byte(':'), p.Maybe(ws), &arg.Value)
}

type Name string

func (n *Name) Parse(c *p.Context) {
	re := p.Regexp(`([_A-Za-z.][-._0-9A-Za-z]*)`)
	c.Parse(re)
	*n = Name(re.Submatches[0])
}
