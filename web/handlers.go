package web

import (
	"os"

	"github.com/rohanthewiz/rweb"
)

func rootHandler(ctx rweb.Context) error {
	return ctx.WriteJSON(map[string]interface{}{
		"response": "OK",
		"ENV":      os.Getenv("ENV"),
	})
}
