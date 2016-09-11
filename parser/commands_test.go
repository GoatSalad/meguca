package parser

import (
	"reflect"

	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/types"
	r "github.com/dancannon/gorethink"
	. "gopkg.in/check.v1"
)

func (*Tests) TestParseFlip(c *C) {
	com, err := parseCommand([]byte("flip"), "a")
	c.Assert(err, IsNil)
	c.Assert(com.Type, Equals, types.Flip)
	c.Assert(reflect.TypeOf(com.Val).Kind(), Equals, reflect.Bool)
}

func (*Tests) TestDice(c *C) {
	samples := [...]struct {
		in         string
		isNil      bool
		rolls, max int
	}{
		{`d101`, true, 0, 0},
		{`11d100`, true, 0, 0},
		{`11d101`, true, 0, 0},
		{`d10`, false, 1, 10},
		{`10d100`, false, 10, 100},
	}
	for _, s := range samples {
		com, err := parseCommand([]byte(s.in), "a")
		c.Assert(err, IsNil)
		if s.isNil {
			c.Assert(com.Val, IsNil)
		} else {
			c.Assert(com.Type, Equals, types.Dice)
			val := com.Val.([]uint16)
			c.Assert(len(val), Equals, s.rolls)
		}
	}
}

func (*Tests) Test8ball(c *C) {
	answers := []string{"Yes", "No"}
	q := r.Table("boards").Insert(config.BoardConfigs{
		ID:        "a",
		Eightball: answers,
	})
	c.Assert(db.Write(q), IsNil)

	com, err := parseCommand([]byte("8ball"), "a")
	c.Assert(err, IsNil)
	c.Assert(com.Type, Equals, types.EightBall)
	val := com.Val.(string)
	if val != answers[0] && val != answers[1] {
		c.Fatalf("eightball answer not mached: %s", val)
	}
}

func (*Tests) TestPyuDisabled(c *C) {
	for _, in := range [...][]byte{pyuCommand, pcountCommand} {
		com, err := parseCommand(in, "a")
		c.Assert(err, IsNil)
		c.Assert(com, DeepEquals, types.Command{})
	}
}

func (*Tests) TestPyuAndPcount(c *C) {
	(*config.Get()).Pyu = true
	info := db.Document{ID: "info"}
	c.Assert(db.Write(r.Table("main").Insert(info)), IsNil)

	samples := [...]struct {
		in   []byte
		Type types.CommandType
		Val  int
	}{
		{pcountCommand, types.Pcount, 0},
		{pyuCommand, types.Pyu, 1},
		{pcountCommand, types.Pcount, 1},
	}

	for _, s := range samples {
		com, err := parseCommand(s.in, "a")
		c.Assert(err, IsNil)
		c.Assert(com, DeepEquals, types.Command{
			Type: s.Type,
			Val:  s.Val,
		})
	}
}
