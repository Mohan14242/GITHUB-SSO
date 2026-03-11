import { useState, useEffect } from "react"
import { useAuth } from "../auth/AuthContext"
import {
  fetchTemplateVersions,
  createTemplateVersion,
  deprecateTemplateVersion,
  releaseTemplateVersion,
} from "../api/templatesApi"

const RUNTIME_OPTIONS = ["go", "node", "python", "java", "rust"]

const inputStyle = {
  width: "100%", padding: "9px 12px",
  background: "#060b12", border: "1px solid #1e293b",
  borderRadius: 7, color: "#e2e8f0", fontSize: 13,
  outline: "none", transition: "border-color 0.15s",
  boxSizing: "border-box",
}
const labelStyle = {
  display: "block", fontSize: 10, fontWeight: 700,
  color: "#334155", letterSpacing: "0.1em",
  textTransform: "uppercase", marginBottom: 6,
}

/* ── Toast ── */
function Toast({ message, type, onClose }) {
  useEffect(() => {
    const t = setTimeout(onClose, 3500)
    return () => clearTimeout(t)
  }, [])
  return (
    <div style={{
      position: "fixed", bottom: 28, right: 28, zIndex: 9999,
      padding: "12px 20px", borderRadius: 10,
      background: type === "success" ? "#001a0f" : "#1a0a0a",
      border: `1px solid ${type === "success" ? "#10b98155" : "#e74c3c55"}`,
      color: type === "success" ? "#10b981" : "#e74c3c",
      fontSize: 13, fontWeight: 600,
      boxShadow: "0 8px 32px rgba(0,0,0,0.4)",
      display: "flex", alignItems: "center", gap: 10,
      maxWidth: 480,
    }}>
      {type === "success" ? "✅" : "❌"} {message}
    </div>
  )
}

