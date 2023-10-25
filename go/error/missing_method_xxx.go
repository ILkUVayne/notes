package main

import "fmt"

type Eater interface {
	eat(s string)
}

type Dog struct{}

func (d *Dog) eat(s string) {
	fmt.Printf("eat %s", s)
}

var animalMaps = map[string]interface{}{
	"dog": Dog{},
}

func doEat(animal interface{}, s string) {
	animal.(Eater).eat(s)
}

func main() {
	doEat(animalMaps["dog"], "banana")
}
