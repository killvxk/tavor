package filter

import (
	"fmt"
	"sort"

	"github.com/zimmski/container/list/linkedlist"

	"github.com/zimmski/tavor/token"
	"github.com/zimmski/tavor/token/lists"
)

// Filter defines a fuzzing filter
type Filter interface {
	// Apply applies the fuzzing filter onto the token and returns a replacement token, or nil if there is no replacement.
	// If a fatal error is encountered the error return argument is not nil.
	Apply(tok token.Token) ([]token.Token, error)
}

var filterLookup = make(map[string]func() Filter)

// New returns a new fuzzing filter instance given the registered name of the filter.
// The error return argument is not nil, if the name does not exist in the registered fuzzing filter list.
func New(name string) (Filter, error) {
	filt, ok := filterLookup[name]
	if !ok {
		return nil, fmt.Errorf("unknown fuzzing filter %q", name)
	}

	return filt(), nil
}

// List returns a list of all registered fuzzing filter names.
func List() []string {
	keyFilterLookup := make([]string, 0, len(filterLookup))

	for key := range filterLookup {
		keyFilterLookup = append(keyFilterLookup, key)
	}

	sort.Strings(keyFilterLookup)

	return keyFilterLookup
}

// Register registers a fuzzing filter instance function with the given name.
func Register(name string, filt func() Filter) {
	if filt == nil {
		panic("register fuzzing filter is nil")
	}

	if _, ok := filterLookup[name]; ok {
		panic("fuzzing filter " + name + " already registered")
	}

	filterLookup[name] = filt
}

// ApplyFilters applies a set of filters onto a token.
// Filters are not applied onto filter generated tokens.
func ApplyFilters(filters []Filter, root token.Token) (token.Token, error) {
	type Pair struct {
		token  token.Token
		parent token.Token
	}

	var known = make(map[token.Token]struct{})

	var queue = linkedlist.New()

	queue.Push(&Pair{
		token:  root,
		parent: nil,
	})

	for !queue.Empty() {
		v, _ := queue.Shift()
		pair := v.(*Pair)

		tok := pair.token

		// only apply filters if the token is not from one
		if _, ok := known[tok]; !ok {
			var newTokens []token.Token

			// apply filters
			for i := range filters {
				replacement, err := filters[i].Apply(tok)
				if err != nil {
					return nil, fmt.Errorf("error in fuzzing filter %v: %s", filters[i], err)
				}

				if len(replacement) > 0 {
					newTokens = append(newTokens, replacement...)
				}
			}

			// replace if there is something to replace with
			if l := len(newTokens); l > 0 {
				for i := range newTokens {
					known[newTokens[i]] = struct{}{}
				}

				if l == 1 {
					tok = newTokens[0]
				} else {
					tok = lists.NewOne(newTokens...)
				}

				if pair.parent == nil {
					root = tok
				} else {
					switch t := pair.parent.(type) {
					case token.ForwardToken:
						t.InternalReplace(pair.token, tok)
					case token.ListToken:
						t.InternalReplace(pair.token, tok)
					}
				}
			}
		}

		// go deeper into the graph
		switch t := tok.(type) {
		case token.ForwardToken:
			c := t.InternalGet()

			queue.Push(&Pair{
				token:  c,
				parent: tok,
			})
		case token.ListToken:
			l := t.InternalLen()

			for i := 0; i < l; i++ {
				c, _ := t.InternalGet(i)

				queue.Push(&Pair{
					token:  c,
					parent: tok,
				})
			}
		}
	}

	return root, nil
}
