package main

import (
	"fmt"
	
	"bakashier/core"
)

func main() {
	fmt.Println("Hello, World!")
	
	core.Backup("src", "dist", "password")
}