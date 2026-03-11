import { BrowserRouter, Routes, Route, Link, useLocation, Navigate } from "react-router-dom"
import ServiceCreationApprovals from "./pages/ServiceCreationApprovals" 
import { AuthProvider, useAuth } from "./auth/AuthContext"
import ServicesList from "./pages/ServicesList"
import ServiceDashboard from "./pages/ServiceDashboard"
import CreateServicePage from "./pages/CreateServicePage"
import AdminApprovals from "./pages/AdminApprovals"
import LoginPage from "./pages/LoginPage"
import AuthCallback from "./pages/AuthCallback"
import ProtectedRoute from "./components/ProtectedRoute"
import { fetchPlatformStats } from "./api/services"
import { useState, useEffect } from "react"
import AuditingPage from "./pages/AuditingPage"
import TemplateVersionsPage from "./pages/TemplateVersionsPage"
import TemplateRegistryPage from "./pages/TemplateRegistryPage"


const ROLE_BADGE = {
  admin:     { label: "Admin",     color: "#6366f1" },
  operator:  { label: "Operator",  color: "#0ea5e9" },
  developer: { label: "Developer", color: "#10b981" },
  readonly:  { label: "Read-only", color: "#64748b" },
}

const NAV_ITEMS = [
  {
    key: "home",
    label: "Home",
    path: "/",
    icon: (
      <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" viewBox="0 0 24 24">
        <path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/>
        <path d="M9 22V12h6v10"/>
      </svg>
    ),
    minRole: "readonly",
  },
  {
    key: "services",
    label: "Services",
    path: "/services",
    icon: (
      <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" viewBox="0 0 24 24">
        <rect x="2" y="3" width="20" height="14" rx="2"/>
        <path d="M8 21h8M12 17v4"/>
      </svg>
    ),
    minRole: "readonly",
  },
  {
    key: "create",
    label: "Create Service",
    path: "/create",
    icon: (
      <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="10"/>
        <path d="M12 8v8M8 12h8"/>
      </svg>
    ),
    minRole: "developer",
  },
  {
    key: "approvals",
    label: "Deployment Approvals",
    path: "/approvals",
    icon: (
      <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" viewBox="0 0 24 24">
        <path d="M9 11l3 3L22 4"/>
        <path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11"/>
      </svg>
    ),
    minRole: "operator",
  },
  {
    key: "service-approvals",
    label: "Service Creation Approvals",
    path: "/service-approvals",
    icon: (
      <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" viewBox="0 0 24 24">
        <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
        <path d="M14 2v6h6M9 13h6M9 17h4"/>
      </svg>
    ),
    minRole: "operator",
  },
  {
    key: "auditing",
    label: "Auditing",
    path: "/auditing",
    icon: (
      <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" viewBox="0 0 24 24">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
        <path d="M9 12l2 2 4-4"/>
      </svg>
    ),
    minRole: "operator",
  },
  {
    key: "templates",
    label: "Template Versions",
    path: "/templates",
    icon: (
      <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" viewBox="0 0 24 24">
        <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/>
        <path d="M3.27 6.96L12 12.01l8.73-5.05M12 22.08V12"/>
      </svg>
    ),
    minRole: "operator",
  },
  {
    key: "template-registry",
    label: "Template Registry",
    path: "/template-registry",
    icon: (
      <svg width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" viewBox="0 0 24 24">
        <ellipse cx="12" cy="5" rx="9" ry="3"/>
        <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/>
        <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
      </svg>
    ),
    minRole: "developer",
  },

]

/* ─────────────────────────────────────────
   Placeholder page
───────────────────────────────────────── */
function ComingSoon({ title }) {
  return (
    <div style={{
      display: "flex", flexDirection: "column",
      alignItems: "center", justifyContent: "center",
      height: "60vh", gap: 16,
    }}>
      <div style={{ fontSize: 48 }}>🚧</div>
      <h2 style={{ color: "#e2e8f0", margin: 0 }}>{title}</h2>
      <p style={{ color: "#64748b", margin: 0 }}>This feature is coming soon.</p>
    </div>
  )
}

