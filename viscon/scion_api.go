package main

import "C"

import (
	"fmt"
)

//export Add
func Add() {
	fmt.Println("Add")
}

func main() {}
