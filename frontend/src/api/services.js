import { apiFetch } from "./index"

export async function fetchServices() {
  const res = await apiFetch("/api/services")
  if (!res || !res.ok) throw new Error("Failed to fetch services")
  return res.json()
}

export async function fetchServiceDashboard(serviceName) {
  const res = await apiFetch(`/api/servicesdashboard/${serviceName}/dashboard`, {
    headers: { Accept: "application/json" },
  })
  if (!res || !res.ok) throw new Error("Failed to fetch service dashboard")
  return res.json()
}

export async function deployService(serviceName, environment) {
  const res = await apiFetch(`/api/deploy-services/${serviceName}/deploy`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ environment }),
  })
  if (!res || !res.ok) throw new Error("Deployment failed")
  return res.json()
}


export async function fetchPlatformStats() {
  const res = await apiFetch("/api/stats")
  if (!res || !res.ok) throw new Error("Failed to fetch stats")
  return res.json()
}


