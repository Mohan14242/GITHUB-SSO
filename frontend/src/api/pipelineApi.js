import { apiFetch } from "./index"

export async function fetchPipelineRun(runId) {
  const res = await apiFetch(`/api/pipeline/${runId}`)
  if (!res || !res.ok) throw new Error("Failed to fetch pipeline run")
  return res.json()
}

export async function fetchLatestPipelineRun(serviceName, environment) {
  const res = await apiFetch(`/api/pipeline/service/${serviceName}/${environment}`)
  if (!res || !res.ok) throw new Error("No pipeline runs found")
  return res.json()
}

// SSE — returns an EventSource and cleanup function
export function streamPipelineRun(runId, { onStageUpdated, onRunUpdated, onCompleted, onSnapshot, onError }) {
  const token = sessionStorage.getItem("token")
  const url   = `/api/pipeline/${runId}/stream?token=${encodeURIComponent(token)}`

  const es = new EventSource(url)

  es.addEventListener("run_snapshot",   e => onSnapshot?.(JSON.parse(e.data)))
  es.addEventListener("stage_updated",  e => onStageUpdated?.(JSON.parse(e.data)))
  es.addEventListener("run_updated",    e => onRunUpdated?.(JSON.parse(e.data)))
  es.addEventListener("run_completed",  e => {
    onCompleted?.(JSON.parse(e.data))
    es.close()
  })

  es.onerror = (err) => {
    onError?.(err)
    es.close()
  }

  return () => es.close() // cleanup function
}
