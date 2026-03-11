// Returns the JWT from sessionStorage
export function getToken() {
  return sessionStorage.getItem("jwt_token")
}

// Drop-in replacement for fetch() that auto-injects the Bearer token.
// On 401 it clears the session and bounces the user to /login.
export async function apiFetch(url, options = {}) {
  const token = getToken()

  const headers = { ...(options.headers || {}) }

  if (token) {
    headers["Authorization"] = `Bearer ${token}`
  }

  const res = await fetch(url, { ...options, headers })

  if (res.status === 401) {
    sessionStorage.clear()
    window.location.href = "/login"
    return null
  }

  return res
}

