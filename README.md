**Step by step guide:**

Im writing this for myself, it is not AI generated

1. Before anything run: $ git mod init <github path to the repo>
In this case $ go mod init github.com/W7SP/Job-Processor-API
It is important for the path to be the same as in the repo for later purposes.


2. Create a main.go file with some printing.
Then run $ go build and <the .exe file> to see that everything is working properly


3. Make sure to add .idea or vscode to the git ignore so they are not commited

4. Create the skeleton
   Job-Processor-API/
   <br>├── cmd/
   <br>│   └── api/
   <br>│       └── main.go
   <br>├── internal/
   <br>│   ├── handler/
   <br>│   ├── service/
   <br>│   └── repository/
   <br>├── go.mod
   <br>└── .gitignore

5. Explanation:
   1. **cmd/api/main.go** — the entry point.
      Named cmd/ because a project can have multiple binaries (e.g. later you might add cmd/worker/ for a standalone background processor). main.go should stay thin — just wiring things together, no real logic.

   2. **internal/** — this isn't just a naming convention, it's enforced by the Go compiler. Any package under a folder called internal/ can only be imported by code inside the same module. If you ever published this as a library, nobody outside your repo could import internal/service even if they tried. It's Go's way of saying "this is implementation detail, not public API."

   3. **handler / service / repository** — (HTTP layer → business logic → data access), each able to depend on interfaces from the layer below.

6. Create a minimal server with a health check endpoint (this is just to check if everything is OK)
7. Switch the router to chi.
<br>Why do this? 
<br>Why not continue with mux := http.NewServeMux() ???
<BR>**EXPLANATION:** 
<br>The previous versions of http.NewServeMux fires for any method — GET, POST, DELETE, all the same. You'd have to manually check r.Method
<br>Something like /jobs/{id} — pulling id out of the URL — isn't supported at all. You'd have to manually parse r.URL.Path yourself with string splitting
<BR>As of 2024 this has been fixed, BUT chi gives you:
<br>A clean, standard way to wrap requests through shared logic (logging, auth checks, panic recovery, request timeouts) without writing that plumbing yourself. Stdlib has no built-in convention for this
<br>Route grouping / mounting sub-routers — cleanly saying "everything under /api/v1/jobs goes through this group of middleware and these routes," which matters once your API has more than a handful of endpoints.
Alright now first we need to run: **$ go get github.com/go-chi/chi/v5**
<br> This will download the required files and update go.mod (the requirements file).
<br>It also updates the go.sum file.
<br>go.sum contains cryptographic checksums of every dependency's exact code, at the exact version you're using. Its job is tamper/corruption detection: next time anyone builds this project (you on another machine, a CI pipeline, a teammate), Go re-downloads chi and verifies its checksum still matches what's in go.sum. If it doesn't match — corrupted download, or worse, a compromised package registry serving different code — the build fails loudly instead of silently using different code than you tested with
Switch the router from the above to chi and now run **$ go mod tidy**
<br>it scans your code, sees you're importing github.com/go-chi/chi/v5/middleware (a sub-package of chi you didn't explicitly go get), confirms everything lines up correctly in go.mod/go.sum, and would warn/fix things if anything were missing or unused
<br> Pay attention to the middle wares
<br> We basically only changed the router to chi here but we also added some middlewares.
<br> Same as in settings.py in Django, there is a list of middlewares stacked in a specific order.
<br> Every middleware is executed in order before every request.
<br> The logging middleware is pretty self explanatory, it saves us from putting logging separately in every view
<br> The recover one is more interesting.
<br> if a panic happens inside a handler and nothing catches it, it can crash your entire server process, killing every in-flight request, not just the one that panicked. Recoverer is middleware that catches panics per-request, turns them into a 500 Internal Server Error response, and logs the panic — so one bad request degrades gracefully instead of taking your whole service down