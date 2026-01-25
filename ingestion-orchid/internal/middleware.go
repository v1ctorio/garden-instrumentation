package internal

import "net/http"

func ApiKeyAuth(validKeys map[string]struct{}) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				http.Error(w, "missing API key", http.StatusUnauthorized)
				return
			}

			if _, present := validKeys[key]; !present {
				http.Error(w, "invalid API key", http.StatusUnauthorized)
				return
			}

			h.ServeHTTP(w, r)

		})
	}
}
