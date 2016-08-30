package parsing

import (
	"fmt"
	"strings"
)

type Error struct {
	Context *Context
	Err     error
}

func (me Error) Error() string {
	var lines []string
	if me.Err == nil {
		lines = append(lines, "syntax error")
	} else {
		lines = append(lines, fmt.Sprintf("syntax error: %s", me.Err))
	}
	for c := me.Context; c != nil; c = c.Parent {
		if c.Parent == nil && c.p == nil {
			break
		}
		lines = append(lines, fmt.Sprintf("while parsing %s at %s", ParserName(c.p), c.Stream().Position()))
	}
	return strings.Join(lines, "\n")
}
