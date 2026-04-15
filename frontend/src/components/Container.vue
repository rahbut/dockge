<template>
    <div class="shadow-box big-padding mb-3">
        <div class="flex gap-2">
            <div class="flex-1 min-w-0">
                <h4 class="truncate">{{ name }}</h4>
                <div class="image mb-2 text-sm text-gray-500 dark:text-gray-300">{{ imageName }}:{{ imageTag }}</div>
                <div v-if="!isEditMode" class="flex flex-wrap gap-1">
                    <span class="badge" :class="bgStyle">{{ status }}</span>
                    <span v-if="updateStatus?.updateAvailable" class="badge bg-warning" :title="$t('updateAvailable')">
                        <ArrowUpCircleIcon :size="12" class="mr-1 inline" />{{ $t("updateAvailable") }}
                    </span>
                    <span v-else-if="updateStatus?.error === 'registryError'" class="badge bg-secondary" :title="updateStatus.error">
                        {{ $t(updateStatus.error) || updateStatus.error }}
                    </span>
                    <a v-for="port in (ports ?? envsubstService.ports)" :key="port" :href="parsePort(port).url" target="_blank">
                        <span class="badge bg-secondary">{{ parsePort(port).display }}</span>
                    </a>
                </div>
            </div>
            <div class="flex items-center justify-end shrink-0">
                <router-link v-if="!isEditMode" class="btn btn-normal" :to="terminalRouteLink">
                    <TerminalIcon :size="14" class="mr-1" /> Bash
                </router-link>
            </div>
        </div>

        <div v-if="isEditMode" class="mt-2 flex gap-2">
            <button class="btn btn-normal" @click="showConfig = !showConfig">
                <PencilIcon :size="14" class="mr-1" /> {{ $t("Edit") }}
            </button>
            <button class="btn btn-danger" @click="remove">
                <Trash2Icon :size="14" class="mr-1" /> {{ $t("deleteContainer") }}
            </button>
        </div>

        <transition name="slide-fade" appear>
            <div v-if="isEditMode && showConfig" class="config mt-3">
                <!-- Image -->
                <div class="mb-4">
                    <label class="form-label">{{ $t("dockerImage") }}</label>
                    <div class="input-group mb-3">
                        <input v-model="service.image" class="form-control" list="image-datalist" />
                    </div>
                    <datalist id="image-datalist"><option value="louislam/uptime-kuma:1" /></datalist>
                </div>
                <!-- Ports -->
                <div class="mb-4">
                    <label class="form-label">{{ $t("port", 2) }}</label>
                    <ArrayInput name="ports" :display-name="$t('port')" placeholder="HOST:CONTAINER" :target="service" />
                </div>
                <!-- Volumes -->
                <div class="mb-4">
                    <label class="form-label">{{ $t("volume", 2) }}</label>
                    <ArrayInput name="volumes" :display-name="$t('volume')" placeholder="HOST:CONTAINER" :target="service" />
                </div>
                <!-- Restart Policy -->
                <div class="mb-4">
                    <label class="form-label">{{ $t("restartPolicy") }}</label>
                    <select v-model="service.restart" class="form-select">
                        <option value="always">{{ $t("restartPolicyAlways") }}</option>
                        <option value="unless-stopped">{{ $t("restartPolicyUnlessStopped") }}</option>
                        <option value="on-failure">{{ $t("restartPolicyOnFailure") }}</option>
                        <option value="no">{{ $t("restartPolicyNo") }}</option>
                    </select>
                </div>
                <!-- Environment Variables -->
                <div class="mb-4">
                    <label class="form-label">{{ $t("environmentVariable", 2) }}</label>
                    <ArrayInput name="environment" :display-name="$t('environmentVariable')" placeholder="KEY=VALUE" :target="service" />
                </div>
                <!-- Networks -->
                <div class="mb-4">
                    <label class="form-label">{{ $t("network", 2) }}</label>
                    <div v-if="networkList.length === 0 && service.networks && service.networks.length > 0" class="text-yellow-500 mb-3">
                        {{ $t("NoNetworksAvailable") }}
                    </div>
                    <ArraySelect name="networks" :display-name="$t('network')" placeholder="Network Name" :options="networkList" :target="service" />
                </div>
                <!-- Depends on -->
                <div class="mb-4">
                    <label class="form-label">{{ $t("dependsOn") }}</label>
                    <ArrayInput name="depends_on" :display-name="$t('dependsOn')" :placeholder="$t('containerName')" :target="service" />
                </div>
            </div>
        </transition>
    </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { ArrowUpCircleIcon, TerminalIcon, PencilIcon, Trash2Icon } from "lucide-vue-next";
import { parseDockerPort } from "../utils";
import { useSocketStore } from "../stores/socket";
import ArrayInput from "./ArrayInput.vue";
import ArraySelect from "./ArraySelect.vue";
import type { ComposeConfig, Service, UpdateStatus } from "../types";

const props = defineProps<{
    name: string;
    isEditMode?: boolean;
    first?: boolean;
    status?: string;
    ports?: string[] | null;
    updateStatus?: UpdateStatus & { error?: string };
    /** Passed down from Compose.vue — replaces $parent.$parent */
    jsonConfig: ComposeConfig;
    envsubstJSONConfig?: ComposeConfig;
    endpoint: string;
    stackName: string;
    primaryHostname?: string;
}>();

const socketStore = useSocketStore();
const showConfig = ref(false);

const service = computed((): Service => props.jsonConfig?.services?.[props.name] ?? {});
const envsubstService = computed((): Service => props.envsubstJSONConfig?.services?.[props.name] ?? {});

const networkList = computed(() => Object.keys(props.jsonConfig?.networks ?? {}));

const bgStyle = computed(() => {
    if (props.status === "running" || props.status === "healthy") {
        return "bg-primary";
    }
    if (props.status === "unhealthy") {
        return "bg-danger";
    }
    return "bg-secondary";
});

const imageName = computed(() => (envsubstService.value.image ?? "").split(":")[0]);
const imageTag = computed(() => {
    const tag = (envsubstService.value.image ?? "").split(":")[1];
    return tag ?? "latest";
});

const terminalRouteLink = computed(() => {
    if (props.endpoint) {
        return {
            name: "containerTerminalEndpoint",
            params: {
                endpoint: props.endpoint,
                stackName: props.stackName,
                serviceName: props.name,
                type: "bash",
            },
        };
    }
    return {
        name: "containerTerminal",
        params: {
            stackName: props.stackName,
            serviceName: props.name,
            type: "bash",
        },
    };
});

function parsePort(port: string) {
    const hostname = props.primaryHostname
        || (socketStore.info as Record<string, string>).primaryHostname
        || location.hostname;
    return parseDockerPort(port, hostname as string);
}

function remove() {
    if (props.jsonConfig.services) {
        // eslint-disable-next-line vue/no-mutating-props
        delete props.jsonConfig.services[props.name];
    }
}
</script>

<style scoped>
.image { font-size: 0.8rem; color: #6c757d; }
.dark .image { color: #9ca3af; }
</style>
