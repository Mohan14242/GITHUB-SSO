import { useState, useEffect, useCallback } from "react"
import { fetchAuditLogs } from "../api/auditApi"

const ACTION_META = {
  login:                      { color: "#0ea5e9", bg: "#001020", label: "Login",              icon: "🔐" },
  deployment_triggered:       { color: "#6366f1", bg: "#1a1040", label: "Deploy",             icon: "🚀" },
  deployment_approved:        { color: "#10b981", bg: "#001a0f", label: "Deploy Approved",    icon: "✅" },
  deployment_rejected:        { color: "#e74c3c", bg: "#1a0a0a", label: "Deploy Rejected",    icon: "❌" },
  service_creation_request:   { color: "#f59e0b", bg: "#1a1200", label: "Service Request",    icon: "📋" },
  service_creation_approved:  { color: "#10b981", bg: "#001a0f", label: "Service Approved",   icon: "✅" },
  service_creation_rejected:  { color: "#e74c3c", bg: "#1a0a0a", label: "Service Rejected",   icon: "❌" },
  rollback:                   { color: "#0ea5e9", bg: "#001020", label: "Rollback",            icon: "↩️" },
  artifact_registered:        { color: "#8b5cf6", bg: "#150a2a", label: "Artifact",           icon: "📦" },
}

const ACTION_OPTIONS = [
  { value: "",                           label: "All Actions"               },
  { value: "login",                      label: "Logins"                    },
  { value: "deployment_triggered",       label: "Deployments"               },
  { value: "deployment_approved",        label: "Deployment Approvals"      },
  { value: "deployment_rejected",        label: "Deployment Rejections"     },
  { value: "service_creation_request",   label: "Service Requests"          },
  { value: "service_creation_approved",  label: "Service Approvals"         },
  { value: "service_creation_rejected",  label: "Service Rejections"        },
  { value: "rollback",                   label: "Rollbacks"                 },
  { value: "artifact_registered",        label: "Artifacts"                 },
]

const ENV_OPTIONS    = ["", "dev", "test", "prod"]
const STATUS_OPTIONS = ["", "success", "pending", "rejected", "failed"]

const labelStyle = {
  display: "block", fontSize: 10, fontWeight: 700,
  color: "#334155", letterSpacing: "0.1em",
  textTransform: "uppercase", marginBottom: 6,
}
const inputStyle = {
  width: "100%", padding: "8px 10px",
  background: "#060b12", border: "1px solid #1e293b",
  borderRadius: 7, color: "#e2e8f0", fontSize: 13,
  outline: "none", transition: "border-color 0.15s",
  boxSizing: "border-box",
}
const selectStyle = { ...inputStyle, cursor: "pointer", appearance: "none" }

