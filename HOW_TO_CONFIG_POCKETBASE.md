Now I have a complete picture. Let me put together a clear explanation.

Here's what's happening under the hood and the correct approaches for each setting.

---

## How PocketBase Configuration Actually Works

There are **two separate configuration layers**, which is the root of the confusion:

### Layer 1: App-level config (`pocketbase.Config`)

Data directory, encryption, dev mode, query timeout, and DB pool settings are controlled via `pocketbase.NewWithConfig()`. This is the **official, clean API** for those settings.

```47:63:pocketbase.go
// Config is the PocketBase initialization config struct.
type Config struct {
	// hide the default console server info on app startup
	HideStartBanner bool

	// optional default values for the console flags
	DefaultDev           bool
	DefaultDataDir       string // if not set, it will fallback to "./pb_data"
	DefaultEncryptionEnv string
	DefaultQueryTimeout  time.Duration // default to core.DefaultQueryTimeout (in seconds)

	// optional DB configurations
	DataMaxOpenConns int                // default to core.DefaultDataMaxOpenConns
	DataMaxIdleConns int                // default to core.DefaultDataMaxIdleConns
	AuxMaxOpenConns  int                // default to core.DefaultAuxMaxOpenConns
	AuxMaxIdleConns  int                // default to core.DefaultAuxMaxIdleConns
	DBConnect        core.DBConnectFunc // default to core.dbConnect
}
```

So for `pb_data`:

```go
app := pocketbase.NewWithConfig(pocketbase.Config{
    DefaultDataDir:  "/my/custom/pb_data",
    HideStartBanner: true,
    DefaultDev:      false,
})
```

Note the word "Default" in the field names - these are defaults that CLI flags can still override. That is intentional design: the binary user can still pass `--dir=...` to override them if needed.

---

### Layer 2: HTTP address (`apis.ServeConfig`)

The HTTP host/port is a **different concern** - it belongs to the `serve` subcommand, not the app itself. It's handled in `cmd/serve.go` which calls `apis.Serve()`:

```39:45:cmd/serve.go
			err := apis.Serve(app, apis.ServeConfig{
				HttpAddr:           httpAddr,
				HttpsAddr:          httpsAddr,
				ShowStartBanner:    showStartBanner,
				AllowedOrigins:     allowedOrigins,
				CertificateDomains: args,
			})
```

There is **no field** in `pocketbase.Config` for this, by design. The three correct ways to configure it programmatically are:

---

#### Option A: Use the `OnServe` hook to change `e.Server.Addr`

This is the most idiomatic approach if you still want `pb.Start()`. The `ServeEvent` exposes the `*http.Server` directly, and the listener is created using `e.Server.Addr` inside the hook chain:

```go
app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
    Func: func(e *core.ServeEvent) error {
        e.Server.Addr = "0.0.0.0:9090"
        return e.Next()
    },
    Priority: 999, // run before the inner finalizer creates the listener
})

if err := app.Start(); err != nil {
    log.Fatal(err)
}
```

This works because inside `apis.Serve()`, `addr := e.Server.Addr` is read after the hooks run:

```230:247:apis/serve.go
		addr := e.Server.Addr
		if addr == "" {
			// fallback similar to the std Server.ListenAndServe/ListenAndServeTLS
			if config.HttpsAddr != "" {
				addr = ":https"
			} else {
				addr = ":http"
			}
		}

		if e.Listener == nil {
			listener, err = net.Listen("tcp", addr)
			if err != nil {
				return err
			}
		} else {
			listener = e.Listener
		}
```

#### Option B: Set a custom `net.Listener` via `e.Listener`

Even more direct - you control the listening socket entirely:

```go
app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
    Func: func(e *core.ServeEvent) error {
        ln, err := net.Listen("tcp", "0.0.0.0:9090")
        if err != nil {
            return err
        }
        e.Listener = ln
        return e.Next()
    },
})
```

The `Listener` field is explicitly documented for this purpose:

```101:111:core/events.go
type ServeEvent struct {
	hook.Event
	App         App
	Router      *router.Router[*RequestEvent]
	Server      *http.Server
	CertManager *autocert.Manager

	// Listener allow specifying a custom network listener.
	//
	// Leave it nil to use the default net.Listen("tcp", e.Server.Addr).
	Listener net.Listener
```

#### Option C: Skip `pb.Start()`, register your own serve command

If you want full control with no reliance on hooks, skip `pb.Start()` (which registers the default serve command) and use `pb.Execute()` instead with your own command:

```go
app := pocketbase.NewWithConfig(pocketbase.Config{
    DefaultDataDir: "/my/pb_data",
})

app.RootCmd.AddCommand(cmd.NewSuperuserCommand(app)) // keep superuser cmd
app.RootCmd.AddCommand(&cobra.Command{
    Use:          "serve",
    SilenceUsage: true,
    RunE: func(command *cobra.Command, args []string) error {
        return apis.Serve(app, apis.ServeConfig{
            HttpAddr:        "0.0.0.0:9090",
            ShowStartBanner: true,
            AllowedOrigins:  []string{"*"},
        })
    },
})

if err := app.Execute(); err != nil {
    log.Fatal(err)
}
```

---

## Summary

| Setting | Correct API |
|---|---|
| `pb_data` location | `pocketbase.NewWithConfig(Config{DefaultDataDir: "..."})` |
| Dev mode | `pocketbase.NewWithConfig(Config{DefaultDev: true})` |
| Encryption env | `pocketbase.NewWithConfig(Config{DefaultEncryptionEnv: "..."})` |
| DB pool sizes | `pocketbase.NewWithConfig(Config{DataMaxOpenConns: ...})` |
| HTTP host/port | `OnServe` hook (change `e.Server.Addr` or set `e.Listener`), or custom serve command |
| HTTPS/TLS | Same as above via a custom serve command calling `apis.Serve()` with `HttpsAddr` set |

Injecting values into `os.Args` is indeed hacky and unnecessary. The `DefaultDataDir` in `Config` is exactly what PocketBase provides for the data directory, and the `OnServe` hook (Option A or B above) is the idiomatic way to override the HTTP address without touching CLI args at all.