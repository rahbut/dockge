/**
 * Shared TypeScript interfaces for the Dockge frontend.
 */

// ---------------------------------------------------------------------------
// Socket
// ---------------------------------------------------------------------------

export interface SocketResponse {
    ok: boolean;
    msg?: string;
    msgi18n?: boolean;
}

// ---------------------------------------------------------------------------
// Stack / Compose
// ---------------------------------------------------------------------------

export interface Stack {
    name: string;
    composeYAML: string;
    composeENV: string;
    composeFileName?: string;
    isManagedByDockge?: boolean;
    endpoint?: string;
    status?: number;
    updateAvailable?: boolean;
    updateDetails?: Record<string, UpdateStatus>;
    primaryHostname?: string;
}

export interface Service {
    image?: string;
    container_name?: string;
    restart?: string;
    ports?: string[];
    volumes?: string[];
    environment?: string[];
    networks?: string[];
    depends_on?: string[];
    [key: string]: unknown;
}

export interface ComposeConfig {
    services?: Record<string, Service>;
    networks?: Record<string, NetworkConfig | null>;
    volumes?: Record<string, unknown>;
    "x-dockge"?: XDockgeConfig;
    [key: string]: unknown;
}

export interface NetworkConfig {
    external?: boolean;
    name?: string;
    driver?: string;
    [key: string]: unknown;
}

export interface XDockgeConfig {
    urls?: string[];
    [key: string]: unknown;
}

export interface UpdateStatus {
    updateAvailable?: boolean;
    error?: string;
}

// ---------------------------------------------------------------------------
// Service status (runtime)
// ---------------------------------------------------------------------------

export interface ServiceStatus {
    state: string;
    ports?: string[];
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

export interface AgentItem {
    url: string;
    username?: string;
}

// ---------------------------------------------------------------------------
// Docker
// ---------------------------------------------------------------------------

export interface DockerPort {
    url: string;
    display: string;
}
