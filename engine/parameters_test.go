package engine

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestParameterTable(t *testing.T) {
	Convey("Given a parameter table", t, func() {
		p := NewParameterTable()
		Convey("It should allow declarations", func() {
			err := p.Declare("A")
			So(err, ShouldBeNil)
		})
		Convey("It should reject duplicate declarations", func() {
			p.Declare("A")
			err := p.Declare("A")
			So(err, ShouldNotBeNil)
		})
		Convey("It should allow values to be set and retrieved", func() {
			p.Declare("A")
			err := p.Set("A", 1)
			So(err, ShouldBeNil)
			v, ok := p.Get("A")
			So(ok, ShouldBeTrue)
			So(v, ShouldEqual, 1)
			_, ok = p.Get("B")
			So(ok, ShouldBeFalse)
		})
		Convey("It should reject a value set if the parameter has not been declared", func() {
			err := p.Set("A", 1)
			So(err, ShouldNotBeNil)
			_, ok := p.Get("A")
			So(ok, ShouldBeFalse)
		})
	})
}

func TestParameterTableAsDestination(t *testing.T) {
	Convey("Given a parameter table, stream, logger, stopper", t, func() {
		pp := NewParameterTable()
		p := NewParameterTableDestination(pp, []string{"AA", "bb"})
		pp.Declare("Aa")
		pp.Declare("Bb")
		st := NewStopper()
		l := NewConsoleLogger(Trace)
		s := NewStream([]string{"CC", "DD"}, 100)
		Convey("It should correctly populate the parameter table from valid messages", func() {
			s.Chan("") <- Message{Data: []interface{}{1, 2}}
			s.Chan("") <- Message{Data: []interface{}{3, 4}}
			close(s.Chan(""))
			p.Open(s, l, st)
			a, ok := pp.Get("Aa")
			So(ok, ShouldBeTrue)
			So(a, ShouldEqual, 3)
			b, ok := pp.Get("BB")
			So(ok, ShouldBeTrue)
			So(b, ShouldEqual, 4)
		})
	})
}
