# Heroku Platform API

[![GoDoc](https://godoc.org/github.com/cyberdelia/heroku-go/v3?status.svg)](https://godoc.org/github.com/cyberdelia/heroku-go/v3) 

An API client interface for Heroku Platform API for the Go (golang) programming language.

## Installation

To download, build, and install the package:

```
$ go get github.com/cyberdelia/heroku-go/v3
```

## Example

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/cyberdelia/heroku-go/v3"
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
