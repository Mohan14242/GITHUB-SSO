import { useEffect, useState } from "react"
import {
  fetchServiceCreationRequests,
  approveServiceCreation,
  rejectServiceCreation,
} from "../api/serviceCreationApi"

const STATUS_STYLE = {
  pending:  { color: "#f59e0b", bg: "#1a1400", border: "#f59e0b33", label: "Pending"  },
  approved: { color: "#10b981", bg: "#001a0f", border: "#10b98133", label: "Approved" },
  rejected: { color: "#e74c3c", bg: "#1a0a0a", border: "#e74c3c33", label: "Rejected" },
}

function StatusBadge({ status }) {
  const s = STATUS_STYLE[status] ?? STATUS_STYLE.pending
  return (
    <span style={{
      padding: "3px 10px", borderRadius: 20,
      background: s.bg, border: `1px solid ${s.border}`,
      color: s.color, fontSize: 11, fontWeight: 700,
      letterSpacing: "0.05em", textTransform: "uppercase",
    }}>
      ● {s.label}
    </span>
  )
}

function YAMLPreview({ yaml }) {
  const [open, setOpen] = useState(false)
  return (
    <div style={{ marginTop: 10 }}>
      <button
        onClick={() => setOpen(o => !o)}
        style={{
          background: "none", border: "1px solid #1e293b",
          borderRadius: 5, color: "#475569", cursor: "pointer",
          fontSize: 11, padding: "3px 10px",
          display: "flex", alignItems: "center", gap: 5,
        }}
        onMouseEnter={e => { e.currentTarget.style.borderColor = "#334155"; e.currentTarget.style.color = "#64748b" }}
        onMouseLeave={e => { e.currentTarget.style.borderColor = "#1e293b"; e.currentTarget.style.color = "#475569" }}
      >
        <svg width="10" height="10" fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
          <path d={open ? "M19 9l-7 7-7-7" : "M9 18l6-6-6-6"}/>
        </svg>
        {open ? "Hide" : "View"} YAML
      </button>
      {open && (
        <pre style={{
          marginTop: 8, padding: "12px 14px",
          background: "#080d14", border: "1px solid #1e293b",
          borderRadius: 8, color: "#94a3b8",
          fontSize: 12, fontFamily: "monospace",
          overflowX: "auto", lineHeight: 1.6,
          maxHeight: 240, overflowY: "auto",
        }}>
          {yaml}
        </pre>
      )}
    </div>
  )
}

function RejectModal({ onConfirm, onCancel, loading }) {
  const [reason, setReason] = useState("")
  return (
    <div style={{
      position: "fixed", inset: 0, zIndex: 100,
      background: "rgba(0,0,0,0.7)", backdropFilter: "blur(4px)",
      display: "flex", alignItems: "center", justifyContent: "center",
    }}>
      <div style={{
        background: "#0f172a", border: "1px solid #1e293b",
        borderRadius: 14, padding: 28, width: 420, maxWidth: "90vw",
      }}>
        <h3 style={{ color: "#f1f5f9", margin: "0 0 6px", fontFamily: "'Georgia', serif" }}>
          Reject Request
        </h3>
        <p style={{ color: "#475569", fontSize: 13, margin: "0 0 18px" }}>
          Optionally provide a reason — the requester will see this.
        </p>
        <textarea
          value={reason}
          onChange={e => setReason(e.target.value)}
          placeholder="e.g. Service name conflicts with existing service..."
          rows={3}
          style={{
            width: "100%", padding: "10px 12px",
            background: "#080d14", border: "1px solid #1e293b",
            borderRadius: 8, color: "#e2e8f0",
            fontSize: 13, fontFamily: "monospace",
            resize: "vertical", outline: "none",
            boxSizing: "border-box",
          }}
        />
        <div style={{ display: "flex", gap: 10, marginTop: 16, justifyContent: "flex-end" }}>
          <button
            onClick={onCancel}
            style={{
              padding: "9px 20px", borderRadius: 7,
              border: "1px solid #1e293b", background: "transparent",
              color: "#64748b", cursor: "pointer", fontSize: 13,
            }}
          >
            Cancel
          </button>
          <button
            onClick={() => onConfirm(reason)}
            disabled={loading}
            style={{
              padding: "9px 20px", borderRadius: 7,
              border: "none", background: "#e74c3c",
              color: "#fff", cursor: loading ? "not-allowed" : "pointer",
              fontSize: 13, fontWeight: 700, opacity: loading ? 0.7 : 1,
            }}
          >
            {loading ? "Rejecting…" : "Confirm Reject"}
          </button>
        </div>
      </div>
    </div>
  )
}

