# Passkeys with Go + React

A minimal educational WebAuthn/passkey example using:

- Go 1.26+
- `github.com/go-webauthn/webauthn` v0.18.2
- React 19 + Vite 7 + TypeScript
- `@simplewebauthn/browser` v14.0.0

The server uses an in-memory user/credential store and in-memory WebAuthn ceremony sessions. This is intentionally simple for learning and is **not production storage**.

## Project layout

```text
server/
  main.go          Go API: WebAuthn config, in-memory store, CORS, endpoints
web/
  src/App.tsx      React UI: register / sign in with a passkey
  src/style.css    Styles
  vite.config.ts   Vite dev server (pinned to localhost:5174)
  tsconfig.json    TypeScript config
```

## Ports

| Service  | URL                     | Configured in                                         |
|----------|-------------------------|-------------------------------------------------------|
| Go API   | `http://localhost:8082` | `server/main.go` (`ListenAndServe`), `API` in `web/src/App.tsx` |
| React UI | `http://localhost:5174` | `web/vite.config.ts`, `frontendURL` in `server/main.go` |

Each port is set in two places, and both must match. `frontendURL` is the allowed WebAuthn origin and the CORS origin, so if the UI runs anywhere else, the server rejects its requests.

## Run

### 1. Start the Go API

```bash
cd server
go mod tidy
go run .
```

### 2. Start React

In another terminal:

```bash
cd web
npm install
npm run dev
```

Open:

http://localhost:5174

Use `localhost`, not `127.0.0.1`: the server only accepts `localhost` as the WebAuthn relying party ID and `http://localhost:5174` as the origin. Vite is pinned to port 5174 (`strictPort`) and fails to start rather than moving to another port if 5174 is taken.

To type-check and produce a production build of the client:

```bash
cd web
npm run build
```

## Using the demo

1. Enter an email and click **Create Passkey**. Your browser/OS prompts you to create a passkey.
2. With the same email, click **Sign in with Passkey**. On success, a welcome screen is shown.
3. **Sign out** only clears the client state. The demo server doesn't create an application session.

The client disables the buttons while a ceremony is in progress, warns when the browser doesn't support WebAuthn, and shows readable errors (for example when the prompt is cancelled or the authenticator is already registered).

## Flow

Registration:

```text
React
  -> POST /register/begin  { email }
  <- { publicKey: creation options }   + session cookie

React
  -> startRegistration({ optionsJSON: publicKey })
  -> Authenticator creates passkey

React
  -> POST /register/finish  (attestation response, session cookie)
  -> Go verifies and stores credential
```

Login:

```text
React
  -> POST /login/begin  { email }
  <- { publicKey: request options }    + session cookie

React
  -> startAuthentication({ optionsJSON: publicKey })
  -> Authenticator signs challenge

React
  -> POST /login/finish  (assertion response, session cookie)
  -> Go verifies signature
```

go-webauthn wraps its options as `{ publicKey: {...} }`, but SimpleWebAuthn expects only the inner object. The client unwraps `publicKey` before calling `startRegistration` / `startAuthentication`.

The ceremony session ID is stored in an `HttpOnly` cookie, so every request uses `credentials: "include"`, and the server sends `Access-Control-Allow-Credentials: true`.

## Troubleshooting

**"Failed to fetch"** usually means the browser reached something other than the Go API. Check that only the Go server is listening on the API port:

```bash
lsof -nP -iTCP:8082 -sTCP:LISTEN
```

Another app (for example a container) bound to the same port on IPv4 can capture `localhost` requests from the browser even though `curl localhost:8082` reaches the Go server over IPv6. Its responses lack CORS headers, so the browser blocks them and `fetch` throws. Stop the other app or change the API port.

**"user not found; register a passkey first"** after restarting the server: everything is stored in memory, so register again.

**Origin / RP ID errors**: make sure you opened `http://localhost:5174` (not `127.0.0.1` or a different port).

## Important

The example keeps the WebAuthn session and credential in memory. Restarting the Go process loses all registered passkeys. Passkeys created earlier stay in your authenticator, but the server no longer knows them.

For production, persist the complete WebAuthn credential record and ceremony session state securely, use HTTPS, add proper application sessions/authorization, CSRF protection where applicable, rate limiting, logging, and error handling.
