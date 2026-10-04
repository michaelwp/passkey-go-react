import { useState } from "react";
import {
  browserSupportsWebAuthn,
  startAuthentication,
  startRegistration,
  WebAuthnError,
  type PublicKeyCredentialCreationOptionsJSON,
  type PublicKeyCredentialRequestOptionsJSON,
} from "@simplewebauthn/browser";

const API = "http://localhost:8082";

// go-webauthn wraps the options as { publicKey: {...}, mediation?: ... },
// while SimpleWebAuthn expects the inner publicKey object.
type CredentialCreation = { publicKey: PublicKeyCredentialCreationOptionsJSON };
type CredentialAssertion = { publicKey: PublicKeyCredentialRequestOptionsJSON };

type Status = { kind: "info" | "success" | "error"; text: string };

async function jsonRequest<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${API}${path}`, {
    method: "POST",
    credentials: "include", // send/receive the WebAuthn session cookie
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const message = (await response.text()).trim();
    throw new Error(message || `Request failed (${response.status})`);
  }

  return response.json() as Promise<T>;
}

function describeError(error: unknown, fallback: string): string {
  if (error instanceof WebAuthnError) {
    switch (error.code) {
      case "ERROR_AUTHENTICATOR_PREVIOUSLY_REGISTERED":
        return "This authenticator already has a passkey for this account.";
      case "ERROR_INVALID_DOMAIN":
      case "ERROR_INVALID_RP_ID":
        return "This page's origin doesn't match the server's relying party ID.";
    }
  }

  if (error instanceof Error) {
    if (error.name === "NotAllowedError") {
      return "The passkey prompt was cancelled or timed out.";
    }
    return error.message;
  }

  return fallback;
}

export default function App() {
  const [email, setEmail] = useState("demo@example.com");
  const [status, setStatus] = useState<Status | null>(null);
  const [busy, setBusy] = useState(false);
  const [signedInAs, setSignedInAs] = useState<string | null>(null);

  const supported = browserSupportsWebAuthn();
  const trimmedEmail = email.trim();

  async function run(action: () => Promise<void>, fallback: string) {
    if (!trimmedEmail) {
      setStatus({ kind: "error", text: "Please enter an email." });
      return;
    }

    setBusy(true);
    try {
      await action();
    } catch (error) {
      setStatus({ kind: "error", text: describeError(error, fallback) });
    } finally {
      setBusy(false);
    }
  }

  function register() {
    return run(async () => {
      setStatus({ kind: "info", text: "Starting passkey registration..." });

      const { publicKey } = await jsonRequest<CredentialCreation>(
        "/register/begin",
        { email: trimmedEmail },
      );

      const registrationResponse = await startRegistration({
        optionsJSON: publicKey,
      });

      await jsonRequest("/register/finish", registrationResponse);

      setStatus({
        kind: "success",
        text: "Passkey registered. You can now sign in with it.",
      });
    }, "Registration failed.");
  }

  function login() {
    return run(async () => {
      setStatus({ kind: "info", text: "Starting passkey login..." });

      const { publicKey } = await jsonRequest<CredentialAssertion>(
        "/login/begin",
        { email: trimmedEmail },
      );

      const authenticationResponse = await startAuthentication({
        optionsJSON: publicKey,
      });

      await jsonRequest("/login/finish", authenticationResponse);

      setSignedInAs(trimmedEmail);
      setStatus(null);
    }, "Login failed.");
  }

  function logout() {
    // The demo server doesn't issue an app session, so this is client-side only.
    setSignedInAs(null);
    setStatus(null);
  }

  if (signedInAs) {
    return (
      <main className="container">
        <section className="card">
          <h1>Welcome 🎉</h1>
          <p className="description">
            You signed in with a passkey as <strong>{signedInAs}</strong>.
          </p>
          <div className="buttons">
            <button className="secondary" onClick={logout}>
              Sign out
            </button>
          </div>
        </section>
      </main>
    );
  }

  return (
    <main className="container">
      <section className="card">
        <h1>Passkey Demo</h1>

        <p className="description">
          A tiny Go + React WebAuthn example.
        </p>

        {!supported && (
          <p className="message error">
            This browser doesn't support passkeys (WebAuthn).
          </p>
        )}

        <form onSubmit={(event) => event.preventDefault()}>
          <label htmlFor="email">Email</label>

          <input
            id="email"
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            autoComplete="username webauthn"
            disabled={busy}
            required
          />

          <div className="buttons">
            <button type="button" onClick={register} disabled={busy || !supported}>
              Create Passkey
            </button>

            <button
              type="button"
              className="secondary"
              onClick={login}
              disabled={busy || !supported}
            >
              Sign in with Passkey
            </button>
          </div>
        </form>

        {status && <p className={`message ${status.kind}`}>{status.text}</p>}
      </section>
    </main>
  );
}
