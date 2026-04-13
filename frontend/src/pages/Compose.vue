<template>
    <transition name="slide-fade" appear>
        <div>
            <!-- Title -->
            <h1 v-if="isAdd" class="mb-3">{{ $t("compose") }}</h1>
            <h1 v-else class="mb-3 flex items-center gap-2">
                <Uptime :stack="globalStack" />
                {{ stack.name }}
                <span v-if="socketStore.agentCount > 1" class="text-sm text-[#575c62]">({{ endpoint }})</span>
            </h1>

            <!-- Action toolbar -->
            <ComposeActions
                :stack="stack"
                :is-edit-mode="isEditMode"
                :is-add="isAdd"
                :active="active"
                :processing="processing"
                @deploy="deployStack"
                @save="saveStack"
                @edit="isEditMode = true"
                @start="startStack"
                @stop="stopStack"
                @restart="restartStack"
                @update="updateStack"
                @down="downStack"
                @discard="discardStack"
                @delete="showDeleteDialog = true"
                :update-available="globalStack?.updateAvailable"
            />

            <!-- URLs -->
            <div v-if="urls.length > 0" class="mb-3 flex flex-wrap gap-2">
                <a v-for="(link, index) in urls" :key="index" target="_blank" :href="link.url">
                    <span class="badge bg-secondary">{{ link.display }}</span>
                </a>
            </div>

            <!-- Progress Terminal -->
            <transition name="slide-fade" appear>
                <Terminal
                    v-show="showProgressTerminal"
                    ref="progressTerminal"
                    class="mb-3 terminal"
                    :name="terminalName"
                    :endpoint="endpoint"
                    :rows="PROGRESS_TERMINAL_ROWS"
                    @has-data="showProgressTerminal = true; submitted = true"
                />
            </transition>

            <!-- Tab bar (narrow screens) -->
            <div v-if="stack.isManagedByDockge && layout.isNarrow" class="compose-tabs mb-4">
                <button :class="['compose-tab', activeTab === 'containers' ? 'active' : '']" @click="activeTab = 'containers'">
                    {{ $t("container", 2) }}
                </button>
                <button :class="['compose-tab', activeTab === 'compose' ? 'active' : '']" @click="activeTab = 'compose'">
                    Compose
                </button>
            </div>

            <!-- Two-column / tabbed layout -->
            <div v-if="stack.isManagedByDockge" class="flex flex-wrap gap-4">
                <!-- Left: containers + terminal -->
                <div :class="layout.isNarrow ? (activeTab === 'containers' ? 'w-full' : 'hidden') : 'w-full lg:w-[calc(50%-0.5rem)]'">
                    <ComposeContainers
                        ref="containersPanel"
                        :stack="stack"
                        :json-config="jsonConfig"
                        :envsubst-json-config="Object.keys(envsubstJSONConfig).length ? envsubstJSONConfig : undefined"
                        :service-status-list="serviceStatusList"
                        :update-details="updateDetails"
                        :is-edit-mode="isEditMode"
                        :is-add="isAdd"
                        :endpoint="endpoint"
                        :combined-terminal-name="combinedTerminalName"
                    />
                </div>

                <!-- Right: YAML editor + .env + networks -->
                <div :class="layout.isNarrow ? (activeTab === 'compose' ? 'w-full' : 'hidden') : 'w-full lg:w-[calc(50%-0.5rem)]'">
                    <ComposeEditor
                        :stack="stack"
                        :json-config="jsonConfig"
                        :is-edit-mode="isEditMode"
                        :endpoint="endpoint"
                        :editor-focus="editorFocus"
                        :yaml-error="yamlError"
                        :focus-effect-handler="focusEffectHandler"
                        @yaml-change="yamlCodeChange"
                    />
                </div>
            </div>

            <!-- Unmanaged stack message -->
            <div v-if="!stack.isManagedByDockge && !processing" class="shadow-box p-4">
                <p class="mb-3">{{ $t("stackNotManagedByDockgeMsg") }}</p>
                <button class="btn btn-danger" :disabled="processing" @click="downStack">{{ $t("cleanupStack") }}</button>
            </div>

            <!-- Delete confirmation dialog -->
            <TransitionRoot appear :show="showDeleteDialog" as="template">
                <HDialog as="div" class="relative z-50" @close="showDeleteDialog = false">
                    <TransitionChild as="template" enter="duration-200 ease-out" enter-from="opacity-0" enter-to="opacity-100" leave="duration-150 ease-in" leave-from="opacity-100" leave-to="opacity-0">
                        <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" />
                    </TransitionChild>
                    <div class="fixed inset-0 flex items-center justify-center p-4">
                        <TransitionChild as="template" enter="duration-200 ease-out" enter-from="opacity-0 scale-95" enter-to="opacity-100 scale-100" leave="duration-150 ease-in" leave-from="opacity-100 scale-100" leave-to="opacity-0 scale-95">
                            <HDialogPanel class="modal-content w-full max-w-md bg-white dark:bg-[#0d1117] rounded-2xl shadow-2xl p-6">
                                <p class="mb-6 text-sm text-gray-700 dark:text-[#b1b8c0]">{{ $t("deleteStackMsg") }}</p>
                                <div class="flex justify-end gap-2">
                                    <button class="btn btn-secondary" @click="showDeleteDialog = false">{{ $t("cancel") }}</button>
                                    <button class="btn btn-danger" @click="deleteStack">{{ $t("deleteStack") }}</button>
                                </div>
                            </HDialogPanel>
                        </TransitionChild>
                    </div>
                </HDialog>
            </TransitionRoot>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter, onBeforeRouteUpdate, onBeforeRouteLeave } from "vue-router";
import { useI18n } from "vue-i18n";
import { parseDocument, Document } from "yaml";
import dotenv from "dotenv";
import {
    copyYAMLComments, envsubstYAML,
    getCombinedTerminalName, getComposeTerminalName,
    PROGRESS_TERMINAL_ROWS, RUNNING,
} from "../utils";
import { Dialog as HDialog, DialogPanel as HDialogPanel, TransitionRoot, TransitionChild } from "@headlessui/vue";
import { useSocketStore } from "../stores/socket";
import { useLayoutStore } from "../stores/layout";
import { useToastHelper } from "../composables/useToastHelper";
import type { Stack, ComposeConfig, ServiceStatus, UpdateStatus, SocketResponse } from "../types";
import Uptime from "../components/Uptime.vue";
import Terminal from "../components/Terminal.vue";
import ComposeActions from "../components/compose/ComposeActions.vue";
import ComposeContainers from "../components/compose/ComposeContainers.vue";
import ComposeEditor from "../components/compose/ComposeEditor.vue";

const COMPOSE_TEMPLATE = `
services:
  nginx:
    image: nginx:latest
    restart: unless-stopped
    ports:
      - "8080:80"
`;
const ENV_DEFAULT = "# VARIABLE=value #comment";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const socketStore = useSocketStore();
const layout = useLayoutStore();
const toast = useToastHelper();

// Refs
const progressTerminal = ref<InstanceType<typeof Terminal> | null>(null);
const containersPanel = ref<InstanceType<typeof ComposeContainers> | null>(null);

// State — reactive with defaults so string fields are never undefined at runtime
const stack = reactive<Stack>({
    name: "",
    composeYAML: "",
    composeENV: "",
    endpoint: "",
});
const jsonConfig = reactive<ComposeConfig>({});
const envsubstJSONConfig = reactive<ComposeConfig>({});
const yamlError = ref("");
const processing = ref(true);
const showProgressTerminal = ref(false);
const activeTab = ref("containers");
const serviceStatusList = ref<Record<string, ServiceStatus>>({});
const updateDetails = ref<Record<string, UpdateStatus> | null>(null);
const isEditMode = ref(false);
const submitted = ref(false);
const showDeleteDialog = ref(false);

// Editor focus tracking
const editorFocus = ref(false);
const focusEffectHandler = (_state: unknown, focusing: boolean) => {
    editorFocus.value = focusing;
    return null;
};

// Timeouts
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let yamlDoc: any = null;
let yamlErrorTimeout: number | null = null;
let serviceStatusTimeout: number | null = null;
let stopServiceStatusTimeout = false;

// Computed
const isAdd = computed(() => route.path === "/compose" && !submitted.value);

const endpoint = computed(() => stack.endpoint || (route.params.endpoint as string) || "");

const globalStack = computed(() =>
    socketStore.completeStackList[stack.name + "_" + endpoint.value]
);

const status = computed(() => globalStack.value?.status);
const active = computed(() => status.value === RUNNING);

const terminalName = computed(() =>
    stack.name ? getComposeTerminalName(endpoint.value, stack.name) : ""
);

const combinedTerminalName = computed(() =>
    stack.name ? getCombinedTerminalName(endpoint.value, stack.name) : ""
);

const stackUrl = computed(() =>
    stack.endpoint ? `/compose/${stack.name}/${stack.endpoint}` : `/compose/${stack.name}`
);

const urls = computed(() => {
    const xDockge = envsubstJSONConfig["x-dockge"];
    if (!xDockge?.urls || !Array.isArray(xDockge.urls)) {
        return [];
    }
    return xDockge.urls.map((url: string) => {
        try {
            const obj = new URL(url);
            const pathname = obj.pathname === "/" ? "" : obj.pathname;
            return { display: obj.host + pathname + obj.search, url };
        } catch {
            return { display: url, url };
        }
    });
});

// Watchers
watch(() => stack.composeYAML, () => {
    if (editorFocus.value && stack.composeYAML) {
        yamlCodeChange();
    }
});
watch(() => stack.composeENV, () => {
    if (editorFocus.value && stack.composeYAML) {
        yamlCodeChange();
    }
});
watch(jsonConfig, () => {
    if (!editorFocus.value && stack.composeYAML) {
        const doc = new Document(jsonConfig);
        if (yamlDoc) {
            copyYAMLComments(doc, yamlDoc);
        }
        stack.composeYAML = doc.toString();
        yamlDoc = doc;
    }
}, { deep: true });

// Lifecycle
onMounted(() => {
    if (isAdd.value) {
        processing.value = false;
        isEditMode.value = true;
        const composeYAML = socketStore.composeTemplate || COMPOSE_TEMPLATE;
        const composeENV = socketStore.envTemplate || ENV_DEFAULT;
        socketStore.composeTemplate = "";
        socketStore.envTemplate = "";
        Object.assign(stack, { name: "", composeYAML, composeENV, isManagedByDockge: true, endpoint: "" });
        yamlCodeChange();
    } else {
        stack.name = route.params.stackName as string;
        loadStack();
    }
    requestServiceStatus();
});

onUnmounted(() => {
    stopServiceStatusTimeout = true;
    clearTimeout(serviceStatusTimeout ?? 0);
});

onBeforeRouteUpdate((_to, _from, next) => exitConfirm(next));
onBeforeRouteLeave((_to, _from, next) => exitConfirm(next));

// Methods
function yamlCodeChange() {
    try {
        const { config, doc } = yamlToJSON(stack.composeYAML ?? "");
        yamlDoc = doc;
        Object.assign(jsonConfig, config);
        // Remove keys not in new config
        for (const key of Object.keys(jsonConfig)) {
            if (!(key in config)) {
                delete jsonConfig[key];
            }
        }
        const env = dotenv.parse(stack.composeENV ?? "");
        const envYAML = envsubstYAML(stack.composeYAML ?? "", env);
        const envConfig = yamlToJSON(envYAML).config;
        Object.assign(envsubstJSONConfig, envConfig);
        for (const key of Object.keys(envsubstJSONConfig)) {
            if (!(key in envConfig)) {
                delete envsubstJSONConfig[key];
            }
        }
        clearTimeout(yamlErrorTimeout ?? 0);
        yamlError.value = "";
    } catch (e: unknown) {
        clearTimeout(yamlErrorTimeout ?? 0);
        const msg = e instanceof Error ? e.message : String(e);
        if (yamlError.value) {
            yamlError.value = msg;
        } else {
            yamlErrorTimeout = window.setTimeout(() => {
                yamlError.value = msg;
            }, 3000);
        }
    }
}

function yamlToJSON(yamlStr: string) {
    const doc = parseDocument(yamlStr);
    if (doc.errors.length > 0) {
        throw doc.errors[0];
    }
    const config = doc.toJS() ?? {};
    if (!config.services) {
        config.services = {};
    }
    if (Array.isArray(config.services) || typeof config.services !== "object") {
        throw new Error("Services must be an object");
    }
    return { config, doc };
}

function loadStack() {
    processing.value = true;
    socketStore.emitAgent(endpoint.value, "getStack", stack.name, (res: SocketResponse & { stack?: Stack }) => {
        if (res.ok && res.stack) {
            Object.assign(stack, res.stack);
            updateDetails.value = (res.stack as Stack & { updateDetails?: Record<string, UpdateStatus> }).updateDetails ?? null;
            yamlCodeChange();
            processing.value = false;
            bindTerminal();
        } else {
            toast.toastRes(res);
        }
    });
}

function bindTerminal() {
    progressTerminal.value?.bind(endpoint.value, terminalName.value);
}

function requestServiceStatus() {
    if (isAdd.value) {
        return;
    }
    socketStore.emitAgent(endpoint.value, "serviceStatusList", stack.name, (res: SocketResponse & { serviceStatusList?: Record<string, ServiceStatus> }) => {
        if (res.ok && res.serviceStatusList) {
            serviceStatusList.value = res.serviceStatusList;
        }
        if (!stopServiceStatusTimeout) {
            clearTimeout(serviceStatusTimeout ?? 0);
            serviceStatusTimeout = window.setTimeout(() => requestServiceStatus(), 5000);
        }
    });
}

function exitConfirm(next: (arg?: unknown) => void) {
    if (isEditMode.value) {
        if (confirm(t("confirmLeaveStack"))) {
            exitAction();
            next();
        } else {
            next(false);
        }
    } else {
        exitAction();
        next();
    }
}

function exitAction() {
    stopServiceStatusTimeout = true;
    clearTimeout(serviceStatusTimeout ?? 0);
    socketStore.emitAgent(endpoint.value, "leaveCombinedTerminal", stack.name, () => { });
}

function deployStack() {
    processing.value = true;
    if (!jsonConfig.services) {
        toast.toastError("No services found in compose.yaml");
        processing.value = false;
        return;
    }
    if (typeof jsonConfig.services !== "object") {
        toast.toastError("Services must be an object");
        processing.value = false;
        return;
    }

    const serviceNames = Object.keys(jsonConfig.services);
    if (!stack.name && serviceNames.length > 0) {
        const first = jsonConfig.services[serviceNames[0]];
        stack.name = first?.container_name ?? serviceNames[0];
    }

    bindTerminal();
    socketStore.emitAgent(stack.endpoint ?? "", "deployStack", stack.name, stack.composeYAML, stack.composeENV, isAdd.value, (res: SocketResponse) => {
        processing.value = false;
        toast.toastRes(res);
        if (res.ok) {
            isEditMode.value = false;
            router.push(stackUrl.value);
        }
    });
}

function saveStack() {
    processing.value = true;
    socketStore.emitAgent(stack.endpoint ?? "", "saveStack", stack.name, stack.composeYAML, stack.composeENV, isAdd.value, (res: SocketResponse) => {
        processing.value = false;
        toast.toastRes(res);
        if (res.ok) {
            isEditMode.value = false;
            router.push(stackUrl.value);
        }
    });
}

function startStack() {
    processing.value = true;
    socketStore.emitAgent(endpoint.value, "startStack", stack.name, (res: SocketResponse) => {
        processing.value = false;
        toast.toastRes(res);
        if (res.ok) {
            requestServiceStatus();
        }
    });
}

function stopStack() {
    processing.value = true;
    socketStore.emitAgent(endpoint.value, "stopStack", stack.name, (res: SocketResponse) => {
        processing.value = false;
        toast.toastRes(res);
        if (res.ok) {
            requestServiceStatus();
        }
    });
}

function restartStack() {
    processing.value = true;
    socketStore.emitAgent(endpoint.value, "restartStack", stack.name, (res: SocketResponse) => {
        processing.value = false;
        toast.toastRes(res);
        if (res.ok) {
            requestServiceStatus();
        }
    });
}

function updateStack() {
    processing.value = true;
    socketStore.emitAgent(endpoint.value, "updateStack", stack.name, (res: SocketResponse) => {
        processing.value = false;
        toast.toastRes(res);
        if (res.ok) {
            updateDetails.value = null;
            requestServiceStatus();
        }
    });
}

function downStack() {
    processing.value = true;
    socketStore.emitAgent(endpoint.value, "downStack", stack.name, (res: SocketResponse) => {
        processing.value = false;
        toast.toastRes(res);
        if (res.ok) {
            requestServiceStatus();
        }
    });
}

function deleteStack() {
    socketStore.emitAgent(endpoint.value, "deleteStack", stack.name, (res: SocketResponse) => {
        toast.toastRes(res);
        if (res.ok) {
            router.push("/");
        }
    });
}

function discardStack() {
    loadStack();
    isEditMode.value = false;
}
</script>

<style scoped></style>