export default function AuditingPage() {
  const [logs, setLogs]           = useState([])
  const [total, setTotal]         = useState(0)
  const [page, setPage]           = useState(1)
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState("")
  const [expanded, setExpanded]   = useState(null)
  const [autoRefresh, setAutoRefresh] = useState(false)

  const [filters, setFilters] = useState({
    actor: "", environment: "", resourceName: "",
    action: "", status: "", from: "", to: "",
  })
  const [applied, setApplied] = useState({})

  const load = useCallback(async (f = applied, p = page) => {
    setLoading(true)
    setError("")
    try {
      const data = await fetchAuditLogs(f, p)
      setLogs(Array.isArray(data.logs) ? data.logs : [])
      setTotal(data.total ?? 0)
    } catch (err) {
      setError("Failed to load audit logs")
      console.error(err)
    } finally {
      setLoading(false)
    }
  }, [applied, page])

  useEffect(() => { load({}, 1) }, [])

  // Auto-refresh
  useEffect(() => {
    if (!autoRefresh) return
    const interval = setInterval(() => load(applied, page), 10000)
    return () => clearInterval(interval)
  }, [autoRefresh, applied, page])

  const applyFilters = () => {
    const clean = Object.fromEntries(Object.entries(filters).filter(([, v]) => v !== ""))
    setApplied(clean)
    setPage(1)
    load(clean, 1)
  }

  const clearFilters = () => {
    const empty = { actor: "", environment: "", resourceName: "", action: "", status: "", from: "", to: "" }
    setFilters(empty)
    setApplied({})
    setPage(1)
    load({}, 1)
  }

  const removeFilter = (key) => {
    const next = { ...applied }
    delete next[key]
    setFilters(f => ({ ...f, [key]: "" }))
    setApplied(next)
    setPage(1)
    load(next, 1)
  }

  const goToPage = (p) => {
    setPage(p)
    load(applied, p)
  }

  const totalPages = Math.ceil(total / 100)
  const activeFilterCount = Object.values(applied).filter(Boolean).length

  const setFilter = (key, val) => setFilters(f => ({ ...f, [key]: val }))

  const formatTime = (iso) => {
    try { return new Date(iso).toLocaleString() } catch { return iso }
  }

  const getMeta = (action) =>
    ACTION_META[action] ?? { color: "#64748b", bg: "#0f172a", label: action, icon: "●" }

  const exportCSV = () => {
    const header = ["ID", "Time", "Action", "Actor", "Service", "Environment", "Status", "Details", "IP"]
    const rows = logs.map(l => [
      l.id, l.createdAt, l.action, l.actor,
      l.resourceName, l.environment, l.status,
      `"${(l.details || "").replace(/"/g, "'")}"`, l.ipAddress,
    ])
    const csv = [header, ...rows].map(r => r.join(",")).join("\n")
    const blob = new Blob([csv], { type: "text/csv" })
    const url  = URL.createObjectURL(blob)
    const a    = document.createElement("a")
    a.href     = url
    a.download = `audit-logs-${new Date().toISOString().slice(0, 10)}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div style={{ padding: "40px 48px", maxWidth: 1300 }}>

      {/* ── Header ── */}
      <div style={{ marginBottom: 32 }}>
        <span style={{
          fontSize: 11, fontWeight: 700, letterSpacing: "0.15em",
          textTransform: "uppercase", color: "#6366f1", fontFamily: "monospace",
        }}>
          Observability
        </span>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginTop: 6 }}>
          <h2 style={{
            color: "#f1f5f9", margin: 0,
            fontFamily: "'Georgia', serif", fontSize: 28, fontWeight: 700,
          }}>
            Audit Logs
          </h2>
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>

            {/* Live toggle */}
            <button
              onClick={() => setAutoRefresh(v => !v)}
              style={{
                padding: "6px 14px", borderRadius: 7,
                border: `1px solid ${autoRefresh ? "#10b981" : "#1e293b"}`,
                background: autoRefresh ? "#001a0f" : "transparent",
                color: autoRefresh ? "#10b981" : "#475569",
                cursor: "pointer", fontSize: 12, fontWeight: 600,
                transition: "all 0.15s",
              }}
            >
              {autoRefresh ? "⏸ Live" : "▶ Live"}
            </button>

            {/* Export */}
            <button
              onClick={exportCSV}
              disabled={logs.length === 0}
              style={{
                padding: "6px 14px", borderRadius: 7,
                border: "1px solid #1e293b", background: "transparent",
                color: logs.length > 0 ? "#64748b" : "#1e293b",
                cursor: logs.length > 0 ? "pointer" : "not-allowed",
                fontSize: 12, fontWeight: 600,
              }}
              onMouseEnter={e => { if (logs.length > 0) { e.currentTarget.style.borderColor = "#6366f1"; e.currentTarget.style.color = "#6366f1" }}}
              onMouseLeave={e => { e.currentTarget.style.borderColor = "#1e293b"; e.currentTarget.style.color = "#64748b" }}
            >
              ⬇ Export CSV
            </button>

            {activeFilterCount > 0 && (
              <span style={{
                padding: "4px 10px", borderRadius: 20,
                background: "#6366f122", border: "1px solid #6366f144",
                color: "#6366f1", fontSize: 12, fontWeight: 600,
              }}>
                {activeFilterCount} filter{activeFilterCount > 1 ? "s" : ""} active
              </span>
            )}
            <span style={{
              padding: "4px 12px", borderRadius: 20,
              background: "#1e293b", border: "1px solid #334155",
              color: "#64748b", fontSize: 12, fontFamily: "monospace",
            }}>
              {total} records
            </span>
          </div>
        </div>
      </div>

      {/* ── Filters panel ── */}
      <div style={{
        background: "#0a0f1a", border: "1px solid #1e293b",
        borderRadius: 12, padding: "20px 24px", marginBottom: 16,
      }}>
        <div style={{
          fontSize: 11, fontWeight: 700, color: "#334155",
          letterSpacing: "0.1em", textTransform: "uppercase", marginBottom: 16,
        }}>
          Filters
        </div>

        <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 12, marginBottom: 12 }}>

          <div>
            <label style={labelStyle}>Username</label>
            <input
              value={filters.actor}
              onChange={e => setFilter("actor", e.target.value)}
              onKeyDown={e => e.key === "Enter" && applyFilters()}
              placeholder="e.g. Mohan14242"
              style={inputStyle}
              onFocus={e => e.target.style.borderColor = "#6366f1"}
              onBlur={e => e.target.style.borderColor = "#1e293b"}
            />
          </div>

          <div>
            <label style={labelStyle}>Service Name</label>
            <input
              value={filters.resourceName}
              onChange={e => setFilter("resourceName", e.target.value)}
              onKeyDown={e => e.key === "Enter" && applyFilters()}
              placeholder="e.g. orders"
              style={inputStyle}
              onFocus={e => e.target.style.borderColor = "#6366f1"}
              onBlur={e => e.target.style.borderColor = "#1e293b"}
            />
          </div>

          <div>
            <label style={labelStyle}>Action</label>
            <select value={filters.action} onChange={e => setFilter("action", e.target.value)} style={selectStyle}>
              {ACTION_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
          </div>

          <div>
            <label style={labelStyle}>Environment</label>
            <select value={filters.environment} onChange={e => setFilter("environment", e.target.value)} style={selectStyle}>
              {ENV_OPTIONS.map(e => <option key={e} value={e}>{e === "" ? "All Environments" : e}</option>)}
            </select>
          </div>

          <div>
            <label style={labelStyle}>Status</label>
            <select value={filters.status} onChange={e => setFilter("status", e.target.value)} style={selectStyle}>
              {STATUS_OPTIONS.map(s => <option key={s} value={s}>{s === "" ? "All Statuses" : s}</option>)}
            </select>
          </div>

          <div>
            <label style={labelStyle}>From Date</label>
            <input
              type="date" value={filters.from}
              onChange={e => setFilter("from", e.target.value)}
              style={{ ...inputStyle, colorScheme: "dark" }}
              onFocus={e => e.target.style.borderColor = "#6366f1"}
              onBlur={e => e.target.style.borderColor = "#1e293b"}
            />
          </div>

          <div>
            <label style={labelStyle}>To Date</label>
            <input
              type="date" value={filters.to}
              onChange={e => setFilter("to", e.target.value)}
              style={{ ...inputStyle, colorScheme: "dark" }}
              onFocus={e => e.target.style.borderColor = "#6366f1"}
              onBlur={e => e.target.style.borderColor = "#1e293b"}
            />
          </div>

          <div style={{ display: "flex", alignItems: "flex-end", gap: 8 }}>
            <button
              onClick={applyFilters}
              style={{
                flex: 1, padding: "9px 0", borderRadius: 7, border: "none",
                background: "linear-gradient(135deg, #6366f1, #4f46e5)",
                color: "#fff", cursor: "pointer", fontSize: 13, fontWeight: 700,
                boxShadow: "0 4px 14px rgba(99,102,241,0.3)",
              }}
              onMouseEnter={e => e.currentTarget.style.opacity = "0.9"}
              onMouseLeave={e => e.currentTarget.style.opacity = "1"}
            >
              Apply
            </button>
            <button
              onClick={clearFilters}
              style={{
                flex: 1, padding: "9px 0", borderRadius: 7,
                border: "1px solid #1e293b", background: "transparent",
                color: "#475569", cursor: "pointer", fontSize: 13,
              }}
              onMouseEnter={e => { e.currentTarget.style.borderColor = "#334155"; e.currentTarget.style.color = "#64748b" }}
              onMouseLeave={e => { e.currentTarget.style.borderColor = "#1e293b"; e.currentTarget.style.color = "#475569" }}
            >
              Clear
            </button>
          </div>
        </div>
      </div>

      {/* ── Active filter chips ── */}
      {activeFilterCount > 0 && (
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 16 }}>
          {Object.entries(applied).map(([key, val]) => (
            <span key={key} style={{
              display: "inline-flex", alignItems: "center", gap: 6,
              padding: "4px 10px", borderRadius: 20,
              background: "#1e293b", border: "1px solid #334155",
              color: "#94a3b8", fontSize: 12,
            }}>
              <span style={{ color: "#64748b", fontSize: 10, textTransform: "uppercase", fontWeight: 700 }}>
                {key}
              </span>
              {val}
              <span
                onClick={() => removeFilter(key)}
                style={{ cursor: "pointer", color: "#475569", marginLeft: 2, fontWeight: 700, fontSize: 14 }}
              >
                ×
              </span>
            </span>
          ))}
        </div>
      )}

      {/* ── Error ── */}
      {error && (
        <div style={{
          padding: "12px 16px", borderRadius: 8, marginBottom: 16,
          background: "#1a0a0a", border: "1px solid #e74c3c44",
          color: "#e74c3c", fontSize: 13,
        }}>
          ❌ {error}
        </div>
      )}

      {/* ── Loading ── */}
      {loading && (
        <div style={{ display: "flex", alignItems: "center", gap: 10, color: "#475569", padding: "20px 0" }}>
          <div style={{
            width: 16, height: 16, borderRadius: "50%",
            border: "2px solid #1e293b", borderTop: "2px solid #6366f1",
            animation: "spin 1s linear infinite",
          }}/>
          Loading audit logs…
        </div>
      )}

      {/* ── Empty ── */}
      {!loading && logs.length === 0 && (
        <div style={{
          textAlign: "center", padding: "80px 0",
          border: "1px dashed #1e293b", borderRadius: 12,
        }}>
          <div style={{ fontSize: 40, marginBottom: 12 }}>🔍</div>
          <p style={{ color: "#334155", fontSize: 14, margin: 0 }}>
            No audit logs found for the selected filters.
          </p>
        </div>
      )}

      {/* ── Table ── */}
      {!loading && logs.length > 0 && (
        <>
          <div style={{ border: "1px solid #1e293b", borderRadius: 12, overflow: "hidden" }}>

            {/* Header */}
            <div style={{
              display: "grid",
              gridTemplateColumns: "160px 130px 150px 110px 90px 1fr 170px",
              padding: "10px 16px",
              background: "#0a0f1a",
              borderBottom: "1px solid #1e293b",
            }}>
              {["Action", "Actor", "Service", "Environment", "Status", "Details", "Time"].map(h => (
                <span key={h} style={{
                  fontSize: 10, fontWeight: 700, color: "#334155",
                  letterSpacing: "0.1em", textTransform: "uppercase",
                }}>
                  {h}
                </span>
              ))}
            </div>

            {/* Rows */}
            {logs.map((log, i) => {
              const meta = getMeta(log.action)
              const isExpanded = expanded === log.id
              return (
                <div key={log.id}>
                  <div
                    onClick={() => setExpanded(isExpanded ? null : log.id)}
                    style={{
                      display: "grid",
                      gridTemplateColumns: "160px 130px 150px 110px 90px 1fr 170px",
                      padding: "13px 16px",
                      borderBottom: "1px solid #0f172a",
                      background: isExpanded ? "#0d1424" : "transparent",
                      cursor: "pointer",
                      transition: "background 0.12s",
                      alignItems: "center",
                    }}
                    onMouseEnter={e => { if (!isExpanded) e.currentTarget.style.background = "#0a0f1a" }}
                    onMouseLeave={e => { if (!isExpanded) e.currentTarget.style.background = "transparent" }}
                  >
                    {/* Action */}
                    <div>
                      <span style={{
                        display: "inline-flex", alignItems: "center", gap: 5,
                        padding: "3px 8px", borderRadius: 5,
                        background: meta.bg, border: `1px solid ${meta.color}33`,
                        color: meta.color, fontSize: 11, fontWeight: 600,
                      }}>
                        {meta.icon} {meta.label}
                      </span>
                    </div>

                    {/* Actor */}
                    <div style={{ display: "flex", alignItems: "center", gap: 7, fontSize: 12, color: "#94a3b8" }}>
                      <div style={{
                        width: 22, height: 22, borderRadius: "50%",
                        background: "#1e293b", display: "flex",
                        alignItems: "center", justifyContent: "center",
                        fontSize: 10, color: "#6366f1", fontWeight: 700, flexShrink: 0,
                      }}>
                        {log.actor?.[0]?.toUpperCase()}
                      </div>
                      <span style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", maxWidth: 90 }}>
                        {log.actor}
                      </span>
                    </div>

                    {/* Service */}
                    <div style={{
                      fontSize: 12, color: "#e2e8f0", fontFamily: "monospace",
                      overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                    }}>
                      {log.resourceName || "—"}
                    </div>

                    {/* Environment */}
                    <div>
                      {log.environment ? (
                        <span style={{
                          padding: "2px 8px", borderRadius: 4, fontSize: 11,
                          fontFamily: "monospace", fontWeight: 600,
                          background: log.environment === "prod" ? "#1a0030"
                            : log.environment === "test" ? "#1a1200" : "#001020",
                          border: `1px solid ${
                            log.environment === "prod" ? "#a855f733"
                            : log.environment === "test" ? "#f59e0b33" : "#0ea5e933"
                          }`,
                          color: log.environment === "prod" ? "#a855f7"
                            : log.environment === "test" ? "#f59e0b" : "#0ea5e9",
                        }}>
                          {log.environment}
                        </span>
                      ) : <span style={{ color: "#1e293b", fontSize: 12 }}>—</span>}
                    </div>

                    {/* Status */}
                    <div style={{
                      fontSize: 11, fontWeight: 700,
                      color: log.status === "success"  ? "#10b981"
                           : log.status === "rejected" ? "#e74c3c"
                           : log.status === "failed"   ? "#e74c3c"
                           : "#f59e0b",
                    }}>
                      {log.status === "success"  ? "✓ success"
                     : log.status === "rejected" ? "✗ rejected"
                     : log.status === "failed"   ? "✗ failed"
                     : "● pending"}
                    </div>

                    {/* Details */}
                    <div style={{
                      fontSize: 12, color: "#475569",
                      overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap",
                    }}>
                      {log.details || "—"}
                    </div>

                    {/* Time */}
                    <div style={{ fontSize: 11, color: "#334155", textAlign: "right" }}>
                      {formatTime(log.createdAt)}
                    </div>
                  </div>

                  {/* ── Expanded details row ── */}
                  {isExpanded && (
                    <div style={{
                      padding: "16px 20px",
                      background: "#060b12",
                      borderBottom: "1px solid #0f172a",
                      display: "grid",
                      gridTemplateColumns: "repeat(4, 1fr)",
                      gap: 20,
                    }}>
                      <div>
                        <div style={labelStyle}>Resource Type</div>
                        <div style={{ color: "#64748b", fontSize: 12 }}>{log.resourceType}</div>
                      </div>
                      <div>
                        <div style={labelStyle}>IP Address</div>
                        <div style={{ color: "#64748b", fontFamily: "monospace", fontSize: 12 }}>
                          {log.ipAddress || "—"}
                        </div>
                      </div>
                      <div>
                        <div style={labelStyle}>Log ID</div>
                        <div style={{ color: "#334155", fontFamily: "monospace", fontSize: 12 }}>
                          #{log.id}
                        </div>
                      </div>
                      <div>
                        <div style={labelStyle}>Full Details</div>
                        <div style={{ color: "#94a3b8", fontSize: 12, wordBreak: "break-word" }}>
                          {log.details || "—"}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>

          {/* ── Pagination ── */}
          {totalPages > 1 && (
            <div style={{
              display: "flex", alignItems: "center", justifyContent: "center",
              gap: 8, marginTop: 20,
            }}>
              <button
                onClick={() => goToPage(page - 1)}
                disabled={page === 1}
                style={{
                  padding: "7px 14px", borderRadius: 7,
                  border: "1px solid #1e293b", background: "transparent",
                  color: page === 1 ? "#1e293b" : "#64748b",
                  cursor: page === 1 ? "not-allowed" : "pointer", fontSize: 13,
                }}
              >
                ← Prev
              </button>

              {Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
                const p = i + 1
                return (
                  <button
                    key={p}
                    onClick={() => goToPage(p)}
                    style={{
                      padding: "7px 12px", borderRadius: 7,
                      border: `1px solid ${p === page ? "#6366f1" : "#1e293b"}`,
                      background: p === page ? "#6366f122" : "transparent",
                      color: p === page ? "#6366f1" : "#475569",
                      cursor: "pointer", fontSize: 13, fontWeight: p === page ? 700 : 400,
                    }}
                  >
                    {p}
                  </button>
                )
              })}

              <button
                onClick={() => goToPage(page + 1)}
                disabled={page === totalPages}
                style={{
                  padding: "7px 14px", borderRadius: 7,
                  border: "1px solid #1e293b", background: "transparent",
                  color: page === totalPages ? "#1e293b" : "#64748b",
                  cursor: page === totalPages ? "not-allowed" : "pointer", fontSize: 13,
                }}
              >
                Next →
              </button>

              <span style={{ color: "#334155", fontSize: 12, marginLeft: 8 }}>
                Page {page} of {totalPages} · {total} total records
              </span>
            </div>
          )}
        </>
      )}

      <style>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        input::placeholder { color: #2d3748; }
        select option { background: #0f172a; color: #e2e8f0; }
      `}</style>
    </div>
  )
}

