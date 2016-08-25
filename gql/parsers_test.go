package gql

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	p "github.com/dgraph-io/dgraph/parsing"
)

func testParse(pr p.Parser, input string) {
	s := p.NewByteStream(bytes.NewBufferString(input))
	c := p.NewContext(s)
	c.Parse(pr)
}

func TestParsers(t *testing.T) {
	var sel Selection
	testParse(&sel, `friends (first: 10, after: 3) {
            }`)
	assert.EqualValues(t, "friends", sel.Name)
	assert.Len(t, sel.Args.Normal, 0)
	assert.EqualValues(t, 10, sel.Args.First)
	assert.EqualValues(t, 3, sel.Args.After)

	testParse(p.Seq{p.Maybe(wsnl), &sel}, `
    friends(xid:what) {  # xid would be ignored.
                }`)
	assert.EqualValues(t, "friends", sel.Name)
	assert.EqualValues(t, []Argument{{"xid", "what"}}, sel.Args.Normal)

	var ss SelectionSet
	testParse(p.Seq{p.Maybe(wsnl), &ss}, `{
                name,
                friends(xid:what) {  # xid would be ignored.
                }
            }`)
}