/* ── Template Card ── */
function TemplateCard({ t, isAdmin, onDeprecate, onRelease, formatTime }) {
  const [expanded, setExpanded] = useState(false)
  const isActive = t.status === "active"

  return (
    <div style={{
      background: "#0f172a",
      border: `1px solid ${isActive ? "#1e293b" : "#e74c3c22"}`,
      borderRadius: 10,
      overflow: "hidden",
      transition: "border-color 0.15s",
    }}>
      {/* Main row */}
      <div
        onClick={() => setExpanded(v => !v)}
        style={{
          padding: "16px 20px",
          display: "flex", alignItems: "center",
          justifyContent: "space-between",
          cursor: "pointer",
        }}
        onMouseEnter={e => e.currentTarget.style.background = "#0d1424"}
        onMouseLeave={e => e.currentTarget.style.background = "transparent"}
      >
        {/* Left */}
        <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
          <div style={{
            width: 38, height: 38, borderRadius: 8,
            background: isActive ? "#1e293b" : "#1a0a0a",
            display: "flex", alignItems: "center",
            justifyContent: "center", fontSize: 18, flexShrink: 0,
          }}>
            📦
          </div>
          <div>
            <div style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
              <strong style={{ color: "#f1f5f9", fontSize: 15 }}>{t.name}</strong>
              <span style={{
                padding: "2px 8px", borderRadius: 4,
                background: "#1e293b", border: "1px solid #334155",
                color: "#6366f1", fontSize: 11, fontFamily: "monospace", fontWeight: 700,
              }}>
                {t.version}
              </span>
              <span style={{
                padding: "2px 8px", borderRadius: 4,
                background: "#0f172a", border: "1px solid #334155",
                color: "#64748b", fontSize: 11, fontFamily: "monospace",
              }}>
                {t.runtime}
              </span>
            </div>
            {t.description && (
              <div style={{ color: "#475569", fontSize: 12, marginTop: 4 }}>
                {t.description}
              </div>
            )}
          </div>
        </div>

        {/* Right */}
        <div style={{ display: "flex", alignItems: "center", gap: 10, flexShrink: 0 }}>
          <span style={{
            padding: "3px 10px", borderRadius: 20, fontSize: 11, fontWeight: 700,
            background: isActive ? "#001a0f" : "#1a0a0a",
            border: `1px solid ${isActive ? "#10b98133" : "#e74c3c33"}`,
            color: isActive ? "#10b981" : "#e74c3c",
            letterSpacing: "0.06em", textTransform: "uppercase",
          }}>
            {isActive ? "● Active" : "● Deprecated"}
          </span>

          {isAdmin && isActive && (
            <button
              onClick={e => { e.stopPropagation(); onDeprecate() }}
              style={{
                padding: "6px 14px", borderRadius: 7,
                border: "1px solid #e74c3c33",
                background: "transparent", color: "#e74c3c",
                cursor: "pointer", fontSize: 12, fontWeight: 600,
                transition: "background 0.15s",
              }}
              onMouseEnter={e => e.currentTarget.style.background = "#1a0a0a"}
              onMouseLeave={e => e.currentTarget.style.background = "transparent"}
            >
              Deprecate
            </button>
          )}
          {isAdmin && !isActive && (
            <button
              onClick={e => { e.stopPropagation(); onRelease() }}
              style={{
                padding: "6px 14px", borderRadius: 7,
                border: "1px solid #10b98133",
                background: "transparent", color: "#10b981",
                cursor: "pointer", fontSize: 12, fontWeight: 600,
                transition: "background 0.15s",
              }}
              onMouseEnter={e => e.currentTarget.style.background = "#001a0f"}
              onMouseLeave={e => e.currentTarget.style.background = "transparent"}
            >
              Re-release
            </button>
          )}

          <svg
            width="14" height="14" fill="none" stroke="#334155"
            strokeWidth="2" viewBox="0 0 24 24"
            style={{
              transform: expanded ? "rotate(90deg)" : "none",
              transition: "transform 0.15s", flexShrink: 0,
            }}
          >
            <path d="M9 18l6-6-6-6"/>
          </svg>
        </div>
      </div>

      {/* Expanded details */}
      {expanded && (
        <div style={{
          borderTop: "1px solid #0f172a",
          padding: "16px 20px",
          background: "#060b12",
          display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 20,
        }}>
          <div>
            <div style={labelStyle}>Created By</div>
            <div style={{ color: "#94a3b8", fontSize: 12 }}>{t.createdBy}</div>
            <div style={{ color: "#334155", fontSize: 11, marginTop: 2 }}>
              {formatTime(t.createdAt)}
            </div>
          </div>
          {t.changelog && (
            <div>
              <div style={labelStyle}>Changelog</div>
              <div style={{ color: "#64748b", fontSize: 12, lineHeight: 1.6 }}>
                {t.changelog}
              </div>
            </div>
          )}
          {!isActive && t.deprecatedBy && (
            <div>
              <div style={labelStyle}>Deprecated By</div>
              <div style={{ color: "#e74c3c", fontSize: 12 }}>{t.deprecatedBy}</div>
              <div style={{ color: "#334155", fontSize: 11, marginTop: 2 }}>
                {formatTime(t.deprecatedAt)}
              </div>
            </div>
          )}
          {t.releasedBy && (
            <div>
              <div style={labelStyle}>Last Re-released By</div>
              <div style={{ color: "#10b981", fontSize: 12 }}>{t.releasedBy}</div>
              <div style={{ color: "#334155", fontSize: 11, marginTop: 2 }}>
                {formatTime(t.releasedAt)}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

/* ── Main Page ── */
export default function TemplateVersionsPage() {
  const { hasRole } = useAuth()
  const isAdmin = hasRole("admin")

  const [versions, setVersions]         = useState([])
  const [loading, setLoading]           = useState(true)
  const [toast, setToast]               = useState(null)
  const [filterStatus, setFilterStatus] = useState("")
  const [filterRuntime, setFilterRuntime] = useState("")
  const [confirmModal, setConfirmModal] = useState(null)
  const [showCreate, setShowCreate]     = useState(false)
  const [creating, setCreating]         = useState(false)

  const [form, setForm] = useState({
    name: "", version: "", runtime: "go",
    description: "", changelog: "",
  })

  useEffect(() => { load() }, [filterStatus, filterRuntime])

  async function load() {
    setLoading(true)
    try {
      const data = await fetchTemplateVersions({
        status: filterStatus,
        runtime: filterRuntime,
      })
      setVersions(Array.isArray(data) ? data : [])
    } catch (err) {
      showToast("Failed to load template versions", "error")
    } finally {
      setLoading(false)
    }
  }

  function showToast(message, type = "success") {
    setToast({ message, type })
  }

  async function handleDeprecate() {
    const { id, name, version } = confirmModal
    setConfirmModal(null)
    try {
      await deprecateTemplateVersion(id)
      showToast(`${name}@${version} deprecated successfully`)
      load()
    } catch (err) {
      showToast(err.message, "error")
    }
  }

  async function handleRelease() {
    const { id, name, version } = confirmModal
    setConfirmModal(null)
    try {
      await releaseTemplateVersion(id)
      showToast(`${name}@${version} re-released successfully`)
      load()
    } catch (err) {
      showToast(err.message, "error")
    }
  }

  async function handleCreate() {
    if (!form.name || !form.version || !form.runtime) {
      showToast("name, version and runtime are required", "error")
      return
    }
    setCreating(true)
    try {
      await createTemplateVersion(form)
      showToast(`${form.name}@${form.version} created successfully`)
      setShowCreate(false)
      setForm({ name: "", version: "", runtime: "go", description: "", changelog: "" })
      load()
    } catch (err) {
      showToast(err.message, "error")
    } finally {
      setCreating(false)
    }
  }

  const active     = versions.filter(v => v.status === "active")
  const deprecated = versions.filter(v => v.status === "deprecated")

  const formatTime = (iso) => {
    if (!iso) return "—"
    try { return new Date(iso).toLocaleString() } catch { return iso }
  }

  return (
    <div style={{ padding: "40px 48px", maxWidth: 1100 }}>

      {/* ── Header ── */}
      <div style={{ marginBottom: 32 }}>
        <span style={{
          fontSize: 11, fontWeight: 700, letterSpacing: "0.15em",
          textTransform: "uppercase", color: "#6366f1", fontFamily: "monospace",
        }}>
          Golden Path
        </span>
        <div style={{
          display: "flex", alignItems: "center",
          justifyContent: "space-between", marginTop: 6,
        }}>
          <h2 style={{
            color: "#f1f5f9", margin: 0,
            fontFamily: "'Georgia', serif", fontSize: 28, fontWeight: 700,
          }}>
            Template Versions
          </h2>
          <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
            <span style={{
              padding: "4px 12px", borderRadius: 20,
              background: "#001a0f", border: "1px solid #10b98133",
              color: "#10b981", fontSize: 12, fontWeight: 600,
            }}>
              {active.length} active
            </span>
            <span style={{
              padding: "4px 12px", borderRadius: 20,
              background: "#1a0a0a", border: "1px solid #e74c3c33",
              color: "#e74c3c", fontSize: 12, fontWeight: 600,
            }}>
              {deprecated.length} deprecated
            </span>
            {isAdmin && (
              <button
                onClick={() => setShowCreate(v => !v)}
                style={{
                  padding: "8px 18px", borderRadius: 8, border: "none",
                  background: showCreate
                    ? "#1e293b"
                    : "linear-gradient(135deg, #6366f1, #4f46e5)",
                  color: showCreate ? "#64748b" : "#fff",
                  cursor: "pointer", fontSize: 13, fontWeight: 700,
                  boxShadow: showCreate ? "none" : "0 4px 14px rgba(99,102,241,0.3)",
                  display: "flex", alignItems: "center", gap: 7,
                  transition: "all 0.15s",
                }}
              >
                <svg width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                  <circle cx="12" cy="12" r="10"/>
                  <path d="M12 8v8M8 12h8"/>
                </svg>
                {showCreate ? "Cancel" : "New Version"}
              </button>
            )}
          </div>
        </div>
      </div>

      {/* ── Create form ── */}
      {showCreate && isAdmin && (
        <div style={{
          background: "#0a0f1a",
          border: "1px solid #6366f133",
          borderRadius: 12, padding: "24px", marginBottom: 28,
        }}>
          <div style={{
            fontSize: 11, fontWeight: 700, color: "#6366f1",
            letterSpacing: "0.1em", textTransform: "uppercase", marginBottom: 20,
          }}>
            ✦ New Template Version
          </div>

          <div style={{
            display: "grid", gridTemplateColumns: "1fr 1fr 1fr",
            gap: 16, marginBottom: 16,
          }}>
            <div>
              <label style={labelStyle}>Template Name</label>
              <input
                value={form.name}
                onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                placeholder="e.g. microservice"
                style={inputStyle}
                onFocus={e => e.target.style.borderColor = "#6366f1"}
                onBlur={e => e.target.style.borderColor = "#1e293b"}
              />
            </div>
            <div>
              <label style={labelStyle}>Version</label>
              <input
                value={form.version}
                onChange={e => setForm(f => ({ ...f, version: e.target.value }))}
                placeholder="e.g. v2"
                style={inputStyle}
                onFocus={e => e.target.style.borderColor = "#6366f1"}
                onBlur={e => e.target.style.borderColor = "#1e293b"}
              />
            </div>
            <div>
              <label style={labelStyle}>Runtime</label>
              <select
                value={form.runtime}
                onChange={e => setForm(f => ({ ...f, runtime: e.target.value }))}
                style={{ ...inputStyle, cursor: "pointer", appearance: "none" }}
              >
                {RUNTIME_OPTIONS.map(r => (
                  <option key={r} value={r}>{r}</option>
                ))}
              </select>
            </div>
          </div>

          <div style={{
            display: "grid", gridTemplateColumns: "1fr 1fr",
            gap: 16, marginBottom: 20,
          }}>
            <div>
              <label style={labelStyle}>Description</label>
              <textarea
                value={form.description}
                onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
                placeholder="What does this template include?"
                rows={3}
                style={{ ...inputStyle, resize: "vertical", lineHeight: 1.5 }}
                onFocus={e => e.target.style.borderColor = "#6366f1"}
                onBlur={e => e.target.style.borderColor = "#1e293b"}
              />
            </div>
            <div>
              <label style={labelStyle}>Changelog</label>
              <textarea
                value={form.changelog}
                onChange={e => setForm(f => ({ ...f, changelog: e.target.value }))}
                placeholder="What changed from the previous version?"
                rows={3}
                style={{ ...inputStyle, resize: "vertical", lineHeight: 1.5 }}
                onFocus={e => e.target.style.borderColor = "#6366f1"}
                onBlur={e => e.target.style.borderColor = "#1e293b"}
              />
            </div>
          </div>

          <div style={{ display: "flex", gap: 10 }}>
            <button
              onClick={handleCreate}
              disabled={creating}
              style={{
                padding: "10px 28px", borderRadius: 8, border: "none",
                background: creating
                  ? "#1e293b"
                  : "linear-gradient(135deg, #6366f1, #4f46e5)",
                color: creating ? "#475569" : "#fff",
                cursor: creating ? "not-allowed" : "pointer",
                fontSize: 13, fontWeight: 700,
                boxShadow: creating ? "none" : "0 4px 14px rgba(99,102,241,0.3)",
              }}
            >
              {creating ? "Creating…" : "Create Version"}
            </button>
            <button
              onClick={() => {
                setShowCreate(false)
                setForm({ name: "", version: "", runtime: "go", description: "", changelog: "" })
              }}
              style={{
                padding: "10px 20px", borderRadius: 8,
                border: "1px solid #1e293b", background: "transparent",
                color: "#475569", cursor: "pointer", fontSize: 13,
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* ── Filters ── */}
      <div style={{ display: "flex", gap: 12, alignItems: "center", marginBottom: 24 }}>
        <select
          value={filterStatus}
          onChange={e => setFilterStatus(e.target.value)}
          style={{
            padding: "8px 12px", borderRadius: 7,
            background: "#0a0f1a", border: "1px solid #1e293b",
            color: "#94a3b8", fontSize: 13, cursor: "pointer",
            outline: "none", appearance: "none",
          }}
        >
          <option value="">All Statuses</option>
          <option value="active">Active</option>
          <option value="deprecated">Deprecated</option>
        </select>

        <select
          value={filterRuntime}
          onChange={e => setFilterRuntime(e.target.value)}
          style={{
            padding: "8px 12px", borderRadius: 7,
            background: "#0a0f1a", border: "1px solid #1e293b",
            color: "#94a3b8", fontSize: 13, cursor: "pointer",
            outline: "none", appearance: "none",
          }}
        >
          <option value="">All Runtimes</option>
          {RUNTIME_OPTIONS.map(r => (
            <option key={r} value={r}>{r}</option>
          ))}
        </select>

        <button
          onClick={load}
          style={{
            padding: "8px 16px", borderRadius: 7,
            border: "1px solid #1e293b", background: "transparent",
            color: "#64748b", cursor: "pointer", fontSize: 13,
          }}
          onMouseEnter={e => { e.currentTarget.style.borderColor = "#6366f1"; e.currentTarget.style.color = "#6366f1" }}
          onMouseLeave={e => { e.currentTarget.style.borderColor = "#1e293b"; e.currentTarget.style.color = "#64748b" }}
        >
          ↺ Refresh
        </button>
      </div>

      {/* ── Loading ── */}
      {loading && (
        <div style={{ display: "flex", alignItems: "center", gap: 10, color: "#475569", padding: "20px 0" }}>
          <div style={{
            width: 16, height: 16, borderRadius: "50%",
            border: "2px solid #1e293b", borderTop: "2px solid #6366f1",
            animation: "spin 1s linear infinite",
          }}/>
          Loading templates…
        </div>
      )}

      {/* ── Empty ── */}
      {!loading && versions.length === 0 && (
        <div style={{
          textAlign: "center", padding: "80px 0",
          border: "1px dashed #1e293b", borderRadius: 12,
        }}>
          <div style={{ fontSize: 40, marginBottom: 12 }}>📦</div>
          <p style={{ color: "#334155", fontSize: 14, margin: 0 }}>
            No template versions found.
            {isAdmin && " Click 'New Version' to add one."}
          </p>
        </div>
      )}

      {/* ── Active versions ── */}
      {!loading && active.length > 0 && (
        <div style={{ marginBottom: 36 }}>
          <div style={{
            fontSize: 11, fontWeight: 700, color: "#10b981",
            letterSpacing: "0.12em", textTransform: "uppercase",
            marginBottom: 14, display: "flex", alignItems: "center", gap: 8,
          }}>
            <span style={{
              display: "inline-block", width: 8, height: 8,
              borderRadius: "50%", background: "#10b981",
            }}/>
            Active Versions
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            {active.map(t => (
              <TemplateCard
                key={t.id}
                t={t}
                isAdmin={isAdmin}
                onDeprecate={() => setConfirmModal({
                  id: t.id, name: t.name,
                  version: t.version, action: "deprecate",
                })}
                formatTime={formatTime}
              />
            ))}
          </div>
        </div>
      )}

      {/* ── Deprecated versions ── */}
      {!loading && deprecated.length > 0 && (
        <div>
          <div style={{
            fontSize: 11, fontWeight: 700, color: "#e74c3c",
            letterSpacing: "0.12em", textTransform: "uppercase",
            marginBottom: 14, display: "flex", alignItems: "center", gap: 8,
          }}>
            <span style={{
              display: "inline-block", width: 8, height: 8,
              borderRadius: "50%", background: "#e74c3c",
            }}/>
            Deprecated Versions
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
            {deprecated.map(t => (
              <TemplateCard
                key={t.id}
                t={t}
                isAdmin={isAdmin}
                onRelease={() => setConfirmModal({
                  id: t.id, name: t.name,
                  version: t.version, action: "release",
                })}
                formatTime={formatTime}
              />
            ))}
          </div>
        </div>
      )}

      {/* ── Confirm modal ── */}
      {confirmModal && (
        <div style={{
          position: "fixed", inset: 0, zIndex: 1000,
          background: "rgba(0,0,0,0.75)", backdropFilter: "blur(4px)",
          display: "flex", alignItems: "center", justifyContent: "center",
        }}>
          <div style={{
            background: "#0f172a", border: "1px solid #1e293b",
            borderRadius: 14, padding: "32px", width: 420,
            boxShadow: "0 24px 64px rgba(0,0,0,0.5)",
          }}>
            <div style={{ fontSize: 36, marginBottom: 16, textAlign: "center" }}>
              {confirmModal.action === "deprecate" ? "⚠️" : "♻️"}
            </div>
            <h3 style={{
              color: "#f1f5f9", textAlign: "center",
              margin: "0 0 12px", fontSize: 18,
            }}>
              {confirmModal.action === "deprecate"
                ? "Deprecate Version"
                : "Re-release Version"}
            </h3>
            <p style={{
              color: "#64748b", textAlign: "center",
              fontSize: 13, margin: "0 0 8px", lineHeight: 1.6,
            }}>
              {confirmModal.action === "deprecate" ? (
                <>
                  Are you sure you want to deprecate{" "}
                  <strong style={{ color: "#f1f5f9" }}>
                    {confirmModal.name}@{confirmModal.version}
                  </strong>?
                  <br/>New services won't be able to use this template.
                </>
              ) : (
                <>
                  Re-release{" "}
                  <strong style={{ color: "#f1f5f9" }}>
                    {confirmModal.name}@{confirmModal.version}
                  </strong>?
                  <br/>It will become available for new service creation again.
                </>
              )}
            </p>
            <p style={{
              color: "#334155", textAlign: "center",
              fontSize: 11, margin: "0 0 24px",
              fontFamily: "monospace",
            }}>
              ℹ️ The template folder must exist on disk to proceed.
            </p>
            <div style={{ display: "flex", gap: 10 }}>
              <button
                onClick={confirmModal.action === "deprecate"
                  ? handleDeprecate
                  : handleRelease}
                style={{
                  flex: 1, padding: "11px 0", borderRadius: 8, border: "none",
                  background: confirmModal.action === "deprecate"
                    ? "linear-gradient(135deg, #e74c3c, #c0392b)"
                    : "linear-gradient(135deg, #10b981, #059669)",
                  color: "#fff", cursor: "pointer",
                  fontSize: 14, fontWeight: 700,
                }}
              >
                {confirmModal.action === "deprecate"
                  ? "Yes, Deprecate"
                  : "Yes, Re-release"}
              </button>
              <button
                onClick={() => setConfirmModal(null)}
                style={{
                  flex: 1, padding: "11px 0", borderRadius: 8,
                  border: "1px solid #1e293b", background: "transparent",
                  color: "#475569", cursor: "pointer", fontSize: 14,
                }}
                onMouseEnter={e => { e.currentTarget.style.borderColor = "#334155"; e.currentTarget.style.color = "#64748b" }}
                onMouseLeave={e => { e.currentTarget.style.borderColor = "#1e293b"; e.currentTarget.style.color = "#475569" }}
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {toast && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast(null)}
        />
      )}

      <style>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        select option { background: #0f172a; color: #e2e8f0; }
        textarea::placeholder, input::placeholder { color: #2d3748; }
      `}</style>
    </div>
  )
}