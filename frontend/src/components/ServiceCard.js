import { useState } from "react"
import {
  fetchArtifactsByEnv,
  rollbackService,
} from "../api/serviceApi"

const ENV_CFG = {
  dev:  { color: "#6366f1", bg: "#0d0f2e", label: "Development" },
  test: { color: "#f59e0b", bg: "#1a1200", label: "Testing"     },
  prod: { color: "#10b981", bg: "#001a0f", label: "Production"  },
}

function Spinner({ color = "#6366f1", size = 12 }) {
  return (
    <span style={{
      width: size, height: size, borderRadius: "50%",
      border: `2px solid ${color}33`,
      borderTop: `2px solid ${color}`,
      display: "inline-block",
      animation: "sc-spin 0.7s linear infinite",
      flexShrink: 0,
    }}/>
  )
}

export default function ServiceCard({ serviceName, dashboard }) {
  const [selectedEnv,      setSelectedEnv]      = useState("")
  const [artifacts,        setArtifacts]        = useState([])
  const [selectedVersion,  setSelectedVersion]  = useState("")
  const [loadingArtifacts, setLoadingArtifacts] = useState(false)
  const [rollingBack,      setRollingBack]      = useState(false)
  const [rollbackSuccess,  setRollbackSuccess]  = useState(false)
  const [rollbackError,    setRollbackError]    = useState("")

  const environments = Object.keys(dashboard.environments || {})

  /* ── ENV SELECT → LOAD ARTIFACTS ── */
  const handleEnvSelect = async (env) => {
    if (loadingArtifacts || rollingBack) return
    setSelectedEnv(env)
    setSelectedVersion("")
    setArtifacts([])
    setRollbackSuccess(false)
    setRollbackError("")
    setLoadingArtifacts(true)
    try {
      const data = await fetchArtifactsByEnv(serviceName, env)
      const list = Array.isArray(data) ? data : []
      list.sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt))
      setArtifacts(list)
    } catch (err) {
      console.error("[ERROR] Failed to load artifacts", err)
      setArtifacts([])
    } finally {
      setLoadingArtifacts(false)
    }
  }

  /* ── ROLLBACK ── */
  const handleRollback = async () => {
    if (!selectedEnv || !selectedVersion) return
    const currentVersion = dashboard.environments[selectedEnv]?.currentVersion
    if (selectedVersion === currentVersion) {
      setRollbackError("This version is already running in the selected environment.")
      return
    }
    setRollingBack(true)
    setRollbackSuccess(false)
    setRollbackError("")
    try {
      await rollbackService(serviceName, {
        environment: selectedEnv,
        version: selectedVersion,
      })
      setRollbackSuccess(true)
    } catch (err) {
      console.error("[ERROR] Rollback failed", err)
      setRollbackError("Rollback failed to start. Please try again.")
    } finally {
      setRollingBack(false)
    }
  }

  const selectedCfg     = ENV_CFG[selectedEnv] ?? { color: "#6366f1", bg: "#0d0f2e" }
  const currentVersion  = dashboard.environments[selectedEnv]?.currentVersion
  const isReadyToRollback = !!selectedVersion && !rollingBack && selectedVersion !== currentVersion

  /* ── RENDER ── */
  return (
    <div style={{
      background: "#0a0f1a",
      border: "1px solid #1e293b",
      borderRadius: 14,
      overflow: "hidden",
    }}>
      <style>{`
        @keyframes sc-spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        @keyframes sc-fadeIn { from { opacity: 0; transform: translateY(6px); } to { opacity: 1; transform: translateY(0); } }
      `}</style>

      {/* ── Header ── */}
      <div style={{
        padding: "16px 20px",
        borderBottom: "1px solid #1e293b",
        background: "#060b12",
        display: "flex", alignItems: "center", justifyContent: "space-between",
      }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <div style={{
            width: 32, height: 32, borderRadius: 8,
            background: "#1e293b",
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 15,
          }}>
            ↩️
          </div>
          <div>
            <div style={{
              fontSize: 10, color: "#6366f1", fontWeight: 700,
              letterSpacing: "0.15em", textTransform: "uppercase",
              fontFamily: "monospace",
            }}>
              Rollback
            </div>
            <div style={{ fontSize: 13, color: "#f1f5f9", fontWeight: 700 }}>
              {serviceName}
            </div>
          </div>
        </div>

        {/* Owner + Runtime pills */}
        <div style={{ display: "flex", gap: 8 }}>
          {dashboard.ownerTeam && (
            <span style={{
              padding: "3px 10px", borderRadius: 20,
              background: "#1e293b", border: "1px solid #334155",
              color: "#64748b", fontSize: 11, fontFamily: "monospace",
            }}>
              {dashboard.ownerTeam}
            </span>
          )}
          {dashboard.runtime && (
            <span style={{
              padding: "3px 10px", borderRadius: 20,
              background: "#1e293b", border: "1px solid #334155",
              color: "#64748b", fontSize: 11, fontFamily: "monospace",
            }}>
              {dashboard.runtime}
            </span>
          )}
        </div>
      </div>

      <div style={{ padding: "20px" }}>

        {/* ── Step 1: Select Environment ── */}
        <div style={{ marginBottom: 20 }}>
          <div style={{
            fontSize: 10, color: "#334155", fontWeight: 700,
            letterSpacing: "0.12em", textTransform: "uppercase",
            fontFamily: "monospace", marginBottom: 10,
          }}>
            Step 1 — Select environment
          </div>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            {environments.map((env) => {
              const cfg     = ENV_CFG[env] ?? { color: "#6366f1", bg: "#0d0f2e" }
              const isActive = selectedEnv === env
              return (
                <button
                  key={env}
                  onClick={() => handleEnvSelect(env)}
                  disabled={loadingArtifacts || rollingBack}
                  style={{
                    padding: "8px 18px", borderRadius: 8,
                    border: `1px solid ${isActive ? cfg.color + "88" : cfg.color + "33"}`,
                    background: isActive ? cfg.color + "22" : cfg.bg,
                    color: isActive ? cfg.color : cfg.color + "88",
                    fontWeight: 700, fontSize: 12,
                    cursor: loadingArtifacts || rollingBack ? "not-allowed" : "pointer",
                    transition: "all 0.15s",
                    letterSpacing: "0.06em",
                    opacity: loadingArtifacts || rollingBack ? 0.5 : 1,
                    display: "flex", alignItems: "center", gap: 7,
                  }}
                  onMouseEnter={e => {
                    if (!loadingArtifacts && !rollingBack && !isActive) {
                      e.currentTarget.style.background = cfg.color + "15"
                      e.currentTarget.style.borderColor = cfg.color + "66"
                      e.currentTarget.style.color = cfg.color
                    }
                  }}
                  onMouseLeave={e => {
                    if (!isActive) {
                      e.currentTarget.style.background = cfg.bg
                      e.currentTarget.style.borderColor = cfg.color + "33"
                      e.currentTarget.style.color = cfg.color + "88"
                    }
                  }}
                >
                  {isActive && loadingArtifacts
                    ? <Spinner color={cfg.color} size={10}/>
                    : (
                      <span style={{
                        width: 6, height: 6, borderRadius: "50%",
                        background: isActive ? cfg.color : cfg.color + "55",
                        display: "inline-block", flexShrink: 0,
                      }}/>
                    )
                  }
                  {env.toUpperCase()}
                </button>
              )
            })}
          </div>
        </div>

        {/* ── Step 2: Select Version ── */}
        {selectedEnv && (
          <div style={{ animation: "sc-fadeIn 0.2s ease both" }}>

            {/* Current version info bar */}
            <div style={{
              background: "#060b12",
              border: `1px solid ${selectedCfg.color}22`,
              borderRadius: 8, padding: "10px 14px",
              marginBottom: 16,
              display: "flex", alignItems: "center", justifyContent: "space-between",
            }}>
              <div>
                <div style={{
                  fontSize: 9, color: "#334155", fontWeight: 700,
                  letterSpacing: "0.12em", textTransform: "uppercase",
                  fontFamily: "monospace", marginBottom: 3,
                }}>
                  Currently running in {selectedEnv}
                </div>
                <div style={{
                  fontFamily: "monospace", fontSize: 13,
                  color: currentVersion ? "#e2e8f0" : "#334155",
                  fontWeight: 600,
                }}>
                  {currentVersion ?? "— not deployed"}
                </div>
              </div>
              <span style={{
                padding: "3px 10px", borderRadius: 20,
                background: selectedCfg.color + "15",
                border: `1px solid ${selectedCfg.color}33`,
                color: selectedCfg.color,
                fontSize: 10, fontWeight: 700, letterSpacing: "0.08em",
                textTransform: "uppercase",
              }}>
                {selectedEnv}
              </span>
            </div>

            {/* Step 2 label */}
            <div style={{
              fontSize: 10, color: "#334155", fontWeight: 700,
              letterSpacing: "0.12em", textTransform: "uppercase",
              fontFamily: "monospace", marginBottom: 10,
            }}>
              Step 2 — Choose version to rollback to
            </div>

            {/* Version select */}
            <div style={{ position: "relative", marginBottom: 16 }}>
              <select
                value={selectedVersion}
                disabled={loadingArtifacts}
                onChange={e => {
                  setSelectedVersion(e.target.value)
                  setRollbackSuccess(false)
                  setRollbackError("")
                }}
                style={{
                  width: "100%",
                  padding: "10px 36px 10px 14px",
                  background: "#060b12",
                  border: `1px solid ${selectedVersion ? selectedCfg.color + "55" : "#1e293b"}`,
                  borderRadius: 8,
                  color: selectedVersion ? "#e2e8f0" : "#475569",
                  fontSize: 13, fontFamily: "monospace",
                  outline: "none",
                  cursor: loadingArtifacts ? "not-allowed" : "pointer",
                  appearance: "none",
                  transition: "border-color 0.15s",
                  opacity: loadingArtifacts ? 0.6 : 1,
                }}
              >
                <option value="">
                  {loadingArtifacts
                    ? "Loading versions…"
                    : artifacts.length === 0
                    ? "No versions available"
                    : "Select a version to rollback to"}
                </option>
                {artifacts.map((a) => {
                  const isCurrent = a.version === currentVersion
                  return (
                    <option
                      key={a.version}
                      value={a.version}
                      disabled={isCurrent}
                      style={{ background: "#0f172a", color: isCurrent ? "#334155" : "#e2e8f0" }}
                    >
                      {a.version}{isCurrent ? "  (current — cannot rollback)" : ""}
                      {a.createdAt ? `  ·  ${new Date(a.createdAt).toLocaleString()}` : ""}
                    </option>
                  )
                })}
              </select>

              {/* Chevron icon */}
              <svg
                width="12" height="12" fill="none" stroke="#475569"
                strokeWidth="2" viewBox="0 0 24 24"
                style={{
                  position: "absolute", right: 12,
                  top: "50%", transform: "translateY(-50%)",
                  pointerEvents: "none",
                }}
              >
                <path d="M6 9l6 6 6-6"/>
              </svg>
            </div>

            {/* Loading spinner row */}
            {loadingArtifacts && (
              <div style={{
                display: "flex", alignItems: "center", gap: 8,
                color: "#334155", fontSize: 12, marginBottom: 14,
              }}>
                <Spinner color={selectedCfg.color} size={12}/>
                Loading versions for {selectedEnv}…
              </div>
            )}

            {/* Selected version preview */}
            {selectedVersion && selectedVersion !== currentVersion && (
              <div style={{
                background: "#0a0a14",
                border: `1px solid ${selectedCfg.color}33`,
                borderRadius: 8, padding: "10px 14px",
                marginBottom: 16,
                display: "flex", alignItems: "center", gap: 10,
                animation: "sc-fadeIn 0.15s ease both",
              }}>
                <svg width="14" height="14" fill="none" stroke={selectedCfg.color} strokeWidth="2" viewBox="0 0 24 24">
                  <path d="M12 19V5M5 12l7-7 7 7"/>
                </svg>
                <div>
                  <div style={{ fontSize: 9, color: "#334155", fontWeight: 700, letterSpacing: "0.1em", textTransform: "uppercase", marginBottom: 2 }}>
                    Rolling back to
                  </div>
                  <div style={{ fontFamily: "monospace", fontSize: 13, color: selectedCfg.color, fontWeight: 700 }}>
                    {selectedVersion}
                  </div>
                </div>
              </div>
            )}

            {/* ── Rollback button ── */}
            <button
              onClick={handleRollback}
              disabled={!isReadyToRollback}
              style={{
                width: "100%",
                padding: "11px 0", borderRadius: 8,
                border: isReadyToRollback
                  ? `1px solid ${selectedCfg.color}55`
                  : "1px solid #1e293b",
                background: isReadyToRollback
                  ? selectedCfg.color + "22"
                  : "#1e293b",
                color: isReadyToRollback ? selectedCfg.color : "#334155",
                fontWeight: 700, fontSize: 13,
                cursor: isReadyToRollback ? "pointer" : "not-allowed",
                transition: "all 0.15s",
                display: "flex", alignItems: "center",
                justifyContent: "center", gap: 8,
                letterSpacing: "0.04em",
              }}
              onMouseEnter={e => {
                if (isReadyToRollback) {
                  e.currentTarget.style.background = selectedCfg.color + "33"
                  e.currentTarget.style.boxShadow  = `0 4px 20px ${selectedCfg.color}22`
                }
              }}
              onMouseLeave={e => {
                e.currentTarget.style.background = isReadyToRollback ? selectedCfg.color + "22" : "#1e293b"
                e.currentTarget.style.boxShadow  = "none"
              }}
            >
              {rollingBack ? (
                <>
                  <Spinner color={selectedCfg.color} size={13}/>
                  Rolling back…
                </>
              ) : (
                <>
                  <svg width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24">
                    <path d="M3 12a9 9 0 109-9 9 9 0 00-9 9"/>
                    <path d="M3 3v5h5"/>
                  </svg>
                  Rollback to {selectedVersion || "selected version"}
                </>
              )}
            </button>

            {/* ── Success banner ── */}
            {rollbackSuccess && (
              <div style={{
                marginTop: 12, padding: "12px 16px",
                background: "#001a0f", border: "1px solid #10b98144",
                borderRadius: 8, display: "flex", alignItems: "center", gap: 10,
                animation: "sc-fadeIn 0.2s ease both",
              }}>
                <span style={{ fontSize: 16 }}>✅</span>
                <div>
                  <div style={{ color: "#10b981", fontSize: 13, fontWeight: 700 }}>
                    Rollback triggered successfully
                  </div>
                  <div style={{ color: "#334155", fontSize: 11, marginTop: 2 }}>
                    {serviceName} ({selectedEnv}) → {selectedVersion}
                  </div>
                </div>
              </div>
            )}

            {/* ── Error banner ── */}
            {rollbackError && (
              <div style={{
                marginTop: 12, padding: "12px 16px",
                background: "#1a0a0a", border: "1px solid #e74c3c44",
                borderRadius: 8, display: "flex", alignItems: "center", gap: 10,
                animation: "sc-fadeIn 0.2s ease both",
              }}>
                <span style={{ fontSize: 16 }}>❌</span>
                <span style={{ color: "#e74c3c", fontSize: 13 }}>{rollbackError}</span>
              </div>
            )}
          </div>
        )}

        {/* ── Empty state when no env selected ── */}
        {!selectedEnv && (
          <div style={{
            textAlign: "center", padding: "28px 0",
            border: "1px dashed #1e293b", borderRadius: 10,
          }}>
            <div style={{ fontSize: 28, marginBottom: 8 }}>↩️</div>
            <div style={{ color: "#334155", fontSize: 13, fontWeight: 600 }}>
              Select an environment above to begin rollback
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