/* ─────────────────────────────────────────
   Welcome / Home page
───────────────────────────────────────── */
function WelcomePage() {
  const { user } = useAuth()
  const badge = ROLE_BADGE[user?.role] ?? ROLE_BADGE.readonly

  const [stats, setStats] = useState({
    totalServices:           null,
    deploymentsToday:        null,
    pendingDeployments:      null,
    pendingServiceCreations: null,
    activePipelines:         null,
  })
  const [statsLoading, setStatsLoading] = useState(true)

  useEffect(() => {
    async function loadStats() {
      try {
        const data = await fetchPlatformStats()
        setStats(data)
      } catch (err) {
        console.error("[WelcomePage] Failed to load stats:", err)
      } finally {
        setStatsLoading(false)
      }
    }
    loadStats()
    const interval = setInterval(loadStats, 30000)
    return () => clearInterval(interval)
  }, [])

  const statCards = [
    {
      label: "Services",
      value: stats.totalServices,
      icon: "⚙️",
      color: "#6366f1",
      path: "/services",
    },
    {
      label: "Deployments Today",
      value: stats.deploymentsToday,
      icon: "🚀",
      color: "#10b981",
      path: null,
    },
    {
      label: "Deployment Approvals",
      value: stats.pendingDeployments,
      icon: "⏳",
      color: "#f59e0b",
      path: "/approvals",
    },
    {
      label: "Service Creation Approvals",
      value: stats.pendingServiceCreations,
      icon: "📋",
      color: "#e879f9",
      path: "/service-approvals",
    },
  ]

  const quickActions = NAV_ITEMS.filter(n => n.key !== "home" && n.key !== "services")

  return (
    <div style={{ padding: "40px 48px", maxWidth: 1100 }}>

      {/* ── Hero ── */}
      <div style={{
        background: "linear-gradient(135deg, #1e293b 0%, #0f172a 60%, #1a1040 100%)",
        border: "1px solid #334155",
        borderRadius: 16,
        padding: "48px 48px",
        marginBottom: 40,
        position: "relative",
        overflow: "hidden",
      }}>
        <div style={{
          position: "absolute", inset: 0,
          backgroundImage: "linear-gradient(rgba(99,102,241,0.07) 1px, transparent 1px), linear-gradient(90deg, rgba(99,102,241,0.07) 1px, transparent 1px)",
          backgroundSize: "32px 32px",
          pointerEvents: "none",
        }}/>
        <div style={{
          position: "absolute", top: -60, right: -60,
          width: 300, height: 300,
          background: "radial-gradient(circle, rgba(99,102,241,0.15) 0%, transparent 70%)",
          pointerEvents: "none",
        }}/>
        <div style={{ position: "relative" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 20 }}>
            <span style={{ fontSize: 36 }}>🚀</span>
            <span style={{
              fontSize: 11, fontWeight: 700, letterSpacing: "0.15em",
              textTransform: "uppercase", color: "#6366f1", fontFamily: "monospace",
            }}>
              Internal Developer Platform
            </span>
          </div>
          <h1 style={{
            margin: "0 0 12px", fontSize: 40, fontWeight: 800,
            color: "#f1f5f9", fontFamily: "'Georgia', serif", lineHeight: 1.15,
          }}>
            Welcome back,<br />
            <span style={{ color: "#6366f1" }}>{user?.login ?? "Engineer"}</span>
          </h1>
          <p style={{ margin: "0 0 28px", color: "#94a3b8", fontSize: 16, lineHeight: 1.6, maxWidth: 520 }}>
            Manage services, trigger deployments, review approvals,
            and audit activity — all from one place.
          </p>
          <div style={{ display: "flex", gap: 10, flexWrap: "wrap" }}>
            <span style={{
              display: "inline-flex", alignItems: "center", gap: 6,
              padding: "6px 14px",
              background: badge.color + "22", border: `1px solid ${badge.color}55`,
              borderRadius: 20, color: badge.color,
              fontSize: 12, fontWeight: 700, letterSpacing: "0.05em",
            }}>● {badge.label}</span>
            <span style={{
              display: "inline-flex", alignItems: "center", gap: 6,
              padding: "6px 14px",
              background: "#1e293b", border: "1px solid #334155",
              borderRadius: 20, color: "#64748b",
              fontSize: 12, fontFamily: "monospace",
            }}>👤 {user?.login}</span>
          </div>
        </div>
      </div>

      {/* ── Stats row ── */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 16, marginBottom: 40 }}>
        {statCards.map((s) => {
          const card = (
            <div
              style={{
                background: "#0f172a", border: "1px solid #1e293b",
                borderRadius: 12, padding: "20px 24px",
                cursor: s.path ? "pointer" : "default",
                transition: "border-color 0.15s", height: "100%",
              }}
              onMouseEnter={e => { if (s.path) e.currentTarget.style.borderColor = s.color }}
              onMouseLeave={e => { if (s.path) e.currentTarget.style.borderColor = "#1e293b" }}
            >
              <div style={{ fontSize: 24, marginBottom: 10 }}>{s.icon}</div>

              {statsLoading ? (
                <div style={{
                  width: 48, height: 32, borderRadius: 6,
                  background: "#1e293b", marginBottom: 8,
                  animation: "pulse 1.5s ease-in-out infinite",
                }}/>
              ) : (
                <div style={{
                  fontSize: 32, fontWeight: 800,
                  color: s.value > 0 ? s.color : "#f1f5f9",
                  fontFamily: "monospace", transition: "color 0.3s",
                }}>
                  {s.value ?? 0}
                </div>
              )}

              <div style={{ fontSize: 12, color: "#475569", marginTop: 4, fontWeight: 500 }}>
                {s.label}
              </div>
              {s.path && !statsLoading && (
                <div style={{ fontSize: 11, color: s.color, marginTop: 8, opacity: 0.7 }}>
                  View →
                </div>
              )}
            </div>
          )

          return s.path ? (
            <Link key={s.label} to={s.path} style={{ textDecoration: "none" }}>
              {card}
            </Link>
          ) : (
            <div key={s.label}>{card}</div>
          )
        })}
      </div>

      {/* ── Quick actions ── */}
      <h3 style={{
        color: "#64748b", fontSize: 11, fontWeight: 700,
        letterSpacing: "0.12em", textTransform: "uppercase", marginBottom: 16,
      }}>
        Quick Actions
      </h3>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 14 }}>
        {quickActions.map((item) => (
          <Link
            key={item.key} to={item.path}
            style={{
              display: "flex", alignItems: "center", gap: 14,
              padding: "18px 20px",
              background: "#0f172a", border: "1px solid #1e293b",
              borderRadius: 12, textDecoration: "none",
              color: "#cbd5e1", fontSize: 14, fontWeight: 500,
              transition: "border-color 0.15s, background 0.15s",
            }}
            onMouseEnter={e => { e.currentTarget.style.borderColor = "#6366f1"; e.currentTarget.style.background = "#13172a" }}
            onMouseLeave={e => { e.currentTarget.style.borderColor = "#1e293b";  e.currentTarget.style.background = "#0f172a" }}
          >
            <span style={{ color: "#6366f1" }}>{item.icon}</span>
            {item.label}
          </Link>
        ))}
      </div>

      <style>{`
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
      `}</style>
    </div>
  )
}

/* ─────────────────────────────────────────
   Sidebar
───────────────────────────────────────── */
function Sidebar({ collapsed, setCollapsed }) {
  const { user, logout, hasRole } = useAuth()
  const location = useLocation()
  const badge = ROLE_BADGE[user?.role] ?? ROLE_BADGE.readonly

  const visibleNav = NAV_ITEMS.filter(item => hasRole(item.minRole))

  return (
    <aside style={{
      width: collapsed ? 64 : 240,
      minHeight: "100vh",
      background: "#080d14",
      borderRight: "1px solid #1e293b",
      display: "flex",
      flexDirection: "column",
      transition: "width 0.22s cubic-bezier(.4,0,.2,1)",
      overflow: "hidden",
      flexShrink: 0,
      position: "sticky",
      top: 0,
    }}>

      {/* ── Logo row ── */}
      <div style={{
        padding: collapsed ? "20px 0" : "24px 20px",
        borderBottom: "1px solid #1e293b",
        display: "flex",
        alignItems: "center",
        justifyContent: collapsed ? "center" : "space-between",
        gap: 10,
        minHeight: 72,
      }}>
        {!collapsed && (
          <>
            <div style={{ display: "flex", alignItems: "center", gap: 10, overflow: "hidden" }}>
              <span style={{ fontSize: 22, flexShrink: 0 }}>🚀</span>
              <span style={{
                fontWeight: 800, fontSize: 13, color: "#f1f5f9",
                letterSpacing: "-0.01em", whiteSpace: "nowrap",
                fontFamily: "'Georgia', serif",
              }}>
                DevPlatform
              </span>
            </div>
            <button
              onClick={() => setCollapsed(true)}
              style={{
                background: "none", border: "none", color: "#475569",
                cursor: "pointer", padding: 4, borderRadius: 4,
                flexShrink: 0, display: "flex", alignItems: "center",
              }}
            >
              <svg width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
                <path d="M15 18l-6-6 6-6"/>
              </svg>
            </button>
          </>
        )}
        {collapsed && (
          <button
            onClick={() => setCollapsed(false)}
            style={{
              background: "none", border: "none", cursor: "pointer",
              padding: 0, display: "flex", alignItems: "center", justifyContent: "center",
            }}
            title="Expand sidebar"
          >
            <span style={{ fontSize: 22 }}>🚀</span>
          </button>
        )}
      </div>

      {/* ── Nav items ── */}
      <nav style={{ flex: 1, padding: "12px 8px", display: "flex", flexDirection: "column", gap: 2 }}>
        {visibleNav.map((item) => {
          const active = location.pathname === item.path
          return (
            <Link
              key={item.key}
              to={item.path}
              title={collapsed ? item.label : undefined}
              style={{
                display: "flex", alignItems: "center", gap: 12,
                padding: collapsed ? "10px 0" : "10px 12px",
                justifyContent: collapsed ? "center" : "flex-start",
                borderRadius: 8, textDecoration: "none",
                color: active ? "#f1f5f9" : "#64748b",
                background: active ? "#1e293b" : "transparent",
                fontWeight: active ? 600 : 400,
                fontSize: 13.5, whiteSpace: "nowrap",
                position: "relative", transition: "background 0.12s, color 0.12s",
              }}
              onMouseEnter={e => {
                if (!active) { e.currentTarget.style.background = "#0f172a"; e.currentTarget.style.color = "#e2e8f0" }
              }}
              onMouseLeave={e => {
                if (!active) { e.currentTarget.style.background = "transparent"; e.currentTarget.style.color = "#64748b" }
              }}
            >
              {active && (
                <span style={{
                  position: "absolute", left: 0, top: "20%", bottom: "20%",
                  width: 3, borderRadius: 2, background: "#6366f1",
                }}/>
              )}
              <span style={{ color: active ? "#6366f1" : "inherit", flexShrink: 0 }}>
                {item.icon}
              </span>
              {!collapsed && item.label}
            </Link>
          )
        })}
      </nav>

      {/* ── User footer ── */}
      <div style={{
        padding: collapsed ? "16px 0" : "16px",
        borderTop: "1px solid #1e293b",
        display: "flex", flexDirection: "column", gap: 10,
        alignItems: collapsed ? "center" : "stretch",
      }}>
        {!collapsed ? (
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <div style={{
              width: 32, height: 32, borderRadius: "50%",
              background: "#1e293b", display: "flex",
              alignItems: "center", justifyContent: "center",
              fontSize: 14, color: "#6366f1", fontWeight: 700, flexShrink: 0,
            }}>
              {user?.login?.[0]?.toUpperCase()}
            </div>
            <div style={{ overflow: "hidden" }}>
              <div style={{
                fontSize: 12, fontWeight: 600, color: "#e2e8f0",
                whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis",
              }}>
                {user?.login}
              </div>
              <div style={{ fontSize: 11, color: badge.color, fontWeight: 600 }}>
                {badge.label}
              </div>
            </div>
          </div>
        ) : (
          <div style={{
            width: 32, height: 32, borderRadius: "50%",
            background: "#1e293b", display: "flex",
            alignItems: "center", justifyContent: "center",
            fontSize: 14, color: "#6366f1", fontWeight: 700,
          }}
          title={user?.login}
          >
            {user?.login?.[0]?.toUpperCase()}
          </div>
        )}

        <button
          onClick={logout}
          title={collapsed ? "Sign out" : undefined}
          style={{
            display: "flex", alignItems: "center",
            justifyContent: collapsed ? "center" : "flex-start",
            gap: 8, padding: collapsed ? "8px 0" : "8px 10px",
            background: "none", border: "1px solid #1e293b",
            borderRadius: 7, color: "#475569", cursor: "pointer",
            fontSize: 12, fontWeight: 500, width: "100%",
            transition: "border-color 0.12s, color 0.12s",
          }}
          onMouseEnter={e => { e.currentTarget.style.borderColor = "#e74c3c"; e.currentTarget.style.color = "#e74c3c" }}
          onMouseLeave={e => { e.currentTarget.style.borderColor = "#1e293b"; e.currentTarget.style.color = "#475569" }}
        >
          <svg width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
            <path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4M16 17l5-5-5-5M21 12H9"/>
          </svg>
          {!collapsed && "Sign out"}
        </button>
      </div>
    </aside>
  )
}

/* ─────────────────────────────────────────
   Shell
───────────────────────────────────────── */
function Shell({ children }) {
  const [collapsed, setCollapsed] = useState(false)
  return (
    <div style={{
      display: "flex", minHeight: "100vh",
      background: "#060b12", fontFamily: "'DM Sans', 'Segoe UI', sans-serif",
    }}>
      <Sidebar collapsed={collapsed} setCollapsed={setCollapsed} />
      <main style={{ flex: 1, overflowY: "auto" }}>{children}</main>
    </div>
  )
}

/* ─────────────────────────────────────────
   Routes
───────────────────────────────────────── */
function AppRoutes() {
  return (
    <Routes>
      <Route path="/login"         element={<LoginPage />} />
      <Route path="/auth/callback" element={<AuthCallback />} />

      <Route path="/" element={
        <ProtectedRoute minRole="readonly"><Shell><WelcomePage /></Shell></ProtectedRoute>
      }/>
      <Route path="/services" element={
        <ProtectedRoute minRole="readonly"><Shell><ServicesList /></Shell></ProtectedRoute>
      }/>
      <Route path="/services/:serviceName" element={
        <ProtectedRoute minRole="readonly"><Shell><ServiceDashboard /></Shell></ProtectedRoute>
      }/>
      <Route path="/create" element={
        <ProtectedRoute minRole="developer"><Shell><CreateServicePage /></Shell></ProtectedRoute>
      }/>
      <Route path="/approvals" element={
        <ProtectedRoute minRole="operator"><Shell><AdminApprovals /></Shell></ProtectedRoute>
      }/>
      <Route path="/service-approvals" element={
        <ProtectedRoute minRole="operator"><Shell><ServiceCreationApprovals /></Shell></ProtectedRoute>
      }/>
      <Route path="/auditing" element={
        <ProtectedRoute minRole="operator">
          <Shell><AuditingPage /></Shell>
        </ProtectedRoute>
      }/>
      <Route path="/templates" element={
        <ProtectedRoute minRole="operator">
          <Shell><TemplateVersionsPage /></Shell>
        </ProtectedRoute>
      }/>
      <Route path="/template-registry" element={
        <ProtectedRoute minRole="developer">
          <Shell><TemplateRegistryPage /></Shell>
        </ProtectedRoute>
      }/>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

/* ─────────────────────────────────────────
   Root
───────────────────────────────────────── */
export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <style>{`
          @import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700;800&display=swap');
          *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
          html, body { background: #060b12; }
          ::-webkit-scrollbar { width: 6px; }
          ::-webkit-scrollbar-track { background: #0f172a; }
          ::-webkit-scrollbar-thumb { background: #1e293b; border-radius: 3px; }
          a { text-decoration: none; }
        `}</style>
        <AppRoutes />
      </AuthProvider>
    </BrowserRouter>
  )
}
