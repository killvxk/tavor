package strategy

import (
	"strings"
	"testing"

	. "github.com/zimmski/tavor/test/assert"

	"github.com/zimmski/tavor/parser"
	"github.com/zimmski/tavor/test"
	"github.com/zimmski/tavor/token/constraints"
	"github.com/zimmski/tavor/token/lists"
	"github.com/zimmski/tavor/token/primitives"
	"github.com/zimmski/tavor/token/sequences"
)

func TestAlmostAllPermutationsStrategygetLevel(t *testing.T) {
	o := newAlmostAllPermutations(nil)

	{
		a := primitives.NewConstantInt(1)
		b := constraints.NewOptional(primitives.NewConstantInt(2))
		c := primitives.NewPointer(primitives.NewConstantInt(3))
		d := lists.NewConcatenation(a, b, c)

		level := o.getLevel(d, false)

		Equal(t, level, []almostAllPermutationsLevel{
			almostAllPermutationsLevel{
				parent:      d,
				tokenIndex:  -1,
				permutation: 0,
			},
		})

		level = o.getLevel(d, true)

		Equal(t, level, []almostAllPermutationsLevel{
			almostAllPermutationsLevel{
				parent:      d,
				tokenIndex:  0,
				permutation: 0,
			},
			almostAllPermutationsLevel{
				parent:      d,
				tokenIndex:  1,
				permutation: 0,
			},
			almostAllPermutationsLevel{
				parent:      d,
				tokenIndex:  2,
				permutation: 0,
			},
		})
	}
}

