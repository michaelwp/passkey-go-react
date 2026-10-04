# Passkeys with Go + React

A minimal educational WebAuthn/passkey example using:

- Go 1.26+
- `github.com/go-webauthn/webauthn` v0.18.2
- React + Vite
- `@simplewebauthn/browser` v14.0.0

The server uses an in-memory user/credential store and in-memory WebAuthn ceremony sessions. This is intentionally simple for learning and is **not production storage**.

## Run

### 1. Start the Go API

```bash
cd server
go mod tidy
go run .
```

The API listens on `http://localhost:8080`.

### 2. Start React

In another terminal:

```bash
cd web
npm install
npm run dev
```

Open the Vite URL, normally:

http://localhost:5173

Use the same email for registration and login.

## Flow

Registration:

```text
React
  -> POST /register/begin
  <- WebAuthn creation options

React
  -> startRegistration()
  -> Authenticator creates passkey

React
  -> POST /register/finish
  -> Go verifies and stores credential
```

Login:

```text
React
  -> POST /login/begin
  <- WebAuthn request options

React
  -> startAuthentication()
  -> Authenticator signs challenge

React
  -> POST /login/finish
  -> Go verifies signature
```

## Important

The example keeps the WebAuthn session and credential in memory. Restarting the Go process loses all registered passkeys.

For production, persist the complete WebAuthn credential record and ceremony session state securely, use HTTPS, add proper application sessions/authorization, CSRF protection where applicable, rate limiting, logging, and error handling.
