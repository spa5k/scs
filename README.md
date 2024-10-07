# SCS: HTTP Session Management for Go

[![GoDoc](https://godoc.org/github.com/spa5k/scs?status.png)](https://pkg.go.dev/github.com/spa5k/scs?tab=doc)
[![Go report card](https://goreportcard.com/badge/github.com/spa5k/scs)](https://goreportcard.com/report/github.com/spa5k/scs)
[![Test coverage](http://gocover.io/_badge/github.com/spa5k/scs)](https://gocover.io/github.com/spa5k/scs)

## Features

- Automatic loading and saving of session data via middleware.
- Choice of 19 different server-side session stores including PostgreSQL, MySQL, MSSQL, SQLite, Redis and many others. Custom session stores are also supported.
- Supports multiple sessions per request, 'flash' messages, session token regeneration, idle and absolute session timeouts, and 'remember me' functionality.
- Easy to extend and customize. Communicate session tokens to/from clients in HTTP headers or request/response bodies.
- Efficient design. Smaller, faster and uses less memory than [gorilla/sessions](https://github.com/gorilla/sessions).

## Instructions

- [SCS: HTTP Session Management for Go](#scs-http-session-management-for-go)
	- [Features](#features)
	- [Instructions](#instructions)
		- [Installation](#installation)
		- [Basic Use](#basic-use)
		- [Configuring Session Behavior](#configuring-session-behavior)
		- [For more documentation, see the SCS documentation.](#for-more-documentation-see-the-scs-documentation)

### Installation

This package requires Go 1.12 or newer.

```sh
go get github.com/spa5k/scs
```

Please use [versioned releases](https://github.com/spa5k/scs/releases). Code in tip may contain experimental features which are subject to change.

### Basic Use

SCS implements a session management pattern following the [OWASP security guidelines](https://github.com/OWASP/CheatSheetSeries/blob/master/cheatsheets/Session_Management_Cheat_Sheet.md). Session data is stored on the server, and a randomly-generated unique session token (or _session ID_) is communicated to and from the client in a session cookie.

```go
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/spa5k/scs"
)

// Create your router.
router := chi.NewMux()

// Wrap the router with Huma to create an API instance.
api := humachi.New(router, huma.DefaultConfig("My API", "1.0.0"))

api.UseMiddleware(sessionManager.LoadAndSave)

huma.Register(api, huma.Operation{
	Method: http.MethodPut,
	Path:   "/put",
}, func(ctx context.Context, input *struct {
	Cookie string `header:"Cookie"`
}) (*struct {
	Status int `json:"status"`
}, error,
) {
	sessionManager.Put(ctx, "foo", "bar")
	return &struct {
		Status int `json:"status"`
	}{Status: http.StatusOK}, nil
})

huma.Register(api, huma.Operation{
	Method: http.MethodGet,
	Path:   "/get",
}, func(ctx context.Context, input *struct {
	Session string `cookie:"session"`
}) (*struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}, error,
) {
	v := sessionManager.Get(ctx, "foo")
	if v == nil {
		return nil, huma.NewError(http.StatusInternalServerError, "foo does not exist in session")
	}

	return &struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}{Status: http.StatusOK, Body: v.(string)}, nil
})

http.ListenAndServe(":4000", router)
```

```
$ curl -i --cookie-jar cj --cookie cj localhost:4000/put
HTTP/1.1 200 OK
Cache-Control: no-cache="Set-Cookie"
Set-Cookie: session=lHqcPNiQp_5diPxumzOklsSdE-MJ7zyU6kjch1Ee0UM; Path=/; Expires=Sat, 27 Apr 2019 10:28:20 GMT; Max-Age=86400; HttpOnly; SameSite=Lax
Vary: Cookie
Date: Fri, 26 Apr 2019 10:28:19 GMT
Content-Length: 0

$ curl -i --cookie-jar cj --cookie cj localhost:4000/get
HTTP/1.1 200 OK
Date: Fri, 26 Apr 2019 10:28:24 GMT
Content-Length: 21
Content-Type: text/plain; charset=utf-8

Hello from a session!
```

### Configuring Session Behavior

Session behavior can be configured via the `SessionManager` fields.

```go
sessionManager = scs.New()
sessionManager.Lifetime = 3 * time.Hour
sessionManager.IdleTimeout = 20 * time.Minute
sessionManager.Cookie.Name = "session_id"
sessionManager.Cookie.Domain = "example.com"
sessionManager.Cookie.HttpOnly = true
sessionManager.Cookie.Path = "/example/"
sessionManager.Cookie.Persist = true
sessionManager.Cookie.SameSite = http.SameSiteStrictMode
sessionManager.Cookie.Secure = true
```

Documentation for all available settings and their default values can be [found here](https://pkg.go.dev/github.com/spa5k/scs#SessionManager).


### For more documentation, see the [SCS documentation](https://pkg.go.dev/github.com/alexedwards/scs/v2).