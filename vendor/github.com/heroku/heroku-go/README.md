# Heroku Platform API

[![GoDoc](https://godoc.org/github.com/heroku/heroku-go/v4?status.svg)](https://godoc.org/github.com/heroku/heroku-go/v4)

An API client interface for Heroku Platform API for the Go (golang)
programming language.

## Usage

	import "github.com/heroku/heroku-go"

## Example

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/heroku/heroku-go"
)

var (
	username = flag.String("username", "", "api username")
	password = flag.String("password", "", "api password")
)

func main() {
	log.SetFlags(0)
	flag.Parse()

	heroku.DefaultTransport.Username = *username
	heroku.DefaultTransport.Password = *password

	h := heroku.NewService(heroku.DefaultClient)
	addons, err := h.AddOnList(context.TODO(), &heroku.ListRange{Field: "name"})
	if err != nil {
		log.Fatal(err)
	}
	for _, addon := range addons {
		fmt.Println(addon.Name)
	}
}
```
