package microfrontends

import (
	"fmt"
	"github.com/hasangenc0/microfrontends/pkg/client"
	"github.com/hasangenc0/microfrontends/pkg/collector"
	"html/template"
	"io/ioutil"
	"net/http"
	"runtime"
	"sync"
)

type Gateway = collector.Gateway
type Page = collector.Page
type App = collector.App

const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

func getMethod(method string) string {
	switch method {
	case MethodGet: return MethodGet
	case MethodHead: return MethodHead
	case MethodPost: return MethodPost
	case MethodPut: return MethodPut
	case MethodPatch: return MethodPatch
	case MethodDelete: return MethodDelete
	case MethodConnect: return MethodConnect
	case MethodOptions: return MethodOptions
	case MethodTrace: return MethodTrace
	default:
		panic(method + " is not a type of http method.")
	}
}

func getUrl(host string, port string) string {
	return host + ":" + port
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Transfer-Encoding", "chunked")
	//w.Header().Set("X-Content-Type-Options", "nosniff")
}

func initialize(w http.ResponseWriter, page Page) {
	flusher, ok := w.(http.Flusher)

	if !ok {
		panic("expected http.ResponseWriter to be an http.Flusher")
	}

	tmpl, err := template.New(page.Name).Parse(page.Content)

	if err != nil {
		panic("An Error occured when parsing html")
		return
	}

	err = tmpl.Execute(w, "")

	if err != nil {
		panic("Error in Template.Execute")
	}

	flusher.Flush()
}

func sendChunk(w http.ResponseWriter, gateway Gateway, wg *sync.WaitGroup, ch chan http.Flusher) {
	var flusher, ok = w.(http.Flusher)
	if !ok {
		panic("expected http.ResponseWriter to be an http.Flusher")
	}

	_client := &http.Client{}
	req, err := http.NewRequest(getMethod(gateway.Method), getUrl(gateway.Host, gateway.Port), nil)
	if err != nil {
		panic(err)
	}
	resp, err := _client.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		bodyString := string(bodyBytes)

		chunk := client.GetView(gateway.Name, bodyString)

		fmt.Fprintf(w, chunk)
	}

	ch <- flusher
	wg.Done()
}

func finish(w http.ResponseWriter) {
	flusher, ok := w.(http.Flusher)

	if !ok {
		panic("expected http.ResponseWriter to be an http.Flusher")
	}

	_, err := w.Write([]byte(""))

	if err != nil {
		panic("expected http.ResponseWriter to be an http.Flusher")

	}

	flusher.Flush()

}

func Make(w http.ResponseWriter, app App) {
	setHeaders(w)

	var wg sync.WaitGroup

	initialize(w, app.Page)

	runtime.GOMAXPROCS(4)

	var flusher = make(chan http.Flusher)

	for _, gateway := range app.Gateway {
		wg.Add(1)
		go sendChunk(w, gateway, &wg, flusher)
	}

	for range app.Gateway {
		flusher, ok := <-flusher
		if !ok {
			panic("expected http.ResponseWriter to be an http.Flusher")
		}
		flusher.Flush()
	}

	wg.Wait()

	finish(w)
}
