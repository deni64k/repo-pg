package repo

import (
	"fmt"
	"log"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/medvednikov/pg"
)

var (
	db *pg.DB

	typeMap    = make(map[string]string, 0)
	queryDebug string
)

func Init(options *pg.Options) {
	db = pg.Connect(options)
}

func Insert(o interface{}) error {
	query := "INSERT INTO " + typeName(o) + " ("
	typ := reflect.TypeOf(o).Elem()
	cols := make([]string, 0)
	vals := make([]string, 0)
	for i := 0; i < typ.NumField(); i++ {
		fieldName := strings.ToLower(typ.Field(i).Name)
		tag := typ.Field(i).Tag.Get("pg")
		tag2 := typ.Field(i).Tag.Get("repopg")
		//fmt.Println(fieldName, tag)
		if fieldName == "id" || tag == "-" || tag2 == "-" {
			continue
		}
		if tag != "" {
			fieldName = tag
		}
		cols = append(cols, fieldName)
		vals = append(vals, "?"+fieldName)
	}
	query += strings.Join(cols, ",") + ") VALUES (" +
		strings.Join(vals, ",") + ") RETURNING id"
	// Fetch the ID and assign it to the object
	id := &struct{ ID int }{0}
	res, err := db.QueryOne(id, query, o)
	//log.Println("Q=", query)
	h(err)
	val := reflect.ValueOf(o).Elem()
	val.FieldByName("ID").SetInt(int64(id.ID))
	fmt.Println("repo.Insert "+query, "Res=", res)
	return err
}

func UpdateFields(o interface{}, cols ...string) {
	query := "UPDATE  " + typeName(o) + " SET "
	for i, col := range cols {
		col = strings.ToLower(col)
		query += col + "= ?" + col
		if i != len(cols)-1 {
			query += ","
		}
	}
	query += " WHERE id=?id"
	_, err := db.ExecOne(query, o)
	h(err)
	fmt.Println("repo.UpdateFields " + query)
}

func ExecOne(q string, args ...interface{}) {
	_, err := db.ExecOne(q, args...)
	h(err)
}

func Exec(q string, args ...interface{}) {
	_, err := db.Exec(q, args...)
	h(err)
}

// Retrieve searches for a row with a given id and binds the result to a given
// object
func Retrieve(res interface{}, id int) {
	err := SelectOne(res, "WHERE id=?", id)
	h(err)
}

func SelectOne(res interface{}, query string, args ...interface{}) error {
	if strings.Index(query, "SELECT") == -1 {
		query = selectWhere(res, query)
	}

	// Handler pointer to pointer
	t := reflect.TypeOf(res).Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()                        // User
		val := reflect.New(t)               // *User
		dest := reflect.ValueOf(res).Elem() // *User
		_, err := db.QueryOne(val.Interface(), query, args...)
		h(err)
		if err != nil {
			return err
		}

		if dest.IsNil() {
			dest.Set(reflect.New(t))
		}
		dest.Elem().Set(val.Elem()) // User = User
		return err
	}

	var err error
	timeout(func() {
		_, err = db.QueryOne(res, query, args...)
		h(err)
	})
	return err
}

// Select executes a search using a given query and binds the result to a given
// slice
// Example:
// var users []*User
// repo.Select(&users, "Age > 18")
func Select(res pg.Collection, query string, args ...interface{}) error {
	var err error
	if strings.Index(query, "SELECT") == -1 {
		query = selectWhere(res, query)
	}
	timeout(func() {
		_, err = db.Query(res, query, args...)
		h(err)
	})
	return err
}

func SelectInt(query string, args ...interface{}) int {
	var n int
	_, err := db.QueryOne(pg.LoadInto(&n), query, args...)
	h(err)
	return n
}

func Update(query string, o interface{}) {
	query = "UPDATE " + typeName(o) + " " + query + " WHERE id=?id"
	fmt.Println("repo Update", query)
	_, err := db.ExecOne(query, o)

	h(err)

}

//////////////////////////////////////////////////////////////////////////////

// h handles errors (logs them)
func h(err error) {
	if err != nil && err != pg.ErrNoRows {
		log.Println("repo-pg sql error:", err)
		log.Println("query:", queryDebug, "\n")
		log.Println(string(debug.Stack()))
		//panic(err)
	}
}

// selectWhere is a helper method that builds a SELECT * FROM query
func selectWhere(res interface{}, query string) string {
	name := typeName(res)
	//if Debug {
	queryDebug = "SELECT * FROM " + name + " " + query
	//}
	return "SELECT * FROM " + name + " " + query
}

func typeName(obj interface{}) string {
	name := reflect.TypeOf(obj).String()
	name = name[strings.Index(name, ".")+1:]
	if name[len(name)-1] == 's' {
		name = name[:len(name)-1]
	}
	if name == "User" {
		name = "HaUser"
	}

	return name
}

func timeout(fn func()) {
	c := make(chan bool, 1)
	go func() {
		fn()
		c <- true
	}()
	select {
	case res := <-c:
		_ = res
	case <-time.After(time.Millisecond * 500):
		log.Println("pg timeout 1/2 seconds")
		fn()
	}
}
