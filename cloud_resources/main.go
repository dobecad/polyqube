package main

import "fmt"

func main() {
	if err := CreateResources(); err != nil {
		fmt.Printf("Error creating resources: %s\n", err)
	}
}