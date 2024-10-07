package scs

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func addRoutes(api huma.API, sessionManager *SessionManager) {
	api.UseMiddleware(sessionManager.LoadAndSave)

	huma.Register(api, huma.Operation{
		Method: http.MethodPut,
		Path:   "/put",
	}, func(ctx context.Context, input *struct {
		Cookie string `header:"Cookie"`
	}) (*struct {
		Status int `json:"status"`
	}, error) {
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
	}, error) {
		v := sessionManager.Get(ctx, "foo")
		if v == nil {
			return nil, huma.NewError(http.StatusInternalServerError, "foo does not exist in session")
		}

		return &struct {
			Status int    `json:"status"`
			Body   string `json:"body"`
		}{Status: http.StatusOK, Body: v.(string)}, nil
	})
}

func TestEnable(t *testing.T) {
	t.Parallel()

	sessionManager := New()

	_, api := humatest.New(t)

	// Add session middleware
	api.UseMiddleware(sessionManager.LoadAndSave)

	huma.Register(api, huma.Operation{
		Method: http.MethodPut,
		Path:   "/put",
	}, func(ctx context.Context, input *struct {
		Cookie string `header:"Cookie"`
	}) (*struct {
		Status int `json:"status"`
	}, error) {
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
	}, error) {
		v := sessionManager.Get(ctx, "foo")
		if v == nil {
			return nil, huma.NewError(http.StatusInternalServerError, "foo does not exist in session")
		}

		return &struct {
			Status int    `json:"status"`
			Body   string `json:"body"`
		}{Status: http.StatusOK, Body: v.(string)}, nil
	})

	// Test PUT request
	putResp := api.Put("/put")

	if putResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, putResp.Code)
	}

	token1 := extractTokenFromCookie(putResp.Header().Get("Set-Cookie"))
	if token1 == "" {
		t.Fatal("No session token found in PUT response")
	}

	// Make a GET request with the session cookie
	getResp := api.Get("/get", "Cookie: session="+token1)

	if getResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, getResp.Code)
	}

	// Trim any whitespace and quotes from the response body
	responseBody := strings.Trim(getResp.Body.String(), "\"\n\r\t ")

	if responseBody != "bar" {
		t.Errorf("want value %q; got %q", "bar", responseBody)
	}

	if getResp.Header().Get("Set-Cookie") != "" {
		t.Errorf("want %q; got %q", "", getResp.Header().Get("Set-Cookie"))
	}

	putResp = api.Put("/put", "Cookie: session="+token1)

	token2 := extractTokenFromCookie(putResp.Header().Get("Set-Cookie"))
	if token2 == "" {
		t.Fatal("No session token found in GET response")
	}

	if token1 != token2 {
		t.Error("want tokens to be the same")
	}

}

func extractTokenFromCookie(cookieHeader string) string {
	if cookieHeader == "" {
		return ""
	}
	parts := strings.Split(cookieHeader, ";")
	if len(parts) == 0 {
		return ""
	}
	keyValue := strings.SplitN(parts[0], "=", 2)
	if len(keyValue) != 2 {
		return ""
	}
	return keyValue[1]
}

func TestLifetime(t *testing.T) {
	t.Parallel()

	sessionManager := New()
	sessionManager.Lifetime = 500 * time.Millisecond

	_, api := humatest.New(t)

	// Add session middleware
	api.UseMiddleware(sessionManager.LoadAndSave)

	huma.Register(api, huma.Operation{
		Method: http.MethodPut,
		Path:   "/put",
	}, func(ctx context.Context, input *struct {
		Cookie string `header:"Cookie"`
	}) (*struct {
		Status int `json:"status"`
	}, error) {
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
	}, error) {
		v := sessionManager.Get(ctx, "foo")
		if v == nil {
			return nil, huma.NewError(http.StatusInternalServerError, "foo does not exist in session")
		}

		return &struct {
			Status int    `json:"status"`
			Body   string `json:"body"`
		}{Status: http.StatusOK, Body: v.(string)}, nil
	})

	putResp := api.Put("/put")

	if putResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, putResp.Code)
	}

	token1 := extractTokenFromCookie(putResp.Header().Get("Set-Cookie"))
	if token1 == "" {
		t.Fatal("No session token found in PUT response")
	}

	getResp := api.Get("/get", "Cookie: session="+token1)

	if getResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, getResp.Code)
	}

	responseBody := strings.Trim(getResp.Body.String(), "\"\n\r\t ")

	if responseBody != "bar" {
		t.Errorf("want value %q; got %q", "bar", responseBody)
	}

	time.Sleep(600 * time.Millisecond)

	getResp = api.Get("/get", "Cookie: session="+token1)

	if getResp.Code != http.StatusInternalServerError {
		t.Errorf("want status %d; got %d", http.StatusInternalServerError, getResp.Code)
	}

	var errorResponse struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	err := json.Unmarshal(getResp.Body.Bytes(), &errorResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errorResponse.Detail != "foo does not exist in session" {
		t.Errorf("want value %q; got %q", "foo does not exist in session", errorResponse.Detail)
	}
}

