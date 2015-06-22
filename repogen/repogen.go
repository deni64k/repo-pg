package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
)

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Println(flag.Args())
		log.Fatalln("Usage: repogen [type]")

	}

	typ := flag.Args()[0]

	s := `package m

import repo "github.com/medvednikov/repo-pg"

type TYPE struct {
	ID int
}

type TYPEs []*TYPE

func (c *TYPEs) NewRecord() interface{} {
	o := &TYPE{}
	*c = append(*c, o)
	return o
}

func RetrieveTYPE(id int) *TYPE {
	var o *TYPE
	repo.Retrieve(&o, id)
	return o
}

func (o *TYPE) Insert() {
	repo.Insert(o)
}
`
	s = strings.Replace(s, "TYPE", typ, -1)
	fmt.Println(s)
}
