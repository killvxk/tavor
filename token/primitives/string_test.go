package primitives

import (
	"github.com/zimmski/tavor/token"
	"testing"

	. "github.com/stretchr/testify/assert"

	"github.com/zimmski/tavor/test"
)

func TestStringTokensToBeTokens(t *testing.T) {
	var tok *token.Token

	Implements(t, tok, NewConstantString("a"))
}

func TestConstantString(t *testing.T) {
	o := NewConstantString("abc")
	Equal(t, "abc", o.String())

	r := test.NewRandTest(0)
	o.Fuzz(r)
	Equal(t, "abc", o.String())
}