func TestAlmostAllPermutationsStrategy(t *testing.T) {
	r := test.NewRandTest(1)

	{
		a := primitives.NewConstantInt(1)

		ch, err := NewAlmostAllPermutations(a, r)
		Nil(t, err)

		_, ok := <-ch
		True(t, ok)
		Equal(t, "1", a.String())
		ch <- struct{}{}

		_, ok = <-ch
		False(t, ok)
	}
	{
		a := constraints.NewOptional(primitives.NewConstantInt(1))

		ch, err := NewAlmostAllPermutations(a, r)
		Nil(t, err)

		var got []string

		for i := range ch {
			got = append(got, a.String())

			ch <- i
		}

		Equal(t, got, []string{
			"",
			"1",
		})
	}
	{
		a := lists.NewOne(
			primitives.NewConstantInt(1),
			primitives.NewConstantInt(2),
			primitives.NewConstantInt(3),
		)

		ch, err := NewAlmostAllPermutations(a, r)
		Nil(t, err)

		var got []string

		for i := range ch {
			got = append(got, a.String())

			ch <- i
		}

		Equal(t, got, []string{
			"1",
			"2",
			"3",
		})
	}
	{
		a := constraints.NewOptional(primitives.NewConstantInt(1))
		b := constraints.NewOptional(primitives.NewConstantInt(2))
		c := constraints.NewOptional(primitives.NewConstantInt(3))
		abc := lists.NewConcatenation(a, b, c)

		ch, err := NewAlmostAllPermutations(abc, r)
		Nil(t, err)

		var got []string

		for i := range ch {
			got = append(got, abc.String())

			ch <- i
		}

		Equal(t, got, []string{
			"",
			"1",
			"2",
			"12",
			"3",
			"13",
			"23",
			"123",
		})
	}
	{
		abc := lists.NewConcatenation(
			constraints.NewOptional(lists.NewConcatenation(
				lists.NewOne(
					primitives.NewConstantInt(1),
					primitives.NewConstantInt(2),
				),
				primitives.NewConstantInt(3),
			)),
			primitives.NewConstantInt(4),
		)

		ch, err := NewAlmostAllPermutations(abc, r)
		Nil(t, err)

		var got []string

		for i := range ch {
			got = append(got, abc.String())

			ch <- i
		}

		Equal(t, got, []string{
			"4",
			"134",
			"234",
		})
	}
	{
		a := constraints.NewOptional(primitives.NewConstantInt(1))
		b := primitives.NewConstantInt(2)
		c := constraints.NewOptional(primitives.NewConstantInt(3))
		d := lists.NewConcatenation(a, b, c)

		ch, err := NewAlmostAllPermutations(d, r)
		Nil(t, err)

		_, ok := <-ch
		True(t, ok)
		Equal(t, "2", d.String())
		ch <- struct{}{}

		_, ok = <-ch
		True(t, ok)
		Equal(t, "12", d.String())
		ch <- struct{}{}

		_, ok = <-ch
		True(t, ok)
		Equal(t, "23", d.String())
		ch <- struct{}{}

		_, ok = <-ch
		True(t, ok)
		Equal(t, "123", d.String())
		ch <- struct{}{}

		_, ok = <-ch
		False(t, ok)

		// rerun
		ch, err = NewAlmostAllPermutations(d, r)
		Nil(t, err)

		_, ok = <-ch
		True(t, ok)
		Equal(t, "2", d.String())

		close(ch)

		// run with range
		var got []string

		ch, err = NewAlmostAllPermutations(d, r)
		Nil(t, err)
		for i := range ch {
			got = append(got, d.String())

			ch <- i
		}

		Equal(t, got, []string{
			"2",
			"12",
			"23",
			"123",
		})
	}
	{
		a1 := constraints.NewOptional(primitives.NewConstantInt(1))
		a2 := constraints.NewOptional(primitives.NewConstantInt(11))
		a := constraints.NewOptional(lists.NewConcatenation(a1, a2, primitives.NewConstantString("a")))
		b := constraints.NewOptional(primitives.NewConstantString("b"))
		c := lists.NewConcatenation(a, b, primitives.NewConstantString("c"))
		d := constraints.NewOptional(c)

		var got []string

		ch, err := NewAlmostAllPermutations(d, r)
		Nil(t, err)
		for i := range ch {
			got = append(got, d.String())

			ch <- i
		}

		Equal(t, got, []string{
			"",
			"c",
			"ac",
			"1ac",
			"11ac",
			"111ac",
			"bc",
			"abc",
			"1abc",
			"11abc",
			"111abc",
		})
	}
	{
		a := lists.NewConcatenation(
			constraints.NewOptional(primitives.NewConstantInt(1)),
			constraints.NewOptional(primitives.NewConstantInt(2)),
		)
		b := lists.NewRepeat(a, 0, 2)

		var got []string

		ch, err := NewAlmostAllPermutations(b, r)
		Nil(t, err)
		for i := range ch {
			got = append(got, b.String())

			ch <- i
		}

		Equal(t, got, []string{
			"",
			"",
			"1",
			"2",
			"12",
			"12",
			"112",
			"212",
			"12",
			"121",
			"122",
			"1212",
		})
	}
	{
		s := sequences.NewSequence(10, 2)

		Equal(t, 10, s.Next())
		Equal(t, 12, s.Next())

		a := lists.NewConcatenation(
			constraints.NewOptional(primitives.NewConstantString("a")),
			constraints.NewOptional(primitives.NewConstantString("b")),
			s.ResetItem(),
			s.Item(),
			s.ExistingItem(nil),
		)
		b := lists.NewRepeat(a, 0, 1)

		var got []string

		ch, err := NewAlmostAllPermutations(b, r)
		Nil(t, err)
		for i := range ch {
			got = append(got, b.String())

			ch <- i
		}

		Equal(t, got, []string{
			"",
			"1010",
			"a1010",
			"b1010",
			"ab1010",
		})
	}
	{
		// correct sequence and multi-OR token behaviour

		o, err := parser.ParseTavor(strings.NewReader(`
			$Id Sequence = start: 2,
				step: 2

			ExistingLiteral = 1,
				| $Id.Existing,
				| ${Id.Existing + 1}

			And = $Id.Next " " ExistingLiteral " " ExistingLiteral

			START = $Id.Reset And
		`))
		Nil(t, err)

		var got []string

		ch, err := NewAlmostAllPermutations(o, r)
		Nil(t, err)
		for i := range ch {
			got = append(got, o.String())

			ch <- i
		}

		/* TODO original result, should we go back to this?
		Equal(t, got, []string{
			"2 1 1",
			"2 2 1",
			"2 3 1",
			"2 1 2",
			"2 2 2",
			"2 3 2",
			"2 1 3",
			"2 2 3",
			"2 3 3",
		})
		*/
		Equal(t, got, []string{
			"2 1 1",
			"2 2 1",
			"2 3 1",
			"2 3 2",
			"2 3 3",
		})
	}
	{
		// Correct list pointer behaviour

		o, err := parser.ParseTavor(strings.NewReader(`
			$Id Sequence = start: 2,
				step: 2

			Inputs = *(Input)
			Input = $Id.Next

			START = $Id.Reset Inputs
		`))
		Nil(t, err)

		var got []string

		ch, err := NewAlmostAllPermutations(o, r)
		Nil(t, err)
		for i := range ch {
			got = append(got, o.String())

			ch <- i
		}

		Equal(t, got, []string{
			"",
			"2",
			"24",
		})
	}
	{
		// Correct sequence deep or behaviour

		o, err := parser.ParseTavor(strings.NewReader(`
			$Id Sequence = start: 2,
				step: 2

			A = $Id.Next
			B = $Id.Next (1 | 2 | 3)

			START = $Id.Reset A B
		`))
		Nil(t, err)

		var got []string

		ch, err := NewAlmostAllPermutations(o, r)
		Nil(t, err)
		for i := range ch {
			got = append(got, o.String())

			ch <- i
		}

		Equal(t, got, []string{
			"241",
			"242",
			"243",
		})
	}
	{
		// correct unqiue behavior
		o, err := parser.ParseTavor(strings.NewReader(`
				Items = "a" "b" "c"
				START = Items " -> " $Items.Unique
		`))
		Nil(t, err)

		var got []string

		ch, err := NewAlmostAllPermutations(o, r)
		Nil(t, err)
		for i := range ch {
			got = append(got, o.String())

			ch <- i
		}

		Equal(t, got, []string{
			"abc -> a",
			"abc -> b",
			"abc -> c",
		})
	}
	{
		// check if the strategy really works as expected
		o, err := parser.ParseTavor(strings.NewReader(`
				START = +2(?(1)?(2))
		`))
		Nil(t, err)

		var got []string

		ch, err := NewAlmostAllPermutations(o, r)
		Nil(t, err)
		for i := range ch {
			got = append(got, o.String())

			ch <- i
		}

		Equal(t, got, []string{
			"12",
			"112",
			"212",
			"12",
			"121",
			"122",
			"1212",
		})
	}
	{
		// sequences should always start reseted
		validateTavorAlmostAllPermutations(
			t,
			`
				$Id Sequence = start: 0,
					step:  2

				START = +1,5($Id.Next " ")
			`,
			[]string{
				"0 ",
				"0 2 ",
				"0 2 4 ",
				"0 2 4 6 ",
				"0 2 4 6 8 ",
			},
		)
	}
}

func validateTavorAlmostAllPermutations(t *testing.T, format string, expect []string) {
	r := test.NewRandTest(1)

	o, err := parser.ParseTavor(strings.NewReader(format))
	Nil(t, err)

	var got []string

	ch, err := NewAlmostAllPermutations(o, r)
	Nil(t, err)
	for i := range ch {
		got = append(got, o.String())

		ch <- i
	}

	Equal(t, expect, got)
}

func TestAlmostAllPermutationsStrategyLoopDetection(t *testing.T) {
	testStrategyLoopDetection(t, NewAlmostAllPermutations)
}
