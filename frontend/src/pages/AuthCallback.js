import { useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "../auth/AuthContext"

// Backend redirects here after OAuth with ?token=&role=&login=
export default function AuthCallback() {
  const { login } = useAuth()
  const navigate  = useNavigate()

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token  = params.get("token")
    const role   = params.get("role")
    const login_ = params.get("login")

    if (token && role && login_) {
      login(token, role, login_)
      navigate("/", { replace: true })
    } else {
      navigate("/login?error=oauth_failed", { replace: true })
    }
  }, [login, navigate])

  return (
    <p style={{ textAlign: "center", marginTop: 120, color: "#666" }}>
      Authenticating…
    </p>
  )
}