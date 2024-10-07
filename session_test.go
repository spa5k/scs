package scs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

	huma.Register(api, huma.Operation{
		Method: http.MethodDelete,
		Path:   "/delete",
	}, func(ctx context.Context, input *struct {
		Cookie string `header:"Cookie"`
	}) (*struct {
		Status int `json:"status"`
	}, error,
	) {
		sessionManager.Destroy(ctx)
		return &struct {
			Status int `json:"status"`
		}{Status: http.StatusOK}, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPost,
		Path:   "/renew",
	}, func(ctx context.Context, input *struct {
		Cookie string `header:"Cookie"`
	}) (*struct {
		Status int `json:"status"`
	}, error,
	) {
		sessionManager.RenewToken(ctx)
		return &struct {
			Status int `json:"status"`
		}{Status: http.StatusOK}, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPut,
		Path:   "/put-normal",
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
		Method: http.MethodPut,
		Path:   "/put-rememberMe-true",
	}, func(ctx context.Context, input *struct {
		Cookie string `header:"Cookie"`
	}) (*struct {
		Status int `json:"status"`
	}, error,
	) {
		sessionManager.RememberMe(ctx, true)
		sessionManager.Put(ctx, "foo", "bar")
		return &struct {
			Status int `json:"status"`
		}{Status: http.StatusOK}, nil
	})

	huma.Register(api, huma.Operation{
		Method: http.MethodPut,
		Path:   "/put-rememberMe-false",
	}, func(ctx context.Context, input *struct {
		Cookie string `header:"Cookie"`
	}) (*struct {
		Status int `json:"status"`
	}, error,
	) {
		sessionManager.RememberMe(ctx, false)
		sessionManager.Put(ctx, "foo", "bar")
		return &struct {
			Status int `json:"status"`
		}{Status: http.StatusOK}, nil
	})
}

func TestEnable(t *testing.T) {
	t.Parallel()

	sessionManager := New()

	_, api := humatest.New(t)

	addRoutes(api, sessionManager)

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
	addRoutes(api, sessionManager)

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

func TestDestroy(t *testing.T) {
	t.Parallel()

	sessionManager := New()

	_, api := humatest.New(t)

	// Add routes
	addRoutes(api, sessionManager)

	// Initial PUT request
	putResp := api.Put("/put")
	if putResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, putResp.Code)
	}
	token := extractTokenFromCookie(putResp.Header().Get("Set-Cookie"))

	// Destroy session
	destroyResp := api.Delete("/delete", "Cookie: session="+token)
	if destroyResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, destroyResp.Code)
	}

	cookie := destroyResp.Header().Get("Set-Cookie")
	if !strings.HasPrefix(cookie, fmt.Sprintf("%s=;", sessionManager.Cookie.Name)) {
		t.Fatalf("got %q: expected prefix %q", cookie, fmt.Sprintf("%s=;", sessionManager.Cookie.Name))
	}
	if !strings.Contains(cookie, "Expires=Thu, 01 Jan 1970 00:00:01 GMT") {
		t.Fatalf("got %q: expected to contain %q", cookie, "Expires=Thu, 01 Jan 1970 00:00:01 GMT")
	}
	if !strings.Contains(cookie, "Max-Age=0") {
		t.Fatalf("got %q: expected to contain %q", cookie, "Max-Age=0")
	}

	// Try to get the destroyed session
	getResp := api.Get("/get", "Cookie: session="+token)
	if getResp.Code != http.StatusInternalServerError {
		t.Errorf("want status %d; got %d", http.StatusInternalServerError, getResp.Code)
	}

	var errorResponse struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(getResp.Body.Bytes(), &errorResponse); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	expectedErrorDetail := "foo does not exist in session"
	if errorResponse.Detail != expectedErrorDetail {
		t.Errorf("want error detail %q; got %q", expectedErrorDetail, errorResponse.Detail)
	}
}

func TestRenewToken(t *testing.T) {
	t.Parallel()

	sessionManager := New()

	_, api := humatest.New(t)

	// Add routes
	addRoutes(api, sessionManager)

	// Initial PUT request
	putResp := api.Put("/put")
	if putResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, putResp.Code)
	}
	originalToken := extractTokenFromCookie(putResp.Header().Get("Set-Cookie"))

	// Renew token
	renewResp := api.Post("/renew", "Cookie: session="+originalToken)
	if renewResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, renewResp.Code)
	}
	newToken := extractTokenFromCookie(renewResp.Header().Get("Set-Cookie"))

	if newToken == originalToken {
		t.Fatal("token has not changed")
	}

	// Get session data with new token
	getResp := api.Get("/get", "Cookie: session="+newToken)
	if getResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, getResp.Code)
	}

	var responseBody string
	if err := json.Unmarshal(getResp.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}

	if responseBody != "bar" {
		t.Errorf("want %q; got %q", "bar", responseBody)
	}
}

func TestRememberMe(t *testing.T) {
	t.Parallel()

	sessionManager := New()
	sessionManager.Cookie.Persist = false

	_, api := humatest.New(t)

	// Add session middleware
	api.UseMiddleware(sessionManager.LoadAndSave)

	// Add routes
	addRoutes(api, sessionManager)

	// Test normal put
	putNormalResp := api.Put("/put-normal")
	if putNormalResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, putNormalResp.Code)
	}
	normalCookie := putNormalResp.Header().Get("Set-Cookie")
	if strings.Contains(normalCookie, "Max-Age=") || strings.Contains(normalCookie, "Expires=") {
		t.Errorf("want no Max-Age or Expires attributes; got %q", normalCookie)
	}

	// Test put with RememberMe true
	putRememberTrueResp := api.Put("/put-rememberMe-true")
	if putRememberTrueResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, putRememberTrueResp.Code)
	}
	rememberTrueCookie := putRememberTrueResp.Header().Get("Set-Cookie")
	if !strings.Contains(rememberTrueCookie, "Max-Age=") || !strings.Contains(rememberTrueCookie, "Expires=") {
		t.Errorf("want Max-Age and Expires attributes; got %q", rememberTrueCookie)
	}

	// Test put with RememberMe false
	putRememberFalseResp := api.Put("/put-rememberMe-false")
	if putRememberFalseResp.Code != http.StatusOK {
		t.Errorf("want status %d; got %d", http.StatusOK, putRememberFalseResp.Code)
	}
	rememberFalseCookie := putRememberFalseResp.Header().Get("Set-Cookie")
	if strings.Contains(rememberFalseCookie, "Max-Age=") || strings.Contains(rememberFalseCookie, "Expires=") {
		t.Errorf("want no Max-Age or Expires attributes; got %q", rememberFalseCookie)
	}
}

func TestIterate(t *testing.T) {
	t.Parallel()

	sessionManager := New()

	_, api := humatest.New(t)

	// Add session middleware
	api.UseMiddleware(sessionManager.LoadAndSave)
	ctx := context.Background()
	// Add routes
	addRoutes(api, sessionManager)

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		putResp := api.PutCtx(ctx, "/put?foo="+strconv.Itoa(i))
		if putResp.Code != http.StatusOK {
			t.Errorf("want status %d; got %d", http.StatusOK, putResp.Code)
		}
	}
}
