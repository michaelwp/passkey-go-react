import { useState } from "react";
import {
  startAuthentication,
  startRegistration,
} from "@simplewebauthn/browser";

const API = "http://localhost:8080";

async function jsonRequest(path: string, body: unknown) {
  const response = await fetch(`${API}${path}`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || "Request failed");
  }

  return response.json();
}

export default function App() {
  const [email, setEmail] = useState("demo@example.com");
  const [message, setMessage] = useState("");

  async function register() {
    try {
      setMessage("Starting passkey registration...");

      const options = await jsonRequest("/register/begin", { email });

      const registrationResponse = await startRegistration({
        optionsJSON: options,
      });

      await jsonRequest("/register/finish", registrationResponse);

      setMessage("Passkey registered successfully.");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Registration failed.");
    }
  }

  async function login() {
    try {
      setMessage("Starting passkey login...");

      const options = await jsonRequest("/login/begin", { email });

      const authenticationResponse = await startAuthentication({
        optionsJSON: options,
      });

      await jsonRequest("/login/finish", authenticationResponse);

      setMessage("Login successful 🎉");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Login failed.");
    }
  }

  return (
    <main className="container">
      <section className="card">
        <h1>Passkey Demo</h1>

        <p className="description">
          A tiny Go + React WebAuthn example.
        </p>

        <label htmlFor="email">Email</label>

        <input
          id="email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          autoComplete="username webauthn"
        />

        <div className="buttons">
          <button onClick={register}>
            Create Passkey
          </button>

          <button className="secondary" onClick={login}>
            Sign in with Passkey
          </button>
        </div>

        {message && <p className="message">{message}</p>}
      </section>
    </main>
  );
}