export default function ServiceCreationApprovals() {
  const [requests, setRequests]       = useState([])
  const [loading, setLoading]         = useState(true)
  const [actionLoading, setActionLoading] = useState({})
  const [error, setError]             = useState("")
  const [toast, setToast]             = useState("")
  const [rejectTarget, setRejectTarget] = useState(null)
  const [tab, setTab]                 = useState("pending")

  useEffect(() => { load() }, [])

  async function load() {
    try {
      setLoading(true)
      setError("")
      const data = await fetchServiceCreationRequests()
      setRequests(Array.isArray(data) ? data : [])
    } catch (err) {
      setError("Failed to load requests")
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  function showToast(msg) {
    setToast(msg)
    setTimeout(() => setToast(""), 3500)
  }

  async function handleApprove(id, serviceName) {
    setActionLoading(p => ({ ...p, [id]: "approving" }))
    try {
      await approveServiceCreation(id)
      showToast(`✅ "${serviceName}" created successfully`)
      await load()
    } catch (err) {
      showToast(`❌ ${err.message}`)
    } finally {
      setActionLoading(p => ({ ...p, [id]: null }))
    }
  }

  async function handleReject(id, reason) {
    setActionLoading(p => ({ ...p, [id]: "rejecting" }))
    setRejectTarget(null)
    try {
      await rejectServiceCreation(id, reason)
      showToast(`Request rejected`)
      await load()
    } catch (err) {
      showToast(`❌ ${err.message}`)
    } finally {
      setActionLoading(p => ({ ...p, [id]: null }))
    }
  }

  const pending  = requests.filter(r => r.status === "pending")
  const history  = requests.filter(r => r.status !== "pending")
  const displayed = tab === "pending" ? pending : history

  return (
    <div style={{ padding: "40px 48px", maxWidth: 900 }}>

      {/* Header */}
      <div style={{ marginBottom: 32 }}>
        <span style={{
          fontSize: 11, fontWeight: 700, letterSpacing: "0.15em",
          textTransform: "uppercase", color: "#6366f1", fontFamily: "monospace",
        }}>
          Admin Review
        </span>
        <h2 style={{
          color: "#f1f5f9", margin: "6px 0 6px",
          fontFamily: "'Georgia', serif", fontSize: 28,
        }}>
          Service Creation Approvals
        </h2>
        <p style={{ color: "#475569", fontSize: 14 }}>
          Review and approve or reject service provisioning requests from developers.
        </p>
      </div>

      {/* Tabs */}
      <div style={{ display: "flex", gap: 4, marginBottom: 24, borderBottom: "1px solid #1e293b", paddingBottom: 0 }}>
        {[
          { key: "pending", label: "Pending", count: pending.length },
          { key: "history", label: "History", count: history.length },
        ].map(t => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            style={{
              padding: "10px 20px",
              background: "none", border: "none",
              borderBottom: tab === t.key ? "2px solid #6366f1" : "2px solid transparent",
              color: tab === t.key ? "#f1f5f9" : "#475569",
              cursor: "pointer", fontSize: 13, fontWeight: 600,
              display: "flex", alignItems: "center", gap: 8,
              marginBottom: -1,
              transition: "color 0.12s",
            }}
          >
            {t.label}
            <span style={{
              padding: "1px 7px", borderRadius: 10, fontSize: 11,
              background: tab === t.key ? "#1e293b" : "#0f172a",
              color: tab === t.key ? "#94a3b8" : "#334155",
            }}>
              {t.count}
            </span>
          </button>
        ))}
      </div>

      {/* Content */}
      {loading && (
        <div style={{ textAlign: "center", padding: 60, color: "#334155" }}>
          Loading…
        </div>
      )}

      {error && (
        <div style={{
          padding: "12px 16px", background: "#1a0a0a",
          border: "1px solid #e74c3c33", borderRadius: 8, color: "#e74c3c", fontSize: 13,
        }}>
          {error}
        </div>
      )}

      {!loading && !error && displayed.length === 0 && (
        <div style={{ textAlign: "center", padding: "60px 0" }}>
          <div style={{ fontSize: 40, marginBottom: 12 }}>
            {tab === "pending" ? "🎉" : "📭"}
          </div>
          <p style={{ color: "#334155", fontSize: 14 }}>
            {tab === "pending"
              ? "No pending requests — all clear!"
              : "No request history yet."}
          </p>
        </div>
      )}

      {!loading && displayed.map((req) => (
        <div key={req.id} style={{
          background: "#0a0f1a",
          border: "1px solid #1e293b",
          borderRadius: 12,
          padding: "22px 24px",
          marginBottom: 14,
          transition: "border-color 0.15s",
        }}
          onMouseEnter={e => e.currentTarget.style.borderColor = "#334155"}
          onMouseLeave={e => e.currentTarget.style.borderColor = "#1e293b"}
        >
          {/* Row 1: name + status */}
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
            <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
              <div style={{
                width: 36, height: 36, borderRadius: 8,
                background: "#1e293b",
                display: "flex", alignItems: "center", justifyContent: "center",
                fontSize: 16,
              }}>
                ⚙️
              </div>
              <div>
                <div style={{ color: "#f1f5f9", fontWeight: 700, fontSize: 15 }}>
                  {req.serviceName}
                </div>
                <div style={{ color: "#475569", fontSize: 12, marginTop: 2 }}>
                  Requested by{" "}
                  <span style={{ color: "#64748b", fontWeight: 600 }}>
                    {req.requestedBy}
                  </span>
                  {" · "}
                  {new Date(req.createdAt).toLocaleString()}
                </div>
              </div>
            </div>
            <StatusBadge status={req.status} />
          </div>

          {/* Reject reason */}
          {req.rejectReason && (
            <div style={{
              padding: "8px 12px", borderRadius: 6,
              background: "#1a0a0a", border: "1px solid #e74c3c22",
              color: "#94a3b8", fontSize: 12, marginBottom: 10,
            }}>
              <strong style={{ color: "#e74c3c" }}>Reason: </strong>
              {req.rejectReason}
            </div>
          )}

          {/* Reviewed by */}
          {req.reviewedBy && (
            <div style={{ fontSize: 12, color: "#334155", marginBottom: 10 }}>
              Reviewed by <span style={{ color: "#475569" }}>{req.reviewedBy}</span>
              {req.reviewedAt && (
                <> · {new Date(req.reviewedAt).toLocaleString()}</>
              )}
            </div>
          )}

          {/* YAML preview */}
          <YAMLPreview yaml={req.yamlPayload} />

          {/* Actions — pending only */}
          {req.status === "pending" && (
            <div style={{ display: "flex", gap: 10, marginTop: 16 }}>
              <button
                onClick={() => handleApprove(req.id, req.serviceName)}
                disabled={!!actionLoading[req.id]}
                style={{
                  padding: "9px 22px", borderRadius: 7,
                  border: "none",
                  background: actionLoading[req.id] === "approving"
                    ? "#1e293b"
                    : "linear-gradient(135deg, #10b981, #059669)",
                  color: "#fff", cursor: actionLoading[req.id] ? "not-allowed" : "pointer",
                  fontSize: 13, fontWeight: 700,
                  display: "flex", alignItems: "center", gap: 6,
                  opacity: actionLoading[req.id] ? 0.7 : 1,
                  boxShadow: actionLoading[req.id] ? "none" : "0 2px 12px rgba(16,185,129,0.25)",
                  transition: "opacity 0.12s",
                }}
              >
                {actionLoading[req.id] === "approving" ? (
                  <>
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5"
                      style={{ animation: "spin 1s linear infinite" }}>
                      <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
                    </svg>
                    Creating…
                  </>
                ) : (
                  <>
                    <svg width="13" height="13" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                      <path d="M20 6L9 17l-5-5"/>
                    </svg>
                    Approve & Create
                  </>
                )}
              </button>

              <button
                onClick={() => setRejectTarget(req)}
                disabled={!!actionLoading[req.id]}
                style={{
                  padding: "9px 22px", borderRadius: 7,
                  border: "1px solid #e74c3c44",
                  background: "transparent",
                  color: "#e74c3c", cursor: actionLoading[req.id] ? "not-allowed" : "pointer",
                  fontSize: 13, fontWeight: 700,
                  display: "flex", alignItems: "center", gap: 6,
                  opacity: actionLoading[req.id] ? 0.5 : 1,
                  transition: "background 0.12s, border-color 0.12s",
                }}
                onMouseEnter={e => { e.currentTarget.style.background = "#1a0a0a" }}
                onMouseLeave={e => { e.currentTarget.style.background = "transparent" }}
              >
                <svg width="13" height="13" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                  <path d="M18 6L6 18M6 6l12 12"/>
                </svg>
                Reject
              </button>
            </div>
          )}
        </div>
      ))}

      {/* Reject modal */}
      {rejectTarget && (
        <RejectModal
          onConfirm={(reason) => handleReject(rejectTarget.id, reason)}
          onCancel={() => setRejectTarget(null)}
          loading={actionLoading[rejectTarget.id] === "rejecting"}
        />
      )}

      {/* Toast */}
      {toast && (
        <div style={{
          position: "fixed", bottom: 28, right: 28,
          padding: "12px 20px", borderRadius: 10,
          background: "#0f172a", border: "1px solid #1e293b",
          color: "#e2e8f0", fontSize: 13, fontWeight: 500,
          boxShadow: "0 4px 24px rgba(0,0,0,0.5)",
          zIndex: 200, maxWidth: 380,
          animation: "slideUp 0.2s ease",
        }}>
          {toast}
        </div>
      )}

      <style>{`
        @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        @keyframes slideUp { from { transform: translateY(12px); opacity: 0; } to { transform: translateY(0); opacity: 1; } }
      `}</style>
    </div>
  )
}