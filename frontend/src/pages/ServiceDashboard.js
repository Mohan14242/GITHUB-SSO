import { useEffect, useState } from "react"
import { useParams, Link } from "react-router-dom"
import { fetchServiceDashboard, deployService } from "../api/services"
import { fetchLatestPipelineRun } from "../api/pipelineApi"
import PipelineView from "../components/PipelineView"

const DEFAULT_ENVS    = ["dev", "test", "prod"]
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
      fontSize: 11, fontWeight: 700,
      letterSpacing: "0.05em", textTransform: "uppercase",
    }}>
      {cfg.dot && (
        <span style={{
          width: 6, height: 6, borderRadius: "50%",
          background: cfg.color,
          animation: status === "deploying"
            ? "pulse 1.2s ease-in-out infinite" : "none",
        }}/>
      )}
      {cfg.label}
    </span>
  )
}

function EnvCard({ env, data, deploying, onDeploy, onViewPipeline }) {
  const cfg       = ENV_CFG[env] ?? { color: "#6366f1", bg: "#0d0f2e", label: env, icon: "📦" }
  const isLoading = deploying?.[env] === true

  return (
    <div
      style={{
        background: "#0a1020",
        border: `1px solid ${cfg.color}22`,
        borderRadius: 14,
        padding: 22,
        display: "flex", flexDirection: "column", gap: 14,
        position: "relative", overflow: "hidden",
        transition: "border-color 0.2s, box-shadow 0.2s",
      }}
      onMouseEnter={e => {
        e.currentTarget.style.borderColor = cfg.color + "55"
        e.currentTarget.style.boxShadow   = `0 0 20px ${cfg.color}11`
      }}
      onMouseLeave={e => {
        e.currentTarget.style.borderColor = cfg.color + "22"
        e.currentTarget.style.boxShadow   = "none"
      }}
    >
      {/* Ambient glow */}
      <div style={{
        position: "absolute", top: -30, right: -30,
        width: 100, height: 100, borderRadius: "50%",
        background: cfg.color + "0a", pointerEvents: "none",
      }}/>

      {/* Env header */}
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <div style={{
            width: 34, height: 34, borderRadius: 9,
            background: cfg.bg, border: `1px solid ${cfg.color}33`,
            display: "flex", alignItems: "center",
            justifyContent: "center", fontSize: 15,
          }}>
            {cfg.icon}
          </div>
          <div>
            <div style={{
              fontSize: 10, color: cfg.color, fontWeight: 700,
              letterSpacing: "0.12em", textTransform: "uppercase",
              fontFamily: "monospace",
            }}>
              {cfg.label}
            </div>
            <div style={{ fontSize: 11, color: "#334155", fontFamily: "monospace" }}>
              {env}
            </div>
          </div>
        </div>
        <StatusBadge status={data?.status ?? "not_deployed"}/>
      </div>

      {/* Current version */}
      <div style={{
        background: "#060b12", border: "1px solid #0f172a",
        borderRadius: 7, padding: "9px 13px",
      }}>
        <div style={{
          fontSize: 9, color: "#334155", marginBottom: 3,
          textTransform: "uppercase", letterSpacing: "0.1em",
        }}>
          Current Version
        </div>
        <div style={{
          fontFamily: "monospace", fontSize: 12,
          color: data?.currentVersion ? "#e2e8f0" : "#334155",
        }}>
          {data?.currentVersion ?? "—  not deployed"}
        </div>
      </div>

      {/* Last deployed */}
      {data?.deployedAt && (
        <div style={{ fontSize: 11, color: "#334155" }}>
          🕐 {new Date(data.deployedAt).toLocaleString()}
        </div>
      )}

      {/* Actions */}
      <div style={{ display: "flex", gap: 8, marginTop: 2 }}>
        <button
          onClick={onDeploy}
          disabled={isLoading}
          style={{
            flex: 1, padding: "9px 0", borderRadius: 8,
            border: `1px solid ${cfg.color}44`,
            background: isLoading ? cfg.color + "11" : cfg.color + "22",
            color: isLoading ? cfg.color + "88" : cfg.color,
            fontWeight: 700, fontSize: 12,
            cursor: isLoading ? "not-allowed" : "pointer",
            transition: "all 0.15s",
            display: "flex", alignItems: "center",
            justifyContent: "center", gap: 7,
            letterSpacing: "0.05em",
          }}
          onMouseEnter={e => { if (!isLoading) e.currentTarget.style.background = cfg.color + "33" }}
          onMouseLeave={e => { if (!isLoading) e.currentTarget.style.background = cfg.color + "22" }}
        >
          {isLoading
            ? <><Spinner color={cfg.color}/> Triggering…</>
            : <>▶ Deploy to {env.toUpperCase()}</>
          }
        </button>

        <button
          onClick={onViewPipeline}
          title="View latest pipeline"
          style={{
            padding: "9px 13px", borderRadius: 8,
            border: "1px solid #1e293b", background: "#060b12",
            color: "#475569", fontWeight: 600, fontSize: 12,
            cursor: "pointer", transition: "all 0.15s",
          }}
          onMouseEnter={e => {
            e.currentTarget.style.borderColor = "#334155"
            e.currentTarget.style.color       = "#94a3b8"
          }}
          onMouseLeave={e => {
            e.currentTarget.style.borderColor = "#1e293b"
            e.currentTarget.style.color       = "#475569"
          }}
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
    }}/>
  )
}

