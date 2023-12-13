package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/bvedrenne/gotodoc_openapi/server"
	"github.com/gorilla/mux"
)

type StructTest struct {
	S string `json:"s"`
	I int    `json:"i"`
}

func main() {
	server.Init("API", "0.0.1", "API in swagger")
	if err := server.AddHanler[StructTest, int]("/test", http.MethodPost, func(ctx context.Context, t *StructTest) (*int, error) {
		fmt.Println(t)
		v := 1
		return &v, nil
	}); err != nil {
		panic(err)
	}
	myRouter := mux.NewRouter().StrictSlash(true)
	if err := server.Serve(myRouter); err != nil {
		panic(err)
	}
	log.Fatal(http.ListenAndServe(":1701", myRouter))
}
