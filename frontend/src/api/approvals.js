import { apiFetch } from "./index"

export async function fetchProdApprovals() {
  const res = await apiFetch("/api/approvals?environment=prod")
  if (!res || !res.ok) throw new Error("Failed to fetch approvals")
  return res.json()
}

export async function approveDeployment(id) {
  const res = await apiFetch(`/api/approvals/${id}/approve`, { method: "POST" })
  if (!res || !res.ok) throw new Error("Approval failed")
}

export async function rejectDeployment(id) {
  const res = await apiFetch(`/api/approvals/${id}/reject`, { method: "POST" })
  if (!res || !res.ok) throw new Error("Rejection failed")
}

