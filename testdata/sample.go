//go:build ignore
package testdata
import "errors"

type MyStruct struct {
	Name  string
	Value int
}


func NewMyStruct(name string) *MyStruct {
	return &MyStruct{Name: name}
}



func (m *MyStruct) Run() error {
	if m.Name == "" {
		return errors.New("no name")
	}
	return nil
}

func helperFunc(x int) int {
	return x * 2
}

const (

	MyConst = 42
)



func overloaded() string {
	return "hello"
}




var lineonly = 0

var noendVar = "test"
