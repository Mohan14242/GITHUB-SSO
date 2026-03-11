import { useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "../auth/AuthContext"

const ERROR_MESSAGES = {
  not_org_member:   "You are not a member of the required GitHub organization (mohans-organization).",
  no_role_assigned: "Your account has no team/role assigned. Contact your platform admin.",
  oauth_failed:     "GitHub OAuth failed. Please try again.",
  org_check_failed: "Could not verify your organization membership. Try again.",
  role_check_failed:"Could not determine your role. Try again.",
  token_failed:     "Failed to issue a session token. Try again.",
}

export default function LoginPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const error = new URLSearchParams(window.location.search).get("error")

  // Already logged in → go home
  useEffect(() => {
    if (user) navigate("/", { replace: true })
  }, [user, navigate])

  return (
    <div style={{
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
      justifyContent: "center",
      minHeight: "100vh",
      background: "#f5f5f5",
    }}>
      <div style={{
        textAlign: "center",
        maxWidth: 420,
        padding: "40px 32px",
        background: "#fff",
        borderRadius: 12,
        boxShadow: "0 2px 16px rgba(0,0,0,0.09)",
      }}>
        <h1 style={{ marginBottom: 8 }}>🚀 Developer Platform</h1>

        <p style={{ color: "#666", marginBottom: 28 }}>
          Sign in with GitHub to continue.<br />
          You must be a member of <strong>mohans-organization</strong>.
        </p>

        {error && (
          <p style={{
            color: "#c0392b",
            background: "#fdecea",
            padding: "10px 14px",
            borderRadius: 6,
            marginBottom: 20,
            fontSize: 14,
          }}>
            {ERROR_MESSAGES[error] ?? "Authentication failed. Please try again."}
          </p>
        )}

        <a
          href="/api/auth/login"
          style={{
            display: "inline-block",
            padding: "12px 28px",
            background: "#24292e",
            color: "#fff",
            textDecoration: "none",
            borderRadius: 6,
            fontSize: 15,
            fontWeight: 600,
            letterSpacing: 0.3,
          }}
        >
          🐙 Sign in with GitHub
        </a>

      </div>
    </div>
  )
}