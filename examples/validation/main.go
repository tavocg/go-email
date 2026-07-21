package main

import (
	"fmt"
	"log"

	email "github.com/tavocg/go-email"
)

func main() {
	address, err := email.StrictParser("User.Name+signup@Example.com")
	if err != nil {
		log.Fatal(err)
	}

	address.Normalize()

	fmt.Println(address.Address())
	fmt.Println(address.IsBlacklisted([]string{"example.com"}))
}
