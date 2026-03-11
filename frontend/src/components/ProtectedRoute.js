import { Navigate } from "react-router-dom"
import { useAuth } from "../auth/AuthContext"

export default function ProtectedRoute({ minRole = "readonly", children }) {
  const { user, loading, hasRole } = useAuth()

  if (loading) return <p style={{ textAlign: "center", marginTop: 80 }}>Loading…</p>

  // Not logged in → login page
  if (!user) return <Navigate to="/login" replace />

  // Logged in but wrong role → access denied (don't expose the route)
  if (!hasRole(minRole)) {
    return (
      <div style={{ textAlign: "center", marginTop: 80 }}>
        <h2>🚫 Access Denied</h2>
        <p>
          This page requires <strong>{minRole}</strong> role or higher.
          Your current role is <strong>{user.role}</strong>.
        </p>
        <p style={{ color: "#666", fontSize: 13 }}>
          Contact your platform admin if you need access.
        </p>
      </div>
    )
  }

  return children
}