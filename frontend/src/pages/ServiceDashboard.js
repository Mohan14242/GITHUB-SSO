import { useEffect, useState } from "react"
import { useParams, Link } from "react-router-dom"
import { fetchServiceDashboard, deployService } from "../api/services"
import { fetchLatestPipelineRun } from "../api/pipelineApi"
import PipelineView from "../components/PipelineView"

const DEFAULT_ENVS = ["dev", "test", "prod"]
const POLL_INTERVAL_MS = 5000

const ENV_CFG = {
  dev:  { color: "#6366f1", bg: "#0d0f2e", label: "Development", icon: "⚗️"  },
  test: { color: "#f59e0b", bg: "#1a1200", label: "Testing",     icon: "🧪"  },
  prod: { color: "#10b981", bg: "#001a0f", label: "Production",  icon: "🚀"  },
}

const STATUS_CFG = {
  deployed:     { color: "#10b981", bg: "#10b98115", label: "Deployed",     dot: true  },
  not_deployed: { color: "#334155", bg: "#33415515", label: "Not Deployed", dot: false },
  deploying:    { color: "#f59e0b", bg: "#f59e0b15", label: "Deploying",    dot: true  },
  failed:       { color: "#e74c3c", bg: "#e74c3c15", label: "Failed",       dot: false },
}

function StatusBadge({ status }) {
  const cfg = STATUS_CFG[status] ?? STATUS_CFG.not_deployed
  return (
    <span style={{
      display: "inline-flex", alignItems: "center", gap: 6,
      padding: "4px 10px", borderRadius: 20,
      background: cfg.bg,
      border: `1px solid ${cfg.color}33`,
      color: cfg.color,
      fontSize: 11, fontWeight: 700, letterSpacing: "0.05em",
      textTransform: "uppercase",
    }}>
      {cfg.dot && (
        <span style={{
          width: 6, height: 6, borderRadius: "50%",
          background: cfg.color,
          animation: status === "deploying" ? "pulse 1.2s ease-in-out infinite" : "none",
        }} />
      )}
      {cfg.label}
    </span>
  )
}

function EnvCard({ env, data, deploying, onDeploy, onViewPipeline }) {
  const cfg   = ENV_CFG[env] ?? { color: "#6366f1", bg: "#0d0f2e", label: env, icon: "📦" }
  const isLoading = deploying[env]

  return (
    <div style={{
      background: "#0a1020",
      border: `1px solid ${cfg.color}22`,
      borderRadius: 14,
      padding: 24,
      display: "flex", flexDirection: "column", gap: 16,
      position: "relative", overflow: "hidden",
      transition: "border-color 0.2s, box-shadow 0.2s",
    }}
      onMouseEnter={e => {
        e.currentTarget.style.borderColor = cfg.color + "55"
        e.currentTarget.style.boxShadow   = `0 0 24px ${cfg.color}11`
      }}
      onMouseLeave={e => {
        e.currentTarget.style.borderColor = cfg.color + "22"
        e.currentTarget.style.boxShadow   = "none"
      }}
    >
      {/* Ambient glow top-right */}
      <div style={{
        position: "absolute", top: -30, right: -30,
        width: 100, height: 100, borderRadius: "50%",
        background: cfg.color + "0a", pointerEvents: "none",
      }} />

      {/* Env header */}
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <div style={{
            width: 36, height: 36, borderRadius: 10,
            background: cfg.bg,
            border: `1px solid ${cfg.color}33`,
            display: "flex", alignItems: "center", justifyContent: "center",
            fontSize: 16,
          }}>
            {cfg.icon}
          </div>
          <div>
            <div style={{
              fontSize: 11, color: cfg.color, fontWeight: 700,
              letterSpacing: "0.12em", textTransform: "uppercase",
              fontFamily: "monospace",
            }}>
              {cfg.label}
            </div>
            <div style={{ fontSize: 12, color: "#334155", fontFamily: "monospace" }}>
              {env}
            </div>
          </div>
        </div>
        <StatusBadge status={data?.status ?? "not_deployed"} />
      </div>

      {/* Version row */}
      <div style={{
        background: "#060b12",
        border: "1px solid #0f172a",
        borderRadius: 8, padding: "10px 14px",
      }}>
        <div style={{ fontSize: 10, color: "#334155", marginBottom: 4, textTransform: "uppercase", letterSpacing: "0.1em" }}>
          Current Version
        </div>
        <div style={{
          fontFamily: "monospace", fontSize: 12,
          color: data?.currentVersion ? "#e2e8f0" : "#334155",
        }}>
          {data?.currentVersion ?? "—  not deployed"}
        </div>
      </div>

      {/* Meta */}
      {data?.lastDeployedAt && (
        <div style={{ fontSize: 11, color: "#334155", display: "flex", gap: 16 }}>
          <span>🕐 {new Date(data.lastDeployedAt).toLocaleString()}</span>
        </div>
      )}

      {/* Actions */}
      <div style={{ display: "flex", gap: 8, marginTop: 4 }}>
        <button
          onClick={onDeploy}
          disabled={isLoading}
          style={{
            flex: 1,
            padding: "10px 0",
            borderRadius: 8,
            border: `1px solid ${cfg.color}44`,
            background: isLoading ? cfg.color + "11" : cfg.color + "22",
            color: isLoading ? cfg.color + "88" : cfg.color,
            fontWeight: 700, fontSize: 12,
            cursor: isLoading ? "not-allowed" : "pointer",
            transition: "all 0.15s",
            display: "flex", alignItems: "center", justifyContent: "center", gap: 7,
            letterSpacing: "0.05em",
          }}
          onMouseEnter={e => { if (!isLoading) e.currentTarget.style.background = cfg.color + "33" }}
          onMouseLeave={e => { if (!isLoading) e.currentTarget.style.background = cfg.color + "22" }}
        >
          {isLoading
            ? <><Spinner color={cfg.color} /> Triggering...</>
            : <>▶ Deploy to {env.toUpperCase()}</>
          }
        </button>

        <button
          onClick={onViewPipeline}
          title="View latest pipeline"
          style={{
            padding: "10px 14px",
            borderRadius: 8,
            border: "1px solid #1e293b",
            background: "#060b12",
            color: "#475569",
            fontWeight: 600, fontSize: 12,
            cursor: "pointer",
            transition: "all 0.15s",
          }}
          onMouseEnter={e => { e.currentTarget.style.borderColor = "#334155"; e.currentTarget.style.color = "#94a3b8" }}
          onMouseLeave={e => { e.currentTarget.style.borderColor = "#1e293b"; e.currentTarget.style.color = "#475569" }}
        >
          ⚡
        </button>
      </div>
    </div>
  )
}

function Spinner({ color = "#6366f1", size = 12 }) {
  return (
    <span style={{
      width: size, height: size, borderRadius: "50%",
      border: `2px solid ${color}33`,
      borderTop: `2px solid ${color}`,
      display: "inline-block",
      animation: "spin 0.7s linear infinite",
      flexShrink: 0,
    }} />
  )
}

function StatBox({ label, value, color = "#6366f1" }) {
  return (
    <div style={{
      background: "#0a1020",
      border: "1px solid #0f172a",
      borderRadius: 10, padding: "14px 18px",
      display: "flex", flexDirection: "column", gap: 4,
    }}>
      <div style={{ fontSize: 10, color: "#334155", textTransform: "uppercase", letterSpacing: "0.1em" }}>
        {label}
      </div>
      <div style={{ fontSize: 18, fontWeight: 800, color, fontFamily: "monospace" }}>
        {value ?? "—"}
      </div>
    </div>
  )
}

export default function ServiceDashboard() {
  const { serviceName } = useParams()

  const [dashboard,       setDashboard]       = useState(null)
  const [deploying,       setDeploying]       = useState({})
  const [loading,         setLoading]         = useState(true)
  const [showPipeline,    setShowPipeline]    = useState(false)
  const [pipelineRunId,   setPipelineRunId]   = useState(null)
  const [pipelineEnv,     setPipelineEnv]     = useState(null)
  const [lastUpdated,     setLastUpdated]     = useState(null)

  // ── Load + poll ──────────────────────────────────
  useEffect(() => {
    let isMounted = true
    async function load() {
      try {
        const data = await fetchServiceDashboard(serviceName)
        if (isMounted) { setDashboard(data); setLastUpdated(new Date()) }
      } catch {
        if (isMounted) setDashboard(null)
      } finally {
        if (isMounted) setLoading(false)
      }
    }
    load()
    const iv = setInterval(load, POLL_INTERVAL_MS)
    return () => { isMounted = false; clearInterval(iv) }
  }, [serviceName])

  // ── Deploy ───────────────────────────────────────
  const handleDeploy = async (env) => {
    setDeploying(p => ({ ...p, [env]: true }))
    try {
      const res = await deployService(serviceName, env)
      if (res?.runId) {
        setPipelineRunId(res.runId)
        setPipelineEnv(env)
        setShowPipeline(true)
      }
    } catch {
      alert("Failed to trigger deployment")
    } finally {
      setDeploying(p => ({ ...p, [env]: false }))
    }
  }

  // ── View latest pipeline for an env ─────────────
  const handleViewPipeline = async (env) => {
    try {
      const run = await fetchLatestPipelineRun(serviceName, env)
      if (run?.id) {
        setPipelineRunId(run.id)
        setPipelineEnv(env)
        setShowPipeline(true)
      }
    } catch {
      alert("No pipeline runs found for " + env)
    }
  }

  const envs        = dashboard?.environments ? Object.keys(dashboard.environments) : DEFAULT_ENVS
  const deployedEnvs = envs.filter(e => dashboard?.environments?.[e]?.status === "deployed").length
  const totalEnvs    = envs.length

  return (
    <div style={{
      minHeight: "100vh",
      background: "#060b12",
      color: "#e2e8f0",
      fontFamily: "'DM Sans', sans-serif",
      padding: "32px 36px",
    }}>

      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;700;800&display=swap');
        @keyframes spin  { from { transform: rotate(0deg) } to { transform: rotate(360deg) } }
        @keyframes pulse { 0%,100% { opacity: 1 } 50% { opacity: 0.3 } }
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(12px) }
          to   { opacity: 1; transform: translateY(0) }
        }
      `}</style>

      {/* ── Back link ── */}
      <div style={{ marginBottom: 28, animation: "fadeUp 0.4s ease both" }}>
        <Link to="/" style={{
          color: "#334155", fontSize: 12, fontWeight: 600,
          textDecoration: "none", display: "inline-flex", alignItems: "center", gap: 6,
          padding: "6px 12px", borderRadius: 7,
          border: "1px solid #0f172a", background: "#0a1020",
          transition: "all 0.15s",
        }}
          onMouseEnter={e => { e.currentTarget.style.color = "#94a3b8"; e.currentTarget.style.borderColor = "#1e293b" }}
          onMouseLeave={e => { e.currentTarget.style.color = "#334155"; e.currentTarget.style.borderColor = "#0f172a" }}
        >
          ← Back to Services
        </Link>
      </div>

      {/* ── Header ── */}
      <div style={{
        marginBottom: 32,
        display: "flex", alignItems: "flex-start", justifyContent: "space-between",
        animation: "fadeUp 0.4s ease 0.05s both",
      }}>
        <div>
          <div style={{
            fontSize: 10, color: "#6366f1", fontWeight: 700,
            letterSpacing: "0.2em", textTransform: "uppercase",
            fontFamily: "monospace", marginBottom: 8,
          }}>
            Service Dashboard
          </div>
          <h1 style={{
            margin: 0, fontSize: 28, fontWeight: 800,
            color: "#f1f5f9", letterSpacing: "-0.02em",
          }}>
            {serviceName}
          </h1>
          {dashboard?.description && (
            <p style={{ margin: "6px 0 0", color: "#475569", fontSize: 13 }}>
              {dashboard.description}
            </p>
          )}
        </div>

        {/* Live indicator */}
        <div style={{
          display: "flex", alignItems: "center", gap: 8,
          padding: "8px 14px", borderRadius: 20,
          background: "#0a1020", border: "1px solid #0f172a",
          fontSize: 11, color: "#334155",
        }}>
          <span style={{
            width: 6, height: 6, borderRadius: "50%",
            background: "#10b981",
            animation: "pulse 2s ease-in-out infinite",
            display: "inline-block",
          }} />
          Live · refreshes every {POLL_INTERVAL_MS / 1000}s
          {lastUpdated && (
            <span style={{ color: "#1e293b", marginLeft: 4 }}>
              · {lastUpdated.toLocaleTimeString()}
            </span>
          )}
        </div>
      </div>

      {/* ── Stat row ── */}
      <div style={{
        display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 12,
        marginBottom: 28,
        animation: "fadeUp 0.4s ease 0.1s both",
      }}>
        <StatBox label="Environments"  value={totalEnvs}    color="#6366f1" />
        <StatBox label="Deployed"      value={deployedEnvs} color="#10b981" />
        <StatBox label="Template"      value={dashboard?.templateName ?? "—"} color="#f59e0b" />
        <StatBox label="Runtime"       value={dashboard?.runtime ?? "—"}      color="#06b6d4" />
      </div>

      {/* ── Loading ── */}
      {loading && (
        <div style={{
          display: "flex", alignItems: "center", gap: 12,
          color: "#334155", padding: "60px 0",
          animation: "fadeUp 0.3s ease both",
        }}>
          <Spinner color="#6366f1" size={16} />
          Loading {serviceName} dashboard…
        </div>
      )}

      {/* ── Env cards ── */}
      {!loading && (
        <div style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fill, minmax(300px, 1fr))",
          gap: 16,
          animation: "fadeUp 0.4s ease 0.15s both",
        }}>
          {envs.map(env => (
            <EnvCard
              key={env}
              env={env}
              data={dashboard?.environments?.[env]}
              deploying={deploying}
              onDeploy={() => handleDeploy(env)}
              onViewPipeline={() => handleViewPipeline(env)}
            />
          ))}
        </div>
      )}

      {/* ── Empty state ── */}
      {!loading && !dashboard && (
        <div style={{
          textAlign: "center", padding: "80px 0",
          animation: "fadeUp 0.4s ease both",
        }}>
          <div style={{ fontSize: 48, marginBottom: 16 }}>📦</div>
          <div style={{ color: "#475569", fontWeight: 700, fontSize: 15, marginBottom: 8 }}>
            Service not deployed yet
          </div>
          <div style={{ color: "#334155", fontSize: 13 }}>
            Use the deploy buttons above to initialize this service
          </div>
        </div>
      )}

      {/* ── Artifacts table ── */}
      {!loading && dashboard?.artifacts?.length > 0 && (
        <div style={{
          marginTop: 32,
          animation: "fadeUp 0.4s ease 0.2s both",
        }}>
          <div style={{
            fontSize: 10, color: "#334155", fontWeight: 700,
            letterSpacing: "0.15em", textTransform: "uppercase",
            marginBottom: 12, fontFamily: "monospace",
          }}>
            Recent Deployments
          </div>
          <div style={{
            background: "#0a1020",
            border: "1px solid #0f172a",
            borderRadius: 12, overflow: "hidden",
          }}>
            <table style={{ width: "100%", borderCollapse: "collapse" }}>
              <thead>
                <tr style={{ borderBottom: "1px solid #0f172a" }}>
                  {["Version", "Environment", "Action", "Status", "Time"].map(h => (
                    <th key={h} style={{
                      padding: "10px 16px", textAlign: "left",
                      fontSize: 10, color: "#334155", fontWeight: 700,
                      letterSpacing: "0.1em", textTransform: "uppercase",
                    }}>
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {dashboard.artifacts.slice(0, 8).map((a, i) => (
                  <tr key={i} style={{
                    borderBottom: "1px solid #0a1020",
                    transition: "background 0.15s",
                  }}
                    onMouseEnter={e => e.currentTarget.style.background = "#060b12"}
                    onMouseLeave={e => e.currentTarget.style.background = "transparent"}
                  >
                    <td style={{ padding: "10px 16px", fontFamily: "monospace", fontSize: 11, color: "#94a3b8" }}>
                      {a.version}
                    </td>
                    <td style={{ padding: "10px 16px" }}>
                      <span style={{
                        fontSize: 11, fontWeight: 700,
                        color: ENV_CFG[a.environment]?.color ?? "#94a3b8",
                        fontFamily: "monospace",
                      }}>
                        {a.environment}
                      </span>
                    </td>
                    <td style={{ padding: "10px 16px", fontSize: 11, color: "#475569", textTransform: "capitalize" }}>
                      {a.action}
                    </td>
                    <td style={{ padding: "10px 16px" }}>
                      <StatusBadge status={a.status} />
                    </td>
                    <td style={{ padding: "10px 16px", fontSize: 11, color: "#334155", fontFamily: "monospace" }}>
                      {a.createdAt ? new Date(a.createdAt).toLocaleString() : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ── Pipeline modal ── */}
      {showPipeline && pipelineRunId && (
        <PipelineView
          runId={pipelineRunId}
          serviceName={serviceName}
          environment={pipelineEnv}
          onClose={() => setShowPipeline(false)}
        />
      )}
    </div>
  )
}
