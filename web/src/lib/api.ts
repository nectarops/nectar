// SPDX-License-Identifier: AGPL-3.0-only

export type User = {
  id: number
  username: string
  role: string
}

export type VersionInfo = {
  version: string
  commit: string
  sourceUrl: string
}

export type ClusterSnapshot = {
  available: boolean
  error?: string
  hostname: string
  operatingSystem: string
  architecture: string
  kernelVersion: string
  dockerVersion: string
  dockerApiVersion: string
  swarmState: string
  nodeId: string
  managerStatus?: string
  nodeRole: string
  nodeStatus: string
  availability: string
  managers: number
  nodes: number
  cpus: number
  memoryBytes: number
  containersRunning: number
  images: number
}

export type DeploymentInput = {
  serviceName: string
  image: string
  version: string
  domain: string
  acmeEmail: string
  port: number
  replicas: number
}

export type DeploymentResult = {
  serviceId: string
  image: string
  updated: boolean
  warnings: string[]
}

export class ApiError extends Error {
  readonly status: number
  readonly code?: string

  constructor(
    message: string,
    status: number,
    code?: string,
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

async function request(
  path: string,
  options: RequestInit = {},
  signal?: AbortSignal,
): Promise<unknown> {
  const response = await fetch(path, {
    ...options,
    credentials: 'same-origin',
    headers: {
      Accept: 'application/json',
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...options.headers,
    },
    signal,
  })

  if (!response.ok) {
    const payload = await response.json().catch(() => null)
    const error = optionalRecord(optionalRecord(payload)?.error)
    throw new ApiError(
      optionalString(error?.message) ?? `Request failed with status ${response.status}.`,
      response.status,
      optionalString(error?.code),
    )
  }

  if (response.status === 204) {
    return undefined
  }

  return response.json() as Promise<unknown>
}

export async function getSetupStatus(signal?: AbortSignal) {
  const value = requireRecord(await request('/api/v1/setup/status', {}, signal))
  return { completed: requireBoolean(value.completed, 'completed') }
}

export async function getVersion(signal?: AbortSignal): Promise<VersionInfo> {
  const value = requireRecord(await request('/api/v1/version', {}, signal))
  return {
    version: requireString(value.version, 'version'),
    commit: requireString(value.commit, 'commit'),
    sourceUrl: requireString(value.sourceUrl, 'sourceUrl'),
  }
}

export async function getSession(signal?: AbortSignal) {
  return parseUser(await request('/api/v1/auth/session', {}, signal))
}

export async function completeSetup(input: {
  initToken: string
  username: string
  password: string
}) {
  return parseUser(await request('/api/v1/setup/complete', {
    method: 'POST',
    body: JSON.stringify(input),
  }))
}

export async function login(input: { username: string; password: string }) {
  return parseUser(await request('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify(input),
  }))
}

export async function logout() {
  await request('/api/v1/auth/logout', { method: 'POST' })
}

export async function getCluster(signal?: AbortSignal): Promise<ClusterSnapshot> {
  const value = requireRecord(await request('/api/v1/cluster', {}, signal))
  return {
    available: requireBoolean(value.available, 'available'),
    error: optionalString(value.error),
    hostname: requireString(value.hostname, 'hostname'),
    operatingSystem: requireString(value.operatingSystem, 'operatingSystem'),
    architecture: requireString(value.architecture, 'architecture'),
    kernelVersion: requireString(value.kernelVersion, 'kernelVersion'),
    dockerVersion: requireString(value.dockerVersion, 'dockerVersion'),
    dockerApiVersion: requireString(value.dockerApiVersion, 'dockerApiVersion'),
    swarmState: requireString(value.swarmState, 'swarmState'),
    nodeId: optionalString(value.nodeId) ?? '',
    managerStatus: optionalString(value.managerStatus),
    nodeRole: requireString(value.nodeRole, 'nodeRole'),
    nodeStatus: requireString(value.nodeStatus, 'nodeStatus'),
    availability: requireString(value.availability, 'availability'),
    managers: requireNumber(value.managers, 'managers'),
    nodes: requireNumber(value.nodes, 'nodes'),
    cpus: requireNumber(value.cpus, 'cpus'),
    memoryBytes: requireNumber(value.memoryBytes, 'memoryBytes'),
    containersRunning: requireNumber(value.containersRunning, 'containersRunning'),
    images: requireNumber(value.images, 'images'),
  }
}

export async function deployService(input: DeploymentInput): Promise<DeploymentResult> {
  const value = requireRecord(await request('/api/v1/deployments', {
    method: 'POST',
    body: JSON.stringify(input),
  }))
  if (!Array.isArray(value.warnings) || !value.warnings.every((item) => typeof item === 'string')) {
    throw new ApiError('API response field warnings is invalid.', 502)
  }
  return {
    serviceId: requireString(value.serviceId, 'serviceId'),
    image: requireString(value.image, 'image'),
    updated: requireBoolean(value.updated, 'updated'),
    warnings: value.warnings,
  }
}

function parseUser(input: unknown): User {
  const value = requireRecord(input)
  return {
    id: requireNumber(value.id, 'id'),
    username: requireString(value.username, 'username'),
    role: requireString(value.role, 'role'),
  }
}

function requireRecord(input: unknown): Record<string, unknown> {
  const value = optionalRecord(input)
  if (!value) {
    throw new ApiError('API response is not a JSON object.', 502)
  }
  return value
}

function optionalRecord(input: unknown): Record<string, unknown> | undefined {
  if (typeof input !== 'object' || input === null || Array.isArray(input)) return undefined
  return input as Record<string, unknown>
}

function requireString(input: unknown, field: string): string {
  const value = optionalString(input)
  if (value === undefined) throw new ApiError(`API response field ${field} is invalid.`, 502)
  return value
}

function optionalString(input: unknown): string | undefined {
  return typeof input === 'string' ? input : undefined
}

function requireNumber(input: unknown, field: string): number {
  if (typeof input !== 'number' || !Number.isFinite(input)) {
    throw new ApiError(`API response field ${field} is invalid.`, 502)
  }
  return input
}

function requireBoolean(input: unknown, field: string): boolean {
  if (typeof input !== 'boolean') {
    throw new ApiError(`API response field ${field} is invalid.`, 502)
  }
  return input
}
