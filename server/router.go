package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gorilla/mux"
	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi31"
)

type methodFunc struct {
	method string
	fn     func(w http.ResponseWriter, r *http.Request)
	next   *methodFunc
}

var (
	paths                       = map[string]methodFunc{}
	reflector                   = openapi31.NewReflector()
	ErrRequiredParameterMissing = errors.New("missing parameter")
)

func Init(title string, version string, description string) {
	reflector.Spec.Info.WithTitle(title).WithVersion(version).WithDescription(description)
}

func manageParameters[T any](r *http.Request, vars map[string]string, v reflect.Type, t *T) error {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if tag := field.Tag.Get("path"); tag != "" {
			v := ""
			value, found := vars[tag]
			if !found {
				required := field.Tag.Get("required")
				if required == "" || required == "true" {
					return ErrRequiredParameterMissing
				}
			} else {
				v = value
			}
			reflect.ValueOf(t).Elem().FieldByName(field.Name).SetString(v)
		} else if tag := field.Tag.Get("query"); tag != "" {
			v := ""
			value := r.URL.Query().Get(tag)
			if value == "" {
				required := field.Tag.Get("required")
				if required == "" || required == "true" {
					return ErrRequiredParameterMissing
				}
			} else {
				v = value
			}
			reflect.ValueOf(t).Elem().FieldByName(field.Name).SetString(v)
		} else if tag := field.Tag.Get("cookie"); tag != "" {
			value := ""
			cookie, err := r.Cookie(tag)
			if err != nil {
				required := field.Tag.Get("required")
				if required == "" || required == "true" {
					return ErrRequiredParameterMissing
				}
			} else {
				value = cookie.Name
			}
			reflect.ValueOf(t).Elem().FieldByName(field.Name).SetString(value)
		} else if tag := field.Tag.Get("header"); tag != "" {
			v := ""
			value := r.Header.Get(tag)
			if value == "" {
				required := field.Tag.Get("required")
				if required == "" || required == "true" {
					return ErrRequiredParameterMissing
				}
			} else {
				v = value
			}
			reflect.ValueOf(t).Elem().FieldByName(field.Name).SetString(v) // TODO assign to other kind of value
		}
	}
	return nil
}

func runHandler[T any, U any](w http.ResponseWriter, r *http.Request, path string,
	fn func(ctx context.Context, t *T) (*U, error),
) {
	ctx := r.Context()
	var t *T
	_, ok := any(t).(*struct{})
	if !ok {
		t = new(T)
		v := reflect.TypeOf(t).Elem()
		vars := mux.Vars(r)
		if v.Kind() != reflect.Struct {
			panic("bad type" + v.Kind().String())
		}
		if err := manageParameters(r, vars, v, t); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var found bool
		for i := 0; i < v.NumField(); i++ {
			if tag := v.Field(i).Tag.Get("json"); tag != "" {
				found = true
				break
			}
		}
		if found {
			if err := json.NewDecoder(r.Body).Decode(t); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}
	u, err := fn(ctx, t)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if u != nil {
		if err := json.NewEncoder(w).Encode(u); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Possible tag https://github.com/swaggest/jsonschema-go
func AddHanler[T any, U any](path string, method string, fn func(ctx context.Context, t *T) (*U, error)) error {
	np := methodFunc{
		method: method,
		fn: func(w http.ResponseWriter, r *http.Request) {
			if method == r.Method {
				runHandler[T, U](w, r, path, fn)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
		},
		next: &methodFunc{},
	}
	op, err := reflector.NewOperationContext(method, path)
	if err != nil {
		return err
	}
	path = op.PathPattern()
	var t *T
	if _, ok := any(t).(*struct{}); !ok {
		op.AddReqStructure(new(T), openapi.WithContentType("application/json"))
	}
	var u *U
	if _, ok := any(u).(*struct{}); !ok {
		op.AddRespStructure(new(U), openapi.WithContentType("application/json"))
	}
	op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusInternalServerError))
	op.AddRespStructure(nil, openapi.WithHTTPStatus(http.StatusBadRequest))
	if err := reflector.AddOperation(op); err != nil {
		return err
	}
	if p, found := paths[path]; found {
		np.next = &p
	}
	paths[path] = np
	return nil
}

func checkMethod(mf *methodFunc, w http.ResponseWriter, r *http.Request) {
	if mf.method == r.Method {
		mf.fn(w, r)
	} else if mf.next != nil {
		checkMethod(mf.next, w, r)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func Serve(router *mux.Router) error {
	for k, v := range paths {
		if v.next == nil {
			router.HandleFunc(k, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == v.method {
					v.fn(w, r)
					return
				}
				if r.Method == http.MethodOptions {
					w.Header().Set("Allow", strings.Join([]string{http.MethodOptions, v.method}, ", "))
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.WriteHeader(http.StatusMethodNotAllowed)
			})
		} else {
			router.HandleFunc(k, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == v.method {
					v.fn(w, r)
					return
				}
				if r.Method == http.MethodOptions {
					options := []string{http.MethodOptions, v.method}
					p := v.next
					for p != nil {
						options = append(options, p.method)
						p = p.next
					}
					w.Header().Set("Allow", strings.Join(options, ", "))
					w.WriteHeader(http.StatusNoContent)
					return
				}
				checkMethod(&v, w, r)
			})
		}
	}
	for k := range paths {
		delete(paths, k)
	}
	out, err := reflector.Spec.MarshalYAML()
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
