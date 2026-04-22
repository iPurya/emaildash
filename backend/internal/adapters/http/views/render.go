package views

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
)

func Render(ctx context.Context, w http.ResponseWriter, component templ.Component) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = component.Render(ctx, w)
	})
}
