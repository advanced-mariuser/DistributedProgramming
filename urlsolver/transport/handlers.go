package transport

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"urlsolver/mapping"
)

type Handler struct {
	urlMapping      map[string]string
	fallbackUrl     string
	mappingFilePath string
	port            string
}

type PageData struct {
	Port       string
	UrlMapping map[string]string
}

func Router(urlMapping map[string]string, fallbackUrl, mappingFilePath string, port string) http.Handler {
	handler := &Handler{
		urlMapping:      urlMapping,
		fallbackUrl:     fallbackUrl,
		mappingFilePath: mappingFilePath,
		port:            port,
	}

	r := mux.NewRouter()
	r.HandleFunc("/shorten", handler.shortenPageHandler).Methods(http.MethodGet, http.MethodPost)
	r.HandleFunc("/{shortLink}", handler.redirectHandler).Methods(http.MethodGet)

	return logMiddleware(r)
}

func (h *Handler) redirectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortLink := vars["shortLink"]
	lookupKey := "/" + shortLink

	longUrl, ok := h.urlMapping[lookupKey]

	if ok {
		log.WithFields(log.Fields{"short": lookupKey, "long": longUrl}).Info("Redirecting")
		http.Redirect(w, r, longUrl, http.StatusSeeOther)
		return
	}

	log.WithField("short", lookupKey).Warn("Short link not found, redirecting to fallback")
	http.Redirect(w, r, h.fallbackUrl, http.StatusSeeOther)
}

func (h *Handler) shortenPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		short := r.FormValue("short")
		long := r.FormValue("long")

		if short != "" && long != "" {
			if short[0] == '/' {
				short = short[1:]
			}
			storeKey := "/" + short

			h.urlMapping[storeKey] = long
			err := mapping.SaveMapping(h.mappingFilePath, h.urlMapping)

			if err != nil {
				log.WithError(err).Error("Failed to save mapping")
				http.Error(w, "Failed to save URL", http.StatusInternalServerError)
				return
			}
			log.WithFields(log.Fields{"short": storeKey, "long": long}).Info("Added new short link")
		}
		http.Redirect(w, r, "/shorten", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles("templates/form.html")
	if err != nil {
		log.WithError(err).Error("Could not parse template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	mappingCopy := make(map[string]string)
	for k, v := range h.urlMapping {
		mappingCopy[k] = v
	}

	data := PageData{
		Port:       h.port,
		UrlMapping: mappingCopy,
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, data)
}

func logMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"method":     r.Method,
			"url":        r.URL,
			"remoteAddr": r.RemoteAddr,
			"userAgent":  r.UserAgent(),
		}).Info("got a new request")
		h.ServeHTTP(w, r)
	})
}
