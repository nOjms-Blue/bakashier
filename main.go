package main

import (
	"fmt"
	
	"bakashier/core"
)

func main() {
	fmt.Println("Hello, World!")
	
	core.Backup("backup", "dist", "password")
}