import { useState, useEffect } from "react"
import { Link } from "react-router-dom"
import { fetchTemplateVersions } from "../api/templatesApi"

const RUNTIME_COLORS = {
  go:     { bg: "#001a1f", border: "#00b4d855", text: "#00b4d8", dot: "#00b4d8" },
  node:   { bg: "#001a0a", border: "#10b98155", text: "#10b981", dot: "#10b981" },
  python: { bg: "#1a1400", border: "#f59e0b55", text: "#f59e0b", dot: "#f59e0b" },
  java:   { bg: "#1a0a00", border: "#f97316aa", text: "#f97316", dot: "#f97316" },
  dotnet: { bg: "#0d001a", border: "#a855f755", text: "#a855f7", dot: "#a855f7" },
  rust:   { bg: "#1a0500", border: "#ef444455", text: "#ef4444", dot: "#ef4444" },
}

const getRuntimeColor = (runtime) =>
  RUNTIME_COLORS[runtime.toLowerCase()] ?? {
    bg: "#0f172a", border: "#33415555", text: "#64748b", dot: "#64748b",
  }

const RUNTIME_ICONS = {
  go:     "🐹",
  node:   "🟢",
  python: "🐍",
  java:   "☕",
  dotnet: "🔷",
  rust:   "⚙️",
}
const getRuntimeIcon = (r) => RUNTIME_ICONS[r.toLowerCase()] ?? "📦"

export default function TemplateRegistryPage() {
  const [versions, setVersions]         = useState([])
  const [loading, setLoading]           = useState(true)
  const [error, setError]               = useState(null)
  const [filterRuntime, setFilterRuntime] = useState("")
  const [search, setSearch]             = useState("")

  useEffect(() => { load() }, [])

  async function load() {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchTemplateVersions()
      setVersions(Array.isArray(data) ? data : [])
    } catch (err) {
      setError("Failed to load template registry")
    } finally {
      setLoading(false)
    }
  }

  // Derive unique runtimes for filter buttons
  const runtimes = [...new Set(versions.map(v => v.runtime))].sort()

  // Filter + search
  const filtered = versions.filter(v => {
    const matchRuntime = filterRuntime ? v.runtime === filterRuntime : true
    const matchSearch  = search
      ? v.name.toLowerCase().includes(search.toLowerCase()) ||
        v.version.toLowerCase().includes(search.toLowerCase()) ||
        v.runtime.toLowerCase().includes(search.toLowerCase()) ||
        v.description?.toLowerCase().includes(search.toLowerCase())
      : true
    return matchRuntime && matchSearch
  })

  // Group by runtime
  const grouped = filtered.reduce((acc, v) => {
    if (!acc[v.runtime]) acc[v.runtime] = { active: [], deprecated: [] }
    if (v.status === "active") acc[v.runtime].active.push(v)
    else acc[v.runtime].deprecated.push(v)
    return acc
  }, {})

  const totalActive     = versions.filter(v => v.status === "active").length
  const totalDeprecated = versions.filter(v => v.status === "deprecated").length

  const formatTime = (iso) => {
    if (!iso) return "—"
    try { return new Date(iso).toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" }) }
    catch { return iso }
  }

  return (
    <div style={{ padding: "40px 48px", maxWidth: 1100 }}>

      {/* ── Header ── */}
      <div style={{ marginBottom: 32 }}>
        <span style={{
          fontSize: 11, fontWeight: 700, letterSpacing: "0.15em",
          textTransform: "uppercase", color: "#6366f1", fontFamily: "monospace",
        }}>
          Golden Path Templates
        </span>
        <div style={{
          display: "flex", alignItems: "flex-start",
          justifyContent: "space-between", marginTop: 6, gap: 16,
        }}>
          <div>
            <h2 style={{
              color: "#f1f5f9", margin: "0 0 8px",
              fontFamily: "'Georgia', serif", fontSize: 28, fontWeight: 700,
            }}>
              Template Registry
            </h2>
            <p style={{ color: "#475569", fontSize: 14, margin: 0, lineHeight: 1.6 }}>
              Browse available templates before creating a service.
              Deprecated versions cannot be used for new services.
            </p>
          </div>

          <Link
            to="/create"
            style={{
              display: "inline-flex", alignItems: "center", gap: 8,
              padding: "10px 20px", borderRadius: 8, border: "none",
              background: "linear-gradient(135deg, #6366f1, #4f46e5)",
              color: "#fff", fontSize: 13, fontWeight: 700,
              textDecoration: "none", flexShrink: 0,
              boxShadow: "0 4px 14px rgba(99,102,241,0.3)",
            }}
          >
            <svg width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
              <circle cx="12" cy="12" r="10"/><path d="M12 8v8M8 12h8"/>
            </svg>
            Create Service
          </Link>
        </div>
      </div>

      {/* ── Summary banner ── */}
      <div style={{
        display: "grid", gridTemplateColumns: "repeat(3, 1fr)",
        gap: 14, marginBottom: 28,
      }}>
        <div style={{
          background: "#0f172a", border: "1px solid #1e293b",
          borderRadius: 10, padding: "16px 20px",
          display: "flex", alignItems: "center", gap: 14,
        }}>
          <div style={{
            width: 40, height: 40, borderRadius: 8,
            background: "#1e293b",
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 18,
          }}>📦</div>
          <div>
            <div style={{ fontSize: 24, fontWeight: 800, color: "#f1f5f9", fontFamily: "monospace" }}>
              {loading ? "—" : runtimes.length}
            </div>
            <div style={{ fontSize: 12, color: "#475569", marginTop: 2 }}>Runtimes Available</div>
          </div>
        </div>
        <div style={{
          background: "#0f172a", border: "1px solid #10b98122",
          borderRadius: 10, padding: "16px 20px",
          display: "flex", alignItems: "center", gap: 14,
        }}>
          <div style={{
            width: 40, height: 40, borderRadius: 8,
            background: "#001a0f",
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 18,
          }}>✅</div>
          <div>
            <div style={{ fontSize: 24, fontWeight: 800, color: "#10b981", fontFamily: "monospace" }}>
              {loading ? "—" : totalActive}
            </div>
            <div style={{ fontSize: 12, color: "#475569", marginTop: 2 }}>Active Versions</div>
          </div>
        </div>
        <div style={{
          background: "#0f172a", border: "1px solid #e74c3c22",
          borderRadius: 10, padding: "16px 20px",
          display: "flex", alignItems: "center", gap: 14,
        }}>
          <div style={{
            width: 40, height: 40, borderRadius: 8,
            background: "#1a0a0a",
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 18,
          }}>🚫</div>
          <div>
            <div style={{ fontSize: 24, fontWeight: 800, color: "#e74c3c", fontFamily: "monospace" }}>
              {loading ? "—" : totalDeprecated}
            </div>
            <div style={{ fontSize: 12, color: "#475569", marginTop: 2 }}>Deprecated Versions</div>
          </div>
        </div>
      </div>

      {/* ── Search + runtime filter ── */}
      <div style={{ display: "flex", gap: 12, alignItems: "center", marginBottom: 28, flexWrap: "wrap" }}>
        {/* Search */}
        <div style={{ position: "relative", flex: 1, minWidth: 200 }}>
          <svg
            width="14" height="14" fill="none" stroke="#475569"
            strokeWidth="2" viewBox="0 0 24 24"
            style={{ position: "absolute", left: 11, top: "50%", transform: "translateY(-50%)" }}
          >
            <circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35"/>
          </svg>
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search templates…"
            style={{
              width: "100%", padding: "8px 12px 8px 32px",
              background: "#0a0f1a", border: "1px solid #1e293b",
              borderRadius: 7, color: "#e2e8f0", fontSize: 13,
              outline: "none", boxSizing: "border-box",
            }}
            onFocus={e => e.target.style.borderColor = "#6366f1"}
            onBlur={e => e.target.style.borderColor = "#1e293b"}
          />
        </div>

        {/* Runtime filter pills */}
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <button
            onClick={() => setFilterRuntime("")}
            style={{
              padding: "6px 14px", borderRadius: 20, fontSize: 12, fontWeight: 600,
              border: `1px solid ${filterRuntime === "" ? "#6366f1" : "#1e293b"}`,
              background: filterRuntime === "" ? "#6366f122" : "transparent",
              color: filterRuntime === "" ? "#6366f1" : "#475569",
              cursor: "pointer", transition: "all 0.15s",
            }}
          >
            All
          </button>
          {runtimes.map(rt => {
            const c = getRuntimeColor(rt)
            const active = filterRuntime === rt
            return (
              <button
                key={rt}
                onClick={() => setFilterRuntime(active ? "" : rt)}
                style={{
                  padding: "6px 14px", borderRadius: 20, fontSize: 12, fontWeight: 600,
                  border: `1px solid ${active ? c.border : "#1e293b"}`,
                  background: active ? c.bg : "transparent",
                  color: active ? c.text : "#475569",
                  cursor: "pointer", transition: "all 0.15s",
                  display: "flex", alignItems: "center", gap: 6,
                }}
              >
                {getRuntimeIcon(rt)} {rt}
              </button>
            )
          })}
        </div>

        <button
          onClick={load}
          style={{
            padding: "8px 14px", borderRadius: 7,
            border: "1px solid #1e293b", background: "transparent",
            color: "#64748b", cursor: "pointer", fontSize: 13,
            flexShrink: 0,
          }}
          onMouseEnter={e => { e.currentTarget.style.borderColor = "#6366f1"; e.currentTarget.style.color = "#6366f1" }}
          onMouseLeave={e => { e.currentTarget.style.borderColor = "#1e293b"; e.currentTarget.style.color = "#64748b" }}
        >
          ↺
        </button>
      </div>

      {/* ── Loading ── */}
      {loading && (
        <div style={{ display: "flex", alignItems: "center", gap: 10, color: "#475569", padding: "40px 0" }}>
          <div style={{
            width: 16, height: 16, borderRadius: "50%",
            border: "2px solid #1e293b", borderTop: "2px solid #6366f1",
            animation: "spin 1s linear infinite",
          }}/>
          Loading template registry…
        </div>
      )}

      {/* ── Error ── */}
      {error && (
        <div style={{
          padding: "16px 20px", borderRadius: 10,
          background: "#1a0a0a", border: "1px solid #e74c3c33",
          color: "#e74c3c", fontSize: 13,
        }}>
          ❌ {error}
        </div>
      )}

      {/* ── Empty ── */}
      {!loading && !error && filtered.length === 0 && (
        <div style={{
          textAlign: "center", padding: "80px 0",
          border: "1px dashed #1e293b", borderRadius: 12,
        }}>
          <div style={{ fontSize: 40, marginBottom: 12 }}>🔍</div>
          <p style={{ color: "#334155", fontSize: 14, margin: 0 }}>
            No templates match your search.
          </p>
        </div>
      )}

      {/* ── Grouped by runtime ── */}
      {!loading && !error && Object.keys(grouped).sort().map(runtime => {
        const c = getRuntimeColor(runtime)
        const group = grouped[runtime]

        return (
          <div key={runtime} style={{ marginBottom: 36 }}>

            {/* Runtime header */}
            <div style={{
              display: "flex", alignItems: "center", gap: 12,
              marginBottom: 14, paddingBottom: 12,
              borderBottom: `1px solid ${c.border}`,
            }}>
              <div style={{
                width: 36, height: 36, borderRadius: 8,
                background: c.bg, border: `1px solid ${c.border}`,
                display: "flex", alignItems: "center",
                justifyContent: "center", fontSize: 18,
              }}>
                {getRuntimeIcon(runtime)}
              </div>
              <div>
                <div style={{
                  fontSize: 16, fontWeight: 700, color: c.text,
                  textTransform: "capitalize",
                }}>
                  {runtime}
                </div>
                <div style={{ fontSize: 11, color: "#334155", marginTop: 1 }}>
                  {group.active.length} active · {group.deprecated.length} deprecated
                </div>
              </div>
            </div>

            {/* Version cards grid */}
            <div style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
              gap: 12,
            }}>
              {/* Active versions first */}
              {group.active.map(t => (
                <VersionCard key={t.id} t={t} formatTime={formatTime} />
              ))}
              {/* Deprecated versions after */}
              {group.deprecated.map(t => (
                <VersionCard key={t.id} t={t} formatTime={formatTime} />
              ))}
            </div>
          </div>
        )
      })}

      <style>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        input::placeholder { color: #2d3748; }
      `}</style>
    </div>
  )
}

/* ── Version card ── */
function VersionCard({ t, formatTime }) {
  const [expanded, setExpanded] = useState(false)
  const isActive = t.status === "active"
  const c = getRuntimeColor(t.runtime)

  return (
    <div style={{
      background: "#0f172a",
      border: `1px solid ${isActive ? "#1e293b" : "#e74c3c22"}`,
      borderRadius: 10, overflow: "hidden",
      opacity: isActive ? 1 : 0.7,
      transition: "border-color 0.15s, box-shadow 0.15s",
      position: "relative",
    }}
      onMouseEnter={e => {
        if (isActive) e.currentTarget.style.borderColor = c.border
        e.currentTarget.style.boxShadow = "0 4px 20px rgba(0,0,0,0.3)"
      }}
      onMouseLeave={e => {
        e.currentTarget.style.borderColor = isActive ? "#1e293b" : "#e74c3c22"
        e.currentTarget.style.boxShadow = "none"
      }}
    >
      {/* Deprecated ribbon */}
      {!isActive && (
        <div style={{
          position: "absolute", top: 10, right: -22,
          background: "#e74c3c", color: "#fff",
          fontSize: 9, fontWeight: 800, letterSpacing: "0.1em",
          padding: "3px 28px", transform: "rotate(45deg)",
          textTransform: "uppercase",
        }}>
          Deprecated
        </div>
      )}

      <div style={{ padding: "16px" }}>
        {/* Top row */}
        <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", marginBottom: 10 }}>
          <div>
            <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 4 }}>
              <span style={{ color: "#f1f5f9", fontWeight: 700, fontSize: 14 }}>{t.name}</span>
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
              <span style={{
                padding: "2px 8px", borderRadius: 4,
                background: c.bg, border: `1px solid ${c.border}`,
                color: c.text, fontSize: 11, fontFamily: "monospace", fontWeight: 700,
              }}>
                {t.version}
              </span>
              <span style={{
                padding: "2px 8px", borderRadius: 4,
                background: "#1e293b", border: "1px solid #334155",
                color: "#64748b", fontSize: 11,
              }}>
                {t.runtime}
              </span>
            </div>
          </div>

          {/* Status badge */}
          <span style={{
            padding: "3px 9px", borderRadius: 20, fontSize: 10, fontWeight: 700,
            background: isActive ? "#001a0f" : "#1a0a0a",
            border: `1px solid ${isActive ? "#10b98133" : "#e74c3c33"}`,
            color: isActive ? "#10b981" : "#e74c3c",
            letterSpacing: "0.06em", textTransform: "uppercase",
            flexShrink: 0,
          }}>
            {isActive ? "● Active" : "● Deprecated"}
          </span>
        </div>

        {/* Description */}
        {t.description && (
          <p style={{
            color: "#475569", fontSize: 12, margin: "0 0 12px",
            lineHeight: 1.5,
          }}>
            {t.description}
          </p>
        )}

        {/* Usage hint for active */}
        {isActive && (
          <div style={{
            background: "#060b12", border: "1px solid #1e293b",
            borderRadius: 6, padding: "8px 10px", marginBottom: 12,
          }}>
            <div style={{ fontSize: 10, color: "#334155", marginBottom: 4, fontWeight: 700, letterSpacing: "0.08em", textTransform: "uppercase" }}>
              Use in YAML
            </div>
            <code style={{ color: "#6366f1", fontSize: 11, fontFamily: "monospace" }}>
              runtime: {t.runtime}<br />
              templateVersion: {t.version}
            </code>
          </div>
        )}

        {/* Deprecated warning */}
        {!isActive && (
          <div style={{
            background: "#1a0a0a", border: "1px solid #e74c3c22",
            borderRadius: 6, padding: "8px 10px", marginBottom: 12,
            display: "flex", alignItems: "flex-start", gap: 8,
          }}>
            <span style={{ fontSize: 14, flexShrink: 0 }}>⚠️</span>
            <div style={{ fontSize: 11, color: "#e74c3c", lineHeight: 1.5 }}>
              This version is deprecated and <strong>cannot be used</strong> for new services.
              {t.deprecatedBy && (
                <span style={{ color: "#7f1d1d" }}>
                  {" "}Deprecated by <strong>{t.deprecatedBy}</strong> on {formatTime(t.deprecatedAt)}.
                </span>
              )}
            </div>
          </div>
        )}

        {/* Expand toggle */}
        <button
          onClick={() => setExpanded(v => !v)}
          style={{
            display: "flex", alignItems: "center", gap: 6,
            background: "none", border: "none",
            color: "#334155", cursor: "pointer", fontSize: 11,
            padding: 0, fontWeight: 600,
          }}
          onMouseEnter={e => e.currentTarget.style.color = "#64748b"}
          onMouseLeave={e => e.currentTarget.style.color = "#334155"}
        >
          <svg
            width="12" height="12" fill="none" stroke="currentColor"
            strokeWidth="2" viewBox="0 0 24 24"
            style={{ transform: expanded ? "rotate(90deg)" : "none", transition: "transform 0.15s" }}
          >
            <path d="M9 18l6-6-6-6"/>
          </svg>
          {expanded ? "Hide details" : "Show details"}
        </button>

        {/* Expanded details */}
        {expanded && (
          <div style={{
            marginTop: 12, paddingTop: 12,
            borderTop: "1px solid #1e293b",
            display: "flex", flexDirection: "column", gap: 8,
          }}>
            {t.changelog && (
              <div>
                <div style={{ fontSize: 10, fontWeight: 700, color: "#334155", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 3 }}>
                  Changelog
                </div>
                <div style={{ color: "#64748b", fontSize: 12, lineHeight: 1.5 }}>{t.changelog}</div>
              </div>
            )}
            <div>
              <div style={{ fontSize: 10, fontWeight: 700, color: "#334155", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 3 }}>
                Registered
              </div>
              <div style={{ color: "#475569", fontSize: 12 }}>
                {t.createdBy} · {formatTime(t.createdAt)}
              </div>
            </div>
            {t.releasedBy && (
              <div>
                <div style={{ fontSize: 10, fontWeight: 700, color: "#334155", letterSpacing: "0.08em", textTransform: "uppercase", marginBottom: 3 }}>
                  Last Re-released
                </div>
                <div style={{ color: "#10b981", fontSize: 12 }}>
                  {t.releasedBy} · {formatTime(t.releasedAt)}
                </div>
              </div>
            )}
            {!t.existsOnDisk && (
              <div style={{
                padding: "6px 10px", borderRadius: 6,
                background: "#1a0500", border: "1px solid #ef444433",
                color: "#ef4444", fontSize: 11,
              }}>
                ⚠️ Template files missing on disk — contact platform admin
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}