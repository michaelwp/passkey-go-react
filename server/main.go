package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	rpID        = "localhost"
	frontendURL = "http://localhost:5173"
)

type User struct {
	ID          []byte
	Name        string
	DisplayName string

	Credentials []webauthn.Credential
}

func (u *User) WebAuthnID() []byte {
	return u.ID
}

func (u *User) WebAuthnName() string {
	return u.Name
}

func (u *User) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

type Store struct {
	mu sync.RWMutex

	users map[string]*User

	registrationSessions map[string]webauthn.SessionData
	loginSessions        map[string]webauthn.SessionData
}

var store = &Store{
	users:                make(map[string]*User),
	registrationSessions: make(map[string]webauthn.SessionData),
	loginSessions:        make(map[string]webauthn.SessionData),
}

var webAuthn *webauthn.WebAuthn

func main() {
	var err error

	webAuthn, err = webauthn.New(&webauthn.Config{
		RPDisplayName: "Passkey Go + React Demo",
		RPID:          rpID,

		// The browser app runs on port 5173 during development.
		RPOrigins: []string{
			frontendURL,
		},

		// WebAuthn Level 3 requires the top origin to be explicitly
		// configured when using the default strict verification mode.
		RPTopOrigins: []string{
			frontendURL,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /register/begin", beginRegistration)
	mux.HandleFunc("POST /register/finish", finishRegistration)

	mux.HandleFunc("POST /login/begin", beginLogin)
	mux.HandleFunc("POST /login/finish", finishLogin)

	handler := withCORS(mux)

	log.Println("API listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

type userRequest struct {
	Email string `json:"email"`
}

func getOrCreateUser(email string) *User {
	store.mu.Lock()
	defer store.mu.Unlock()

	if user, ok := store.users[email]; ok {
		return user
	}

	id := uuid.New()

	user := &User{
		ID:          id[:],
		Name:        email,
		DisplayName: email,
	}

	store.users[email] = user

	return user
}

func getUser(email string) (*User, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	user, ok := store.users[email]
	if !ok {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func sessionCookieName(kind string) string {
	return "passkey_" + kind + "_session"
}

func setSessionCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // localhost only; use Secure=true with HTTPS in production
	})
}

func readSessionCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", errors.New("session cookie not found")
	}

	return cookie.Value, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func beginRegistration(w http.ResponseWriter, r *http.Request) {
	var req userRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}

	if req.Email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	user := getOrCreateUser(req.Email)

	creation, session, err := webAuthn.BeginRegistration(
		user,
		webauthn.WithRegistrationOrigin(frontendURL),
	)
	if err != nil {
		http.Error(w, "could not begin registration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sessionID := uuid.NewString()

	store.mu.Lock()
	store.registrationSessions[sessionID] = *session
	store.mu.Unlock()

	setSessionCookie(w, sessionCookieName("registration"), sessionID)

	writeJSON(w, http.StatusOK, creation)
}

func finishRegistration(w http.ResponseWriter, r *http.Request) {
	sessionID, err := readSessionCookie(r, sessionCookieName("registration"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	store.mu.RLock()
	session, ok := store.registrationSessions[sessionID]
	store.mu.RUnlock()

	if !ok {
		http.Error(w, "registration session not found", http.StatusBadRequest)
		return
	}

	store.mu.RLock()
	var user *User
	for _, candidate := range store.users {
		if string(candidate.ID) == string(session.UserID) {
			user = candidate
			break
		}
	}
	store.mu.RUnlock()

	if user == nil {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	credential, err := webAuthn.FinishRegistration(user, session, r)
	if err != nil {
		http.Error(w, "registration failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	store.mu.Lock()
	user.Credentials = append(user.Credentials, *credential)
	delete(store.registrationSessions, sessionID)
	store.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "passkey registered successfully",
	})
}

func beginLogin(w http.ResponseWriter, r *http.Request) {
	var req userRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}

	user, err := getUser(req.Email)
	if err != nil {
		http.Error(w, "user not found; register a passkey first", http.StatusNotFound)
		return
	}

	assertion, session, err := webAuthn.BeginLogin(
		user,
		webauthn.WithLoginOrigin(frontendURL),
	)
	if err != nil {
		http.Error(w, "could not begin login: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sessionID := uuid.NewString()

	store.mu.Lock()
	store.loginSessions[sessionID] = *session
	store.mu.Unlock()

	setSessionCookie(w, sessionCookieName("login"), sessionID)

	writeJSON(w, http.StatusOK, assertion)
}

func finishLogin(w http.ResponseWriter, r *http.Request) {
	sessionID, err := readSessionCookie(r, sessionCookieName("login"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	store.mu.RLock()
	session, ok := store.loginSessions[sessionID]
	store.mu.RUnlock()

	if !ok {
		http.Error(w, "login session not found", http.StatusBadRequest)
		return
	}

	store.mu.RLock()
	var user *User
	for _, candidate := range store.users {
		if string(candidate.ID) == string(session.UserID) {
			user = candidate
			break
		}
	}
	store.mu.RUnlock()

	if user == nil {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	_, err = webAuthn.FinishLogin(user, session, r)
	if err != nil {
		http.Error(w, "authentication failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	store.mu.Lock()
	delete(store.loginSessions, sessionID)
	store.mu.Unlock()

	// In a real application, create your normal authenticated application
	// session here, preferably with a secure, HttpOnly cookie.
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "login successful",
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", frontendURL)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
