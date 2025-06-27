package main

import (
	"fmt"
	"math"
)

func unusedFunc(x int) int {
    return x * 2 
}

func calculateArea(radius float64) float64 {
    return math.Pi * radius * radius
}

func main() {
    var result = calculateArea(10)

    fmt.Println("Area is: ", result)

    var a int
    var b int = 5
    a = 5

    if b == 5 {
        fmt.Println("B is five"
    }

    if a == 5 {
        
    }

    a := 10
    fmt.Println("Shadowed a:", a

    fmt.Println("This is a very very very very very very very very very very very very long line which most linters will flag as exceeding the line length")
}
