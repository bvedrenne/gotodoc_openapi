# gotodoc_openapi
API to allow user to create route to server using gorilla/mux auto creating OpenAPI documentation of server.

## How to use

Simple POST:
```go
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
```
Will output:
```yaml
openapi: 3.1.0
info:
  description: API in swagger
  title: API
  version: 0.0.1
paths:
  /test:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/StructTest'
      responses:
        "200":
          content:
            application/json:
              schema:
                type:
                - "null"
                - integer
          description: OK
        "400":
          description: Bad Request
        "500":
          description: Internal Server Error
components:
  schemas:
    StructTest:
      properties:
        i:
          type: integer
        s:
          type: string
      type: object
```

GET with parameter from different origin:
```go
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
```
Will output:
```yaml
openapi: 3.1.0
info:
  description: API in swagger
  title: API
  version: 0.0.1
paths:
  /test/{id}:
    get:
      parameters:
      - in: query
        name: search
        required: false
        schema:
          type: string
      - in: path
        name: id
        required: true
        schema:
          type: string
      - in: cookie
        name: who
        required: false
        schema:
          type: string
      - in: header
        name: head
        required: false
        schema:
          type: string
      responses:
        "400":
          description: Bad Request
        "500":
          description: Internal Server Error
```
