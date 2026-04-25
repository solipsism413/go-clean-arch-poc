package graphql

import (
	"context"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/auth"
)

// NewHandler creates a new GraphQL HTTP handler.
func NewHandler(resolver *Resolver, authService input.AuthService) http.Handler {
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolver}))

	// Add GET support for simple queries and subscriptions
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	// Wrap with optional auth middleware
	return withOptionalAuth(authService, srv)
}

// NewPlaygroundHandler creates a GraphQL Playground handler.
func NewPlaygroundHandler(endpoint string) http.Handler {
	return playground.Handler("GraphQL Playground", endpoint)
}

// withOptionalAuth wraps the handler with optional JWT authentication.
// If a valid token is provided, claims are added to context.
// If no token is provided, the request proceeds unauthenticated.
func withOptionalAuth(authService input.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			next.ServeHTTP(w, r)
			return
		}

		claims, err := authService.ValidateToken(r.Context(), parts[1])
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
