import { defineStore } from "pinia";
import { ref, reactive, computed } from "vue";
import { io, Socket } from "socket.io-client";
import { Terminal } from "@xterm/xterm";
import { AgentSocket } from "./agent-socket";
import { i18n } from "../i18n";
import type { Stack, AgentItem, SocketResponse } from "../types";

const terminalMap = new Map<string, Terminal>();
let socket: Socket;

export const useSocketStore = defineStore("socket", () => {
    const socketIO = reactive({
        token: null as string | null,
        firstConnect: true,
        connected: false,
        connectCount: 0,
        initedSocketIO: false,
        connectionErrorMsg: "",
        showReverseProxyGuide: true,
        connecting: false,
    });

    const info = ref<Record<string, unknown>>({});
    const stackList = ref<Record<string, Stack>>({});
    const allAgentStackList = ref<Record<string, { stackList: Record<string, Stack> }>>({});
    const agentStatusList = ref<Record<string, string>>({});
    const agentList = ref<Record<string, AgentItem>>({});
    const stackListLoading = ref(false);
    const composeTemplate = ref("");
    const envTemplate = ref("");

    const agentCount = computed(() => Object.keys(agentList.value).length);

    const completeStackList = computed(() => {
        const list: Record<string, Stack> = {};
        for (const name in stackList.value) {
            list[name + "_"] = stackList.value[name];
        }
        for (const endpoint in allAgentStackList.value) {
            const instance = allAgentStackList.value[endpoint];
            for (const name in instance.stackList) {
                list[name + "_" + endpoint] = instance.stackList[name];
            }
        }
        return list;
    });

    const frontendVersion = computed(() => {
        try {
            return FRONTEND_VERSION;
        } catch {
            return "";
        }
    });

    const isFrontendBackendVersionMatched = computed(() => {
        if (!(info.value as Record<string, string>).version) {
            return true;
        }
        return (info.value as Record<string, string>).version === frontendVersion.value;
    });

    /** Returns a display string for an endpoint — translates the "local" label. */
    function endpointDisplayFunction(endpoint: string): string {
        if (!endpoint) {
            return (i18n.global as unknown as { t: (key: string) => string }).t("currentEndpoint");
        }
        return endpoint;
    }

    function getSocket(): Socket {
        return socket;
    }

    function emitAgent(endpoint: string, eventName: string, ...args: unknown[]) {
        socket.emit("agent", endpoint, eventName, ...args);
    }

    function bindTerminal(endpoint: string, terminalName: string, terminal: Terminal) {
        emitAgent(endpoint, "terminalJoin", terminalName, (res: SocketResponse & { buffer?: string }) => {
            if (res.ok) {
                if (res.buffer) {
                    terminal.write(res.buffer);
                }
                terminalMap.set(terminalName, terminal);
            } else {
                console.warn("terminalJoin failed:", res.msg);
            }
        });
    }

    function unbindTerminal(terminalName: string) {
        terminalMap.delete(terminalName);
    }

    function afterLogin() {
        stackListLoading.value = true;
        emitAgent("", "requestStackList", () => {
            stackListLoading.value = false;
        });
    }

    function initSocketIO() {
        if (socketIO.initedSocketIO) {
            return;
        }
        socketIO.initedSocketIO = true;

        const url = location.protocol + "//" + location.host;

        const connectingTimeout = setTimeout(() => {
            socketIO.connecting = true;
        }, 1500);

        socket = io(url);

        const agentSocket = new AgentSocket();
        socket.on("agent", (eventName: unknown, ...args: unknown[]) => {
            agentSocket.call(eventName as string, ...args);
        });

        socket.on("connect", () => {
            console.log("Connected to the socket server");
            clearTimeout(connectingTimeout);
            socketIO.connecting = false;
            socketIO.connectCount++;
            socketIO.connected = true;
            socketIO.showReverseProxyGuide = false;
            agentStatusList.value[""] = "online";

            // Dynamic import to avoid circular dep: auth <-> socket
            import("./auth").then(({ useAuthStore }) => {
                const authStore = useAuthStore();
                const token = authStore.storage().getItem("token");
                if (token) {
                    if (token !== "autoLogin") {
                        authStore.loginByToken(token);
                    } else {
                        setTimeout(() => {
                            if (!authStore.loggedIn) {
                                authStore.allowLoginDialog = true;
                                authStore.storage().removeItem("token");
                            }
                        }, 5000);
                    }
                } else {
                    authStore.allowLoginDialog = true;
                }
            }).catch((err) => {
                console.error("Failed to load auth store on connect:", err);
            });

            socketIO.firstConnect = false;
        });

        socket.on("disconnect", () => {
            console.log("disconnect");
            socketIO.connectionErrorMsg = "Lost connection to the socket server. Reconnecting...";
            socketIO.connected = false;
            agentStatusList.value[""] = "offline";
        });

        socket.on("connect_error", (err: Error) => {
            console.error(`Failed to connect: ${err.message}`);
            socketIO.connectionErrorMsg = `Cannot connect to the socket server. [${err}] reconnecting...`;
            socketIO.showReverseProxyGuide = true;
            socketIO.connected = false;
            socketIO.firstConnect = false;
            socketIO.connecting = false;
        });

        socket.on("info", (data: Record<string, unknown>) => {
            info.value = data;
        });

        socket.on("autoLogin", () => {
            import("./auth").then(({ useAuthStore }) => {
                const authStore = useAuthStore();
                authStore.loggedIn = true;
                authStore.storage().setItem("token", "autoLogin");
                socketIO.token = "autoLogin";
                authStore.allowLoginDialog = false;
                afterLogin();
            }).catch((err) => {
                console.error("Failed to load auth store on autoLogin:", err);
            });
        });

        socket.on("setup", () => {
            import("../router").then(({ router }) => router.push("/setup"));
        });

        socket.on("refresh", () => {
            location.reload();
        });

        socket.on("stackStatusList", (res: SocketResponse & { stackStatusList?: Record<string, number> }) => {
            if (res.ok && res.stackStatusList) {
                for (const name in res.stackStatusList) {
                    const obj = stackList.value[name];
                    if (obj) {
                        obj.status = res.stackStatusList[name];
                    }
                }
            }
        });

        socket.on("agentStatus", (res: { endpoint: string; status: string }) => {
            agentStatusList.value[res.endpoint] = res.status;
        });

        socket.on("agentList", (res: SocketResponse & { agentList?: Record<string, AgentItem> }) => {
            if (res.ok && res.agentList) {
                agentList.value = res.agentList;
            }
        });

        agentSocket.on("terminalWrite", (...args: unknown[]) => {
            const [ terminalName, data ] = args as [string, string];
            terminalMap.get(terminalName)?.write(data);
        });

        agentSocket.on("stackList", (...args: unknown[]) => {
            const res = args[0] as SocketResponse & { endpoint?: string; stackList?: Record<string, Stack> };
            if (res.ok) {
                if (!res.endpoint) {
                    const incoming = res.stackList ?? {};
                    for (const key of Object.keys(stackList.value)) {
                        if (!(key in incoming)) {
                            delete stackList.value[key];
                        }
                    }
                    Object.assign(stackList.value, incoming);
                } else {
                    if (!allAgentStackList.value[res.endpoint]) {
                        allAgentStackList.value[res.endpoint] = { stackList: {} };
                    }
                    const incoming = res.stackList ?? {};
                    const target = allAgentStackList.value[res.endpoint].stackList;
                    for (const key of Object.keys(target)) {
                        if (!(key in incoming)) {
                            delete target[key];
                        }
                    }
                    Object.assign(target, incoming);
                }
            }
        });
    }

    return {
        socketIO,
        info,
        stackList,
        allAgentStackList,
        agentStatusList,
        agentList,
        stackListLoading,
        composeTemplate,
        envTemplate,
        agentCount,
        completeStackList,
        frontendVersion,
        isFrontendBackendVersionMatched,
        endpointDisplayFunction,
        getSocket,
        emitAgent,
        bindTerminal,
        unbindTerminal,
        afterLogin,
        initSocketIO,
    };
});
