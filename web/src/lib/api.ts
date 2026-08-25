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
  desiredDockerVersion: string
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

export type ManagementAccess = {
  domain: string
  acmeEmail: string
}

export type NodeRole = 'worker' | 'manager'

export type SwarmNode = {
  id: string
  hostname: string
  role: NodeRole
  status: string
  availability: string
  managerStatus?: string
  address: string
  managerAddress?: string
  operatingSystem: string
  architecture: string
  dockerVersion: string
  desiredDockerVersion: string
  versionDrift: boolean
}

export type NodeEnrollment = {
  id: string
  requestedRole: NodeRole
  status: string
  hostname?: string
  operatingSystem?: string
  architecture?: string
  advertiseAddress?: string
  dataPathAddress?: string
  dockerVersion?: string
  nodeId?: string
  message?: string
  expiresAt: string
  createdBy: number
  createdAt: string
  updatedAt: string
}

export type NodeEnrollmentEvent = {
  id: number
  enrollmentId: string
  status: string
  message: string
  createdAt: string
}

export type NodeEnrollmentCommand = {
  enrollment: NodeEnrollment
  command: string
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
    desiredDockerVersion: requireString(value.desiredDockerVersion, 'desiredDockerVersion'),
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

export async function getManagementAccess(signal?: AbortSignal): Promise<ManagementAccess> {
  return parseManagementAccess(await request('/api/v1/management-access', {}, signal))
}

export async function configureManagementAccess(
  input: ManagementAccess,
): Promise<ManagementAccess> {
  return parseManagementAccess(await request('/api/v1/management-access', {
    method: 'PUT',
    body: JSON.stringify(input),
  }))
}

function parseManagementAccess(input: unknown): ManagementAccess {
  const value = requireRecord(input)
  return { domain: requireString(value.domain, 'domain'), acmeEmail: requireString(value.acmeEmail, 'acmeEmail') }
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

export async function getNodes(signal?: AbortSignal): Promise<SwarmNode[]> {
  const value = requireRecord(await request('/api/v1/nodes', {}, signal))
  if (!Array.isArray(value.nodes)) {
    throw new ApiError('API response field nodes is invalid.', 502)
  }
  return value.nodes.map(parseSwarmNode)
}

export async function getNodeEnrollments(signal?: AbortSignal): Promise<NodeEnrollment[]> {
  const value = requireRecord(await request('/api/v1/node-enrollments', {}, signal))
  if (!Array.isArray(value.enrollments)) {
    throw new ApiError('API response field enrollments is invalid.', 502)
  }
  return value.enrollments.map(parseNodeEnrollment)
}

export async function createNodeEnrollment(role: NodeRole): Promise<NodeEnrollmentCommand> {
  const value = requireRecord(await request('/api/v1/node-enrollments', {
    method: 'POST',
    body: JSON.stringify({ role }),
  }))
  return {
    enrollment: parseNodeEnrollment(value.enrollment),
    command: requireString(value.command, 'command'),
  }
}

export async function revokeNodeEnrollment(id: string): Promise<NodeEnrollment> {
  return parseNodeEnrollment(await request(`/api/v1/node-enrollments/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  }))
}

export function parseNodeEnrollmentEvent(input: unknown): NodeEnrollmentEvent {
  const value = requireRecord(input)
  return {
    id: requireNumber(value.id, 'id'),
    enrollmentId: requireString(value.enrollmentId, 'enrollmentId'),
    status: requireString(value.status, 'status'),
    message: requireString(value.message, 'message'),
    createdAt: requireTimestamp(value.createdAt, 'createdAt'),
  }
}

function parseSwarmNode(input: unknown): SwarmNode {
  const value = requireRecord(input)
  return {
    id: requireString(value.id, 'id'),
    hostname: requireString(value.hostname, 'hostname'),
    role: requireNodeRole(value.role),
    status: requireString(value.status, 'status'),
    availability: requireString(value.availability, 'availability'),
    managerStatus: optionalString(value.managerStatus),
    address: requireString(value.address, 'address'),
    managerAddress: optionalString(value.managerAddress),
    operatingSystem: requireString(value.operatingSystem, 'operatingSystem'),
    architecture: requireString(value.architecture, 'architecture'),
    dockerVersion: requireString(value.dockerVersion, 'dockerVersion'),
    desiredDockerVersion: requireString(value.desiredDockerVersion, 'desiredDockerVersion'),
    versionDrift: requireBoolean(value.versionDrift, 'versionDrift'),
  }
}

function parseNodeEnrollment(input: unknown): NodeEnrollment {
  const value = requireRecord(input)
  return {
    id: requireString(value.id, 'id'),
    requestedRole: requireNodeRole(value.requestedRole),
    status: requireString(value.status, 'status'),
    hostname: optionalString(value.hostname),
    operatingSystem: optionalString(value.operatingSystem),
    architecture: optionalString(value.architecture),
    advertiseAddress: optionalString(value.advertiseAddress),
    dataPathAddress: optionalString(value.dataPathAddress),
    dockerVersion: optionalString(value.dockerVersion),
    nodeId: optionalString(value.nodeId),
    message: optionalString(value.message),
    expiresAt: requireTimestamp(value.expiresAt, 'expiresAt'),
    createdBy: requireNumber(value.createdBy, 'createdBy'),
    createdAt: requireTimestamp(value.createdAt, 'createdAt'),
    updatedAt: requireTimestamp(value.updatedAt, 'updatedAt'),
  }
}

function requireNodeRole(input: unknown): NodeRole {
  if (input !== 'worker' && input !== 'manager') {
    throw new ApiError('API response field role is invalid.', 502)
  }
  return input
}

function requireTimestamp(input: unknown, field: string): string {
  const value = requireString(input, field)
  if (!Number.isFinite(Date.parse(value))) {
    throw new ApiError(`API response field ${field} is invalid.`, 502)
  }
  return value
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
