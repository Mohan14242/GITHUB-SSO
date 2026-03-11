package auth

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type contextKey string

const claimsKey contextKey = "claims"

func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

/* ─────────────────────────────────────────
   Authenticate — validates Bearer JWT
   Also accepts ?token= query param for SSE
───────────────────────────────────────── */

func Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[MIDDLEWARE][AUTHENTICATE] %s %s", r.Method, r.URL.Path)

		token := extractBearerToken(r)
		if token == "" {
			log.Printf("[MIDDLEWARE][AUTHENTICATE][DENY] No Bearer token → %s %s",
				r.Method, r.URL.Path)
			http.Error(w, `{"error":"missing authorization token"}`, http.StatusUnauthorized)
			return
		}

		log.Printf("[MIDDLEWARE][AUTHENTICATE] Token present (length=%d), validating…", len(token))

		claims, err := ValidateJWT(token)
		if err != nil {
			log.Printf("[MIDDLEWARE][AUTHENTICATE][DENY] Invalid/expired token → path=%s err=%v",
				r.URL.Path, err)
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		log.Printf("[MIDDLEWARE][AUTHENTICATE][PASS] login=%s role=%s → %s %s",
			claims.GithubLogin, claims.Role, r.Method, r.URL.Path)

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

/* ─────────────────────────────────────────
   RequireRole — enforces minimum role level
───────────────────────────────────────── */

func RequireRole(minRole string, next http.HandlerFunc) http.HandlerFunc {
	return Authenticate(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())

		userPriority    := rolePriority[claims.Role]
		requiredPriority := rolePriority[minRole]

		log.Printf("[MIDDLEWARE][RBAC] login=%s role=%s(priority=%d) required=%s(priority=%d) path=%s",
			claims.GithubLogin,
			claims.Role, userPriority,
			minRole, requiredPriority,
			r.URL.Path,
		)

		if !hasPermission(claims.Role, minRole) {
			log.Printf("[MIDDLEWARE][RBAC][DENY] login=%s role=%s insufficient for minRole=%s → %s %s",
				claims.GithubLogin, claims.Role, minRole, r.Method, r.URL.Path)
			http.Error(w,
				`{"error":"forbidden: insufficient permissions"}`,
				http.StatusForbidden,
			)
			return
		}

		log.Printf("[MIDDLEWARE][RBAC][PASS] login=%s role=%s cleared minRole=%s → %s %s",
			claims.GithubLogin, claims.Role, minRole, r.Method, r.URL.Path)

		next(w, r)
	})
}

/* ─────────────────────────────────────────
   RequirePipelineKey — for CI/CD callbacks
───────────────────────────────────────── */

func RequirePipelineKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[MIDDLEWARE][PIPELINE_KEY] %s %s", r.Method, r.URL.Path)

		expected := os.Getenv("PIPELINE_API_KEY")
		if expected == "" {
			log.Println("[MIDDLEWARE][PIPELINE_KEY][ERROR] PIPELINE_API_KEY env var not set — all pipeline requests will be rejected")
			http.Error(w, `{"error":"pipeline authentication not configured"}`, http.StatusInternalServerError)
			return
		}

		provided := r.Header.Get("X-API-Key")
		if provided == "" {
			log.Printf("[MIDDLEWARE][PIPELINE_KEY][DENY] X-API-Key header missing → %s %s",
				r.Method, r.URL.Path)
			http.Error(w, `{"error":"missing X-API-Key header"}`, http.StatusUnauthorized)
			return
		}

		if provided != expected {
			preview := provided
			if len(provided) > 4 {
				preview = provided[:4] + "…"
			}
			log.Printf("[MIDDLEWARE][PIPELINE_KEY][DENY] Invalid API key (provided prefix=%s) → %s %s",
				preview, r.Method, r.URL.Path)
			http.Error(w, `{"error":"invalid pipeline api key"}`, http.StatusUnauthorized)
			return
		}

		log.Printf("[MIDDLEWARE][PIPELINE_KEY][PASS] Valid pipeline key → %s %s",
			r.Method, r.URL.Path)
		next(w, r)
	}
}

/* ─────────────────────────────────────────
   WithCORS
───────────────────────────────────────── */

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		frontendURL := os.Getenv("FRONTEND_URL")
		origin      := r.Header.Get("Origin")

		log.Printf("[MIDDLEWARE][CORS] %s %s origin=%s", r.Method, r.URL.Path, origin)

		if frontendURL == "" {
			log.Println("[MIDDLEWARE][CORS][WARN] FRONTEND_URL not set — CORS header will be empty")
		}

		w.Header().Set("Access-Control-Allow-Origin",  frontendURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")

		// Required for SSE — disables nginx/proxy buffering per-request
		if isSSERequest(r) {
			w.Header().Set("X-Accel-Buffering", "no")
			log.Printf("[MIDDLEWARE][CORS] SSE request detected, buffering disabled → %s", r.URL.Path)
		}

		if r.Method == http.MethodOptions {
			log.Printf("[MIDDLEWARE][CORS] Preflight handled → %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[MIDDLEWARE][CORS] Request complete → %s %s elapsed=%s",
			r.Method, r.URL.Path, time.Since(start))
	})
}

/* ─────────────────────────────────────────
   Helpers
───────────────────────────────────────── */

func hasPermission(userRole, requiredRole string) bool {
	return rolePriority[userRole] >= rolePriority[requiredRole]
}

// extractBearerToken reads JWT from:
//  1. Authorization: Bearer <token>  header  — standard for all API calls
//  2. ?token=<token>                 query   — fallback for SSE (EventSource
//     does not support custom headers in browsers)
func extractBearerToken(r *http.Request) string {
	// ── Primary: Authorization header ──
	header := r.Header.Get("Authorization")
	if header != "" {
		if !strings.HasPrefix(header, "Bearer ") {
			log.Printf("[MIDDLEWARE][TOKEN] Authorization header present but not Bearer format → %s %s",
				r.Method, r.URL.Path)
			return ""
		}
		return strings.TrimPrefix(header, "Bearer ")
	}

	// ── Fallback: ?token= query param (SSE connections only) ──
	// EventSource API in browsers cannot set custom headers,
	// so the frontend appends the JWT as a query parameter for /stream endpoints
	if queryToken := r.URL.Query().Get("token"); queryToken != "" {
		// Only allow query-param auth for SSE stream endpoints
		if isSSERequest(r) {
			log.Printf("[MIDDLEWARE][TOKEN] Using ?token= query param for SSE → %s %s",
				r.Method, r.URL.Path)
			return queryToken
		}
		log.Printf("[MIDDLEWARE][TOKEN][DENY] ?token= query param only allowed for SSE endpoints → %s %s",
			r.Method, r.URL.Path)
		return ""
	}

	log.Printf("[MIDDLEWARE][TOKEN] Authorization header absent and no ?token= → %s %s",
		r.Method, r.URL.Path)
	return ""
}

// isSSERequest checks if the request is an SSE stream connection
// by looking at the Accept header and URL path suffix
func isSSERequest(r *http.Request) bool {
	acceptsEventStream := strings.Contains(r.Header.Get("Accept"), "text/event-stream")
	pathIsStream       := strings.HasSuffix(r.URL.Path, "/stream")
	return acceptsEventStream || pathIsStream
}
