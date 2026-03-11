import { createContext, useContext, useState, useCallback, useEffect } from "react"

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  // Rehydrate from sessionStorage on page load
  useEffect(() => {
    const token = sessionStorage.getItem("jwt_token")
    const role  = sessionStorage.getItem("user_role")
    const login = sessionStorage.getItem("user_login")

    if (token && role && login) {
      setUser({ token, role, login })
    }
    setLoading(false)
  }, [])

  const login = useCallback((token, role, login) => {
    sessionStorage.setItem("jwt_token",  token)
    sessionStorage.setItem("user_role",  role)
    sessionStorage.setItem("user_login", login)
    setUser({ token, role, login })
  }, [])

  const logout = useCallback(() => {
    sessionStorage.clear()
    setUser(null)
  }, [])

  // Returns true if the current user meets or exceeds the required role
  const hasRole = useCallback((minRole) => {
    const priority = { admin: 4, operator: 3, developer: 2, readonly: 1 }
    if (!user) return false
    return (priority[user.role] ?? 0) >= (priority[minRole] ?? 0)
  }, [user])

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, hasRole }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used within <AuthProvider>")
  return ctx
}