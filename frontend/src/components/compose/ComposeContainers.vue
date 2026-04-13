<!-- eslint-disable vue/no-mutating-props -->
<template>
    <div>
        <!-- General (add mode only) -->
        <div v-if="isAdd">
            <h4 class="mb-3">{{ $t("general") }}</h4>
            <div class="shadow-box big-padding mb-3">
                <div>
                    <label for="name" class="form-label">{{ $t("stackName") }}</label>
                    <input id="name" v-model="stack.name" type="text" class="form-control" required @blur="stack.name = stack.name?.toLowerCase()" />
                    <div class="form-text">{{ $t("Lowercase only") }}</div>
                </div>
                <div class="mt-3">
                    <label class="form-label">{{ $t("dockgeAgent") }}</label>
                    <select v-model="stack.endpoint" class="form-select">
                        <option v-for="(agent, ep) in socketStore.agentList" :key="ep" :value="ep" :disabled="socketStore.agentStatusList[ep] !== 'online'">
                            ({{ socketStore.agentStatusList[ep] }}) {{ ep ? ep : $t("currentEndpoint") }}
                        </option>
                    </select>
                </div>
            </div>
        </div>

        <!-- Container list -->
        <h4 class="mb-3">{{ $t("container", 2) }}</h4>

        <div v-if="isEditMode" class="input-group mb-3">
            <input v-model="newContainerName" :placeholder="$t('New Container Name...')" class="form-control" @keyup.enter="addContainer" />
            <button class="btn btn-primary" @click="addContainer">{{ $t("addContainer") }}</button>
        </div>

        <div ref="containerListEl">
            <Container
                v-for="(service, name) in (jsonConfig.services ?? {})"
                :key="String(name)"
                :name="String(name)"
                :is-edit-mode="isEditMode"
                :first="String(name) === Object.keys(jsonConfig.services ?? {})[0]"
                :status="serviceStatusList[String(name)]?.state"
                :ports="serviceStatusList[String(name)]?.ports"
                :update-status="updateDetails ? updateDetails[String(name)] : undefined"
                :json-config="jsonConfig"
                :envsubst-json-config="envsubstJSONConfig ?? {}"
                :endpoint="endpoint"
                :stack-name="stack.name ?? ''"
            />
        </div>

        <!-- Extra URLs (edit mode) -->
        <div v-if="isEditMode">
            <h4 class="mb-3">{{ $t("extra") }}</h4>
            <div class="shadow-box big-padding mb-3">
                <div class="mb-4">
                    <label class="form-label">{{ $t("url", 2) }}</label>
                    <ArrayInput
                        name="urls"
                        :display-name="$t('url')"
                        placeholder="https://"
                        object-type="x-dockge"
                        :target="xDockgeTarget"
                        :json-config="jsonConfig"
                    />
                </div>
            </div>
        </div>

        <!-- Combined Terminal (view mode) -->
        <div v-show="!isEditMode">
            <h4 class="mb-3">{{ $t("terminal") }}</h4>
            <Terminal
                ref="combinedTerminalRef"
                class="mb-3 terminal"
                :name="combinedTerminalName"
                :endpoint="endpoint"
                :rows="combinedTerminalRows"
                :cols="combinedTerminalCols"
            />
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { COMBINED_TERMINAL_ROWS, COMBINED_TERMINAL_COLS } from "../../utils";
import { useSocketStore } from "../../stores/socket";
import { useToastHelper } from "../../composables/useToastHelper";
import Container from "../Container.vue";
import Terminal from "../Terminal.vue";
import ArrayInput from "../ArrayInput.vue";
import type { Stack, ComposeConfig, ComposeConfig as EnvsubstConfig, ServiceStatus, UpdateStatus } from "../../types";

const props = defineProps<{
    stack: Stack;
    jsonConfig: ComposeConfig;
    envsubstJSONConfig?: EnvsubstConfig;
    serviceStatusList: Record<string, ServiceStatus>;
    updateDetails: Record<string, UpdateStatus> | null;
    isEditMode: boolean;
    isAdd: boolean;
    endpoint: string;
    combinedTerminalName: string;
}>();

const socketStore = useSocketStore();
const toast = useToastHelper();

const newContainerName = ref("");
const containerListEl = ref<HTMLElement | null>(null);
const combinedTerminalRef = ref<InstanceType<typeof Terminal> | null>(null);

const combinedTerminalRows = COMBINED_TERMINAL_ROWS;
const combinedTerminalCols = COMBINED_TERMINAL_COLS;

function getXDockgeTarget() {
    if (!props.jsonConfig["x-dockge"]) {
        // eslint-disable-next-line vue/no-mutating-props
        props.jsonConfig["x-dockge"] = {};
    }
    return props.jsonConfig["x-dockge"];
}

const xDockgeTarget = computed(() => getXDockgeTarget());

function addContainer() {
    if (!newContainerName.value) {
        toast.toastError("Container name cannot be empty");
        return;
    }
    if (props.jsonConfig.services?.[newContainerName.value]) {
        toast.toastError("Container name already exists");
        return;
    }
    if (!props.jsonConfig.services) {
        // eslint-disable-next-line vue/no-mutating-props
        props.jsonConfig.services = {};
    }
    // eslint-disable-next-line vue/no-mutating-props
    props.jsonConfig.services[newContainerName.value] = { restart: "unless-stopped" };
    newContainerName.value = "";
    const last = containerListEl.value?.lastElementChild;
    last?.scrollIntoView({ block: "start", behavior: "smooth" });
}

defineExpose({ combinedTerminalRef });
</script>