function StatBox({ label, value, color = "#6366f1" }) {
  return (
    <div style={{
      background: "#0a1020", border: "1px solid #0f172a",
      borderRadius: 10, padding: "13px 16px",
      display: "flex", flexDirection: "column", gap: 4,
    }}>
      <div style={{
        fontSize: 9, color: "#334155",
        textTransform: "uppercase", letterSpacing: "0.1em",
      }}>
        {label}
      </div>
      <div style={{
        fontSize: 18, fontWeight: 800,
        color, fontFamily: "monospace",
      }}>
        {value ?? "—"}
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────────────────────────
export default function ServiceDashboard() {
  const { serviceName } = useParams()

  const [dashboard,     setDashboard]     = useState(null)
  const [deploying,     setDeploying]     = useState({})
  const [loading,       setLoading]       = useState(true)
  const [showPipeline,  setShowPipeline]  = useState(false)
  const [pipelineRunId, setPipelineRunId] = useState(null)
  const [pipelineEnv,   setPipelineEnv]   = useState(null)
  const [lastUpdated,   setLastUpdated]   = useState(null)

  // ── Poll dashboard ────────────────────────────────────────────
  useEffect(() => {
    let mounted = true
    async function load() {
      try {
        const data = await fetchServiceDashboard(serviceName)
        if (mounted) { setDashboard(data); setLastUpdated(new Date()) }
      } catch {
        if (mounted) setDashboard(null)
      } finally {
        if (mounted) setLoading(false)
      }
    }
    load()
    const iv = setInterval(load, POLL_INTERVAL_MS)
    return () => { mounted = false; clearInterval(iv) }
  }, [serviceName])

  // ── Deploy → open pipeline panel ─────────────────────────────
  const handleDeploy = async (env) => {
    setDeploying(prev => ({ ...prev, [env]: true }))

    try {
      // Step 1: trigger deployment
      const res = await deployService(serviceName, env)

      let runId = res?.runId

      // Step 2: fallback if backend did not return runId
      if (!runId) {
        console.warn("runId not returned from deploy API, fetching latest pipeline")

        for (let attempt = 0; attempt < 5; attempt++) {
          try {
            const latest = await fetchLatestPipelineRun(serviceName, env)

            if (latest?.id) {
              runId = latest.id
              break
            }

          } catch (err) {
            console.warn("Pipeline not ready yet, retrying...", err)
          }

          // wait 2 seconds before retry
          await new Promise(resolve => setTimeout(resolve, 2000))
        }
      }

      if (!runId) {
        throw new Error("Pipeline run not found after deployment")
      }

      // Step 3: open pipeline view
      setPipelineRunId(runId)
      setPipelineEnv(env)
      setShowPipeline(true)

    } catch (err) {
      console.error("Deploy failed:", err)
      alert("Deployment failed. Check logs.")

    } finally {
      setDeploying(prev => ({ ...prev, [env]: false }))
    }
  }
  // ── View latest pipeline for env ──────────────────────────────
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

  const closePipeline = () => {
    setShowPipeline(false)
    setPipelineRunId(null)
    setPipelineEnv(null)
  }

  const envs         = dashboard?.environments
    ? Object.keys(dashboard.environments) : DEFAULT_ENVS
  const deployedEnvs = envs.filter(e =>
    dashboard?.environments?.[e]?.status === "deployed"
  ).length

  return (
    <div style={{
      minHeight: "100vh",
      background: "#060b12",
      color: "#e2e8f0",
      fontFamily: "'DM Sans', sans-serif",
      padding: "28px 32px",
    }}>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;700;800&display=swap');
        @keyframes spin    { from { transform:rotate(0deg) } to { transform:rotate(360deg) } }
        @keyframes pulse   { 0%,100% { opacity:1 } 50% { opacity:0.3 } }
        @keyframes fadeUp  { from { opacity:0;transform:translateY(12px) } to { opacity:1;transform:translateY(0) } }
        @keyframes slideIn { from { opacity:0;transform:translateX(24px) } to { opacity:1;transform:translateX(0) } }
      `}</style>

      {/* ── Back link ── */}
      <div style={{ marginBottom: 24, animation: "fadeUp 0.35s ease both" }}>
        <Link to="/" style={{
          color: "#334155", fontSize: 12, fontWeight: 600,
          textDecoration: "none",
          display: "inline-flex", alignItems: "center", gap: 6,
          padding: "5px 11px", borderRadius: 7,
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
        marginBottom: 28,
        display: "flex", alignItems: "flex-start",
        justifyContent: "space-between",
        animation: "fadeUp 0.35s ease 0.05s both",
      }}>
        <div>
          <div style={{
            fontSize: 10, color: "#6366f1", fontWeight: 700,
            letterSpacing: "0.2em", textTransform: "uppercase",
            fontFamily: "monospace", marginBottom: 6,
          }}>
            Service Dashboard
          </div>
          <h1 style={{
            margin: 0, fontSize: 26, fontWeight: 800,
            color: "#f1f5f9", letterSpacing: "-0.02em",
          }}>
            {serviceName}
          </h1>
          {dashboard?.description && (
            <p style={{ margin: "5px 0 0", color: "#475569", fontSize: 13 }}>
              {dashboard.description}
            </p>
          )}
        </div>

        {/* Live indicator */}
        <div style={{
          display: "flex", alignItems: "center", gap: 7,
          padding: "7px 13px", borderRadius: 20,
          background: "#0a1020", border: "1px solid #0f172a",
          fontSize: 11, color: "#334155",
        }}>
          <span style={{
            width: 6, height: 6, borderRadius: "50%",
            background: "#10b981", display: "inline-block",
            animation: "pulse 2s ease-in-out infinite",
          }}/>
          Live · {POLL_INTERVAL_MS / 1000}s refresh
          {lastUpdated && (
            <span style={{ color: "#1e293b", marginLeft: 3 }}>
              · {lastUpdated.toLocaleTimeString()}
            </span>
          )}
        </div>
      </div>

      {/* ── Stats row ── */}
      <div style={{
        display: "grid",
        gridTemplateColumns: "repeat(4,1fr)",
        gap: 10, marginBottom: 24,
        animation: "fadeUp 0.35s ease 0.08s both",
      }}>
        <StatBox label="Environments" value={envs.length}                            color="#6366f1"/>
        <StatBox label="Deployed"     value={deployedEnvs}                           color="#10b981"/>
        <StatBox label="Template"     value={dashboard?.templateName ?? "—"}         color="#f59e0b"/>
        <StatBox label="Runtime"      value={dashboard?.runtime ?? "—"}              color="#06b6d4"/>
      </div>

      {/* ── Loading ── */}
      {loading && (
        <div style={{
          display: "flex", alignItems: "center", gap: 10,
          color: "#334155", padding: "60px 0",
        }}>
          <Spinner color="#6366f1" size={16}/>
          Loading {serviceName} dashboard…
        </div>
      )}

      {/* ── Main content: env cards + pipeline panel side by side ── */}
      {!loading && (
        <div style={{
          display: "flex",
          gap: 16,
          alignItems: "flex-start",
          animation: "fadeUp 0.35s ease 0.12s both",
        }}>

          {/* Env cards grid — shrinks when pipeline is open */}
          <div style={{
            flex: showPipeline ? "0 0 auto" : "1 1 auto",
            width: showPipeline ? "min(340px, 42%)" : "100%",
            transition: "width 0.3s ease",
          }}>
            <div style={{
              display: "grid",
              gridTemplateColumns: showPipeline
                ? "1fr"                                    // single column when panel open
                : "repeat(auto-fill,minmax(290px,1fr))",  // normal grid when closed
              gap: 14,
              transition: "grid-template-columns 0.3s ease",
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
          </div>

          {/* Pipeline panel — slides in from the right */}
          {showPipeline && pipelineRunId && (
            <div style={{
              flex: "1 1 auto",
              minWidth: 0,
              height: "calc(100vh - 260px)",
              minHeight: 480,
              animation: "slideIn 0.25s ease both",
              position: "sticky",
              top: 28,
            }}>
              <PipelineView
                runId={pipelineRunId}
                serviceName={serviceName}
                environment={pipelineEnv}
                onClose={closePipeline}
              />
            </div>
          )}
        </div>
      )}

      {/* ── Empty state ── */}
      {!loading && !dashboard && !showPipeline && (
        <div style={{ textAlign: "center", padding: "80px 0" }}>
          <div style={{ fontSize: 48, marginBottom: 14 }}>📦</div>
          <div style={{ color: "#475569", fontWeight: 700, fontSize: 15, marginBottom: 6 }}>
            Service not deployed yet
          </div>
          <div style={{ color: "#334155", fontSize: 13 }}>
            Use the deploy buttons above to initialize this service
          </div>
        </div>
      )}

      {/* ── Artifacts / Recent deployments ── */}
      {!loading && dashboard?.artifacts?.length > 0 && (
        <div style={{ marginTop: 28 }}>
          <div style={{
            fontSize: 10, color: "#334155", fontWeight: 700,
            letterSpacing: "0.15em", textTransform: "uppercase",
            marginBottom: 10, fontFamily: "monospace",
          }}>
            Recent Deployments
          </div>
          <div style={{
            background: "#0a1020", border: "1px solid #0f172a",
            borderRadius: 12, overflow: "hidden",
          }}>
            <table style={{ width: "100%", borderCollapse: "collapse" }}>
              <thead>
                <tr style={{ borderBottom: "1px solid #0f172a" }}>
                  {["Version", "Environment", "Action", "Status", "Time"].map(h => (
                    <th key={h} style={{
                      padding: "9px 14px", textAlign: "left",
                      fontSize: 9, color: "#334155", fontWeight: 700,
                      letterSpacing: "0.1em", textTransform: "uppercase",
                    }}>
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {dashboard.artifacts.slice(0, 8).map((a, i) => (
                  <tr key={i}
                    style={{ borderBottom: "1px solid #0a1020", transition: "background 0.15s" }}
                    onMouseEnter={e => e.currentTarget.style.background = "#060b12"}
                    onMouseLeave={e => e.currentTarget.style.background = "transparent"}
                  >
                    <td style={{ padding: "9px 14px", fontFamily: "monospace", fontSize: 11, color: "#94a3b8" }}>
                      {a.version}
                    </td>
                    <td style={{ padding: "9px 14px" }}>
                      <span style={{
                        fontSize: 11, fontWeight: 700,
                        color: ENV_CFG[a.environment]?.color ?? "#94a3b8",
                        fontFamily: "monospace",
                      }}>
                        {a.environment}
                      </span>
                    </td>
                    <td style={{ padding: "9px 14px", fontSize: 11, color: "#475569", textTransform: "capitalize" }}>
                      {a.action}
                    </td>
                    <td style={{ padding: "9px 14px" }}>
                      <StatusBadge status={a.status}/>
                    </td>
                    <td style={{ padding: "9px 14px", fontSize: 11, color: "#334155", fontFamily: "monospace" }}>
                      {a.createdAt ? new Date(a.createdAt).toLocaleString() : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
