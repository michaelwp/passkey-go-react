import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    // Must match frontendURL / RPOrigins in the Go server.
    host: "localhost",
    port: 5174,
    strictPort: true,
  },
});
