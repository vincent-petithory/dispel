package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
)

func main() {
	addr := flag.String("addr", ":0", "address to bind to")
	flag.Parse()

	host, port, err := net.SplitHostPort(*addr)
	if err != nil {
		log.Fatal(err)
	}
	if host == "" {
		host = "localhost"
	}

	baseURL, err := url.Parse("http://" + net.JoinHostPort(host, port))

	app := &App{
		Router: &GorillaRouter{
			Router:  mux.NewRouter(),
			BaseURL: baseURL,
		},
	}
	// register routes and handlers using dispel generated funcs
	jsonCodec := &JSONCodec{}
	registerRoutes(app.Router)
	registerHandlers(app.Router, app.Router, app, jsonCodec, jsonCodec, app.appHandler)

	go func() {
		http.ListenAndServe(*addr, app)
	}()
	sigc := make(chan os.Signal)
	signal.Notify(sigc, syscall.SIGTERM)
	<-sigc
}

type App struct {
	Router *GorillaRouter
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.Router.ServeHTTP(w, r)
}

func (app *App) appHandler(f errorHTTPHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, err := f(w, r)
		if err != nil {
			log.Printf("HTTP Status %d: %v", status, err)
			http.Error(w, err.Error(), status)
		}
	})
}