func TestIdleTimeout(t *testing.T) {
	t.Parallel()

	sessionManager := New()
	sessionManager.IdleTimeout = 200 * time.Millisecond

	_, api := humatest.New(t)
	addRoutes(api, sessionManager)

	putResp := api.Put("/put")
	if putResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, putResp.Code)
	}
	token := extractTokenFromCookie(putResp.Header().Get("Set-Cookie"))

	time.Sleep(100 * time.Millisecond)

	// First GET request
	getResp1 := api.Get("/get", "Cookie: session="+token)
	if getResp1.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, getResp1.Code)
	}

	time.Sleep(150 * time.Millisecond)

	// Second GET request
	getResp2 := api.Get("/get", "Cookie: session="+token)
	if getResp2.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, getResp2.Code)
	}

	time.Sleep(200 * time.Millisecond)

	// Third GET request
	getResp3 := api.Get("/get", "Cookie: session="+token)
	if getResp3.Code != http.StatusInternalServerError {
		t.Errorf("want status %d; got %d", http.StatusInternalServerError, getResp3.Code)
	}

	var errorResponse struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	err := json.Unmarshal(getResp3.Body.Bytes(), &errorResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errorResponse.Detail != "foo does not exist in session" {
		t.Errorf("want value %q; got %q", "foo does not exist in session", errorResponse.Detail)
	}
}

// func TestIdleTimeout(t *testing.T) {
// 	t.Parallel()

// 	sessionManager := New()
// 	sessionManager.IdleTimeout = 200 * time.Millisecond
// 	sessionManager.Lifetime = time.Second

// 	mux := http.NewServeMux()
// 	mux.HandleFunc("/put", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		sessionManager.Put(r.Context(), "foo", "bar")
// 	}))
// 	mux.HandleFunc("/get", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		v := sessionManager.Get(r.Context(), "foo")
// 		if v == nil {
// 			http.Error(w, "foo does not exist in session", 500)
// 			return
// 		}
// 		w.Write([]byte(v.(string)))
// 	}))

// 	ts := newTestServer(t, sessionManager.LoadAndSave(mux))
// 	defer ts.Close()

// 	ts.execute(t, "/put")

// 	time.Sleep(100 * time.Millisecond)
// 	ts.execute(t, "/get")

// 	time.Sleep(150 * time.Millisecond)
// 	_, body := ts.execute(t, "/get")
// 	if body != "bar" {
// 		t.Errorf("want %q; got %q", "bar", body)
// 	}

// 	time.Sleep(200 * time.Millisecond)
// 	_, body = ts.execute(t, "/get")
// 	if body != "foo does not exist in session\n" {
// 		t.Errorf("want %q; got %q", "foo does not exist in session\n", body)
// 	}
// }

// func TestDestroy(t *testing.T) {
// 	t.Parallel()

// 	sessionManager := New()

// 	mux := http.NewServeMux()
// 	mux.HandleFunc("/put", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		sessionManager.Put(r.Context(), "foo", "bar")
// 	}))
// 	mux.HandleFunc("/destroy", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		err := sessionManager.Destroy(r.Context())
// 		if err != nil {
// 			http.Error(w, err.Error(), 500)
// 			return
// 		}
// 	}))
// 	mux.HandleFunc("/get", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		v := sessionManager.Get(r.Context(), "foo")
// 		if v == nil {
// 			http.Error(w, "foo does not exist in session", 500)
// 			return
// 		}
// 		w.Write([]byte(v.(string)))
// 	}))

// 	ts := newTestServer(t, sessionManager.LoadAndSave(mux))
// 	defer ts.Close()

// 	ts.execute(t, "/put")
// 	header, _ := ts.execute(t, "/destroy")
// 	cookie := header.Get("Set-Cookie")

// 	if strings.HasPrefix(cookie, fmt.Sprintf("%s=;", sessionManager.Cookie.Name)) == false {
// 		t.Fatalf("got %q: expected prefix %q", cookie, fmt.Sprintf("%s=;", sessionManager.Cookie.Name))
// 	}
// 	if strings.Contains(cookie, "Expires=Thu, 01 Jan 1970 00:00:01 GMT") == false {
// 		t.Fatalf("got %q: expected to contain %q", cookie, "Expires=Thu, 01 Jan 1970 00:00:01 GMT")
// 	}
// 	if strings.Contains(cookie, "Max-Age=0") == false {
// 		t.Fatalf("got %q: expected to contain %q", cookie, "Max-Age=0")
// 	}

