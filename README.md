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
   ├── cmd/
   │   └── api/
   │       └── main.go
   ├── internal/
   │   ├── handler/
   │   ├── service/
   │   └── repository/
   ├── go.mod
   └── .gitignore

5. Explanation:
   1. **cmd/api/main.go** — the entry point.
      Named cmd/ because a project can have multiple binaries (e.g. later you might add cmd/worker/ for a standalone background processor). main.go should stay thin — just wiring things together, no real logic.

   2. **internal/** — this isn't just a naming convention, it's enforced by the Go compiler. Any package under a folder called internal/ can only be imported by code inside the same module. If you ever published this as a library, nobody outside your repo could import internal/service even if they tried. It's Go's way of saying "this is implementation detail, not public API."

   3. **handler / service / repository** — (HTTP layer → business logic → data access), each able to depend on interfaces from the layer below.

6. Create a minimal server with a health check endpoint