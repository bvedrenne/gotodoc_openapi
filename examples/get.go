package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bvedrenne/gotodoc_openapi/server"
	"github.com/gorilla/mux"
)

type GetStructTest struct {
	ID     string `path:"id"`
	Search string `query:"search" required:"false"`
	Who    string `cookie:"who" required:"false"`
	Head   string `header:"head" required:"false"`
}

func main() {
	server.Init("API", "0.0.1", "API in swagger")
	if err := server.AddHanler[GetStructTest, struct{}]("/test/{id}", http.MethodGet, func(ctx context.Context, t *GetStructTest) (*struct{}, error) {
		fmt.Println("OK Handler with ID")
		if t.ID == "12" {
			fmt.Println("Good value")
		}
		fmt.Printf("Search text: %s\n", t.Search)
		fmt.Printf("Who text: %s\n", t.Who)
		fmt.Printf("Header text: %s\n", t.Head)
		return nil, nil
	}); err != nil {
		panic(err)
	}
	myRouter := mux.NewRouter().StrictSlash(true)
	if err := server.Serve(myRouter); err != nil {
		panic(err)
	}
}