// 	_, body := ts.execute(t, "/get")
// 	if body != "foo does not exist in session\n" {
// 		t.Errorf("want %q; got %q", "foo does not exist in session\n", body)
// 	}
// }

// func TestRenewToken(t *testing.T) {
// 	t.Parallel()

// 	sessionManager := New()

// 	mux := http.NewServeMux()
// 	mux.HandleFunc("/put", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		sessionManager.Put(r.Context(), "foo", "bar")
// 	}))
// 	mux.HandleFunc("/renew", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		err := sessionManager.RenewToken(r.Context())
// 		if err != nil {
// 			http.Error(w, err.Error(), 500)
// 			return
// 		}
// 	}))
// 	mux.HandleFunc("/get", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		v := sessionManager.Get(r.Context(), "foo")
// 		if v == nil {
// 			http.Error(w, "foo does not exist in session", 500)
// 			return
// 		}
// 		w.Write([]byte(v.(string)))
// 	}))

// 	ts := newTestServer(t, sessionManager.LoadAndSave(mux))
// 	defer ts.Close()

// 	header, _ := ts.execute(t, "/put")
// 	cookie := header.Get("Set-Cookie")
// 	originalToken := extractTokenFromCookie(cookie)

// 	header, _ = ts.execute(t, "/renew")
// 	cookie = header.Get("Set-Cookie")
// 	newToken := extractTokenFromCookie(cookie)

// 	if newToken == originalToken {
// 		t.Fatal("token has not changed")
// 	}

// 	_, body := ts.execute(t, "/get")
// 	if body != "bar" {
// 		t.Errorf("want %q; got %q", "bar", body)
// 	}
// }

// func TestRememberMe(t *testing.T) {
// 	t.Parallel()

// 	sessionManager := New()
// 	sessionManager.Cookie.Persist = false

// 	mux := http.NewServeMux()
// 	mux.HandleFunc("/put-normal", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		sessionManager.Put(r.Context(), "foo", "bar")
// 	}))
// 	mux.HandleFunc("/put-rememberMe-true", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		sessionManager.RememberMe(r.Context(), true)
// 		sessionManager.Put(r.Context(), "foo", "bar")
// 	}))
// 	mux.HandleFunc("/put-rememberMe-false", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		sessionManager.RememberMe(r.Context(), false)
// 		sessionManager.Put(r.Context(), "foo", "bar")
// 	}))

// 	ts := newTestServer(t, sessionManager.LoadAndSave(mux))
// 	defer ts.Close()

// 	header, _ := ts.execute(t, "/put-normal")
// 	header.Get("Set-Cookie")

// 	if strings.Contains(header.Get("Set-Cookie"), "Max-Age=") || strings.Contains(header.Get("Set-Cookie"), "Expires=") {
// 		t.Errorf("want no Max-Age or Expires attributes; got %q", header.Get("Set-Cookie"))
// 	}

// 	header, _ = ts.execute(t, "/put-rememberMe-true")
// 	header.Get("Set-Cookie")

// 	if !strings.Contains(header.Get("Set-Cookie"), "Max-Age=") || !strings.Contains(header.Get("Set-Cookie"), "Expires=") {
// 		t.Errorf("want Max-Age and Expires attributes; got %q", header.Get("Set-Cookie"))
// 	}

// 	header, _ = ts.execute(t, "/put-rememberMe-false")
// 	header.Get("Set-Cookie")

// 	if strings.Contains(header.Get("Set-Cookie"), "Max-Age=") || strings.Contains(header.Get("Set-Cookie"), "Expires=") {
// 		t.Errorf("want no Max-Age or Expires attributes; got %q", header.Get("Set-Cookie"))
// 	}
// }

// func TestIterate(t *testing.T) {
// 	t.Parallel()

// 	sessionManager := New()

// 	mux := http.NewServeMux()
// 	mux.HandleFunc("/put", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		sessionManager.Put(r.Context(), "foo", r.URL.Query().Get("foo"))
// 	}))

// 	for i := 0; i < 3; i++ {
// 		ts := newTestServer(t, sessionManager.LoadAndSave(mux))
// 		defer ts.Close()

// 		ts.execute(t, "/put?foo="+strconv.Itoa(i))
// 	}

// 	results := []string{}

// 	err := sessionManager.Iterate(context.Background(), func(ctx context.Context) error {
// 		i := sessionManager.GetString(ctx, "foo")
// 		results = append(results, i)
// 		return nil
// 	})

// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	sort.Strings(results)

// 	if !reflect.DeepEqual(results, []string{"0", "1", "2"}) {
// 		t.Fatalf("unexpected value: got %v", results)
// 	}

// 	err = sessionManager.Iterate(context.Background(), func(ctx context.Context) error {
// 		return errors.New("expected error")
// 	})
// 	if err.Error() != "expected error" {
// 		t.Fatal("didn't get expected error")
// 	}
// }
