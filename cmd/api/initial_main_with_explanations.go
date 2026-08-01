package main

import (
	"log"
	"net/http"
)

func mainForExplanations() {
	mux := http.NewServeMux()                                   // The url.py router like in Django
	mux.HandleFunc("/healthz", healthHandlerForExplanationOnly) // path() basically on /healthz call healthHandler

	log.Println("starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		/*
			http.ListenAndServe(":8080", mux) -> starts the HTTP server ON 8080
			This is a blocking call: the program sits here running forever, handling requests
			It only returns if there is an error.
		*/

		log.Fatal(err) // logs the error
	}
}

// Basically a Django view
func healthHandlerForExplanationOnly(w http.ResponseWriter, r *http.Request) {
	/*
		In django you build a response object and return it
		def health_handler(request):
			return HttpResponse("ok", status=200)

		In Go, there's no return value.
		Instead, you get handed w, which is a live, writable connection to the client,
		and you build the response by calling methods on it, imperatively, in order:

		w http.ResponseWriter -> a file you're writing output into, live, as the function runs.
		That's why order matters here.

		r *http.Request -> Same as a Django request.
		It contains method, headers, URL, body, query params, etc

		w.WriteHeader(http.StatusOK) — sets the HTTP status code (200).
		This must happen before you write any body content.
		In the actual HTTP protocol, status/headers are sent before the body.
		Go's API mirrors that ordering constraint directly.

		w.Write([]byte("ok")) — writes the response body.
		It takes []byte (a byte slice), not a plain string — you have to explicitly convert "ok" into bytes.
		This is because Write comes from a general-purpose Go interface (io.Writer) used everywhere.
		It is for writing raw data — files, network connections, buffers — not just HTTP text responses,
		So it deals in bytes rather than assuming everything is a string. In Django, HttpResponse("ok") accepts a string.
	*/

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
