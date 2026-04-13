<template>
    <div>
        <h5>{{ $t("Internal Networks") }}</h5>
        <ul class="list-group">
            <li v-for="(networkRow, index) in networkList" :key="index" class="list-group-item">
                <input v-model="networkRow.key" type="text" class="no-bg domain-input" :placeholder="$t('Network name...')" />
                <XIcon :size="14" class="action remove ml-2 mr-3 text-red-500" @click="remove(index)" />
            </li>
        </ul>
        <button class="btn btn-normal btn-sm mt-3 me-2" @click="addField">{{ $t("addInternalNetwork") }}</button>

        <h5 class="mt-3">{{ $t("External Networks") }}</h5>
        <div v-if="externalNetworkList.length === 0">{{ $t("No External Networks") }}</div>
        <div v-for="(networkName, index) in externalNetworkList" :key="networkName" class="form-check form-switch my-3">
            <input :id="'external-network' + index" v-model="selectedExternalList[networkName]" class="form-check-input" type="checkbox" />
            <label class="form-check-label" :for="'external-network' + index">{{ networkName }}</label>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from "vue";
import { XIcon } from "lucide-vue-next";
import { useSocketStore } from "../stores/socket";
import { useToastHelper } from "../composables/useToastHelper";
import type { ComposeConfig, NetworkConfig, SocketResponse, Stack } from "../types";

type NetworkRow = { key: string; value: NetworkConfig | null };

const props = defineProps<{
    jsonConfig: ComposeConfig;
    stack: Stack;
    endpoint: string;
    editorFocus: boolean;
}>();

const socketStore = useSocketStore();
const toast = useToastHelper();

const networkList = ref<NetworkRow[]>([]);
const externalList = reactive<Record<string, NetworkConfig>>({});
const selectedExternalList = reactive<Record<string, boolean>>({});
const externalNetworkList = ref<string[]>([]);

watch(() => props.jsonConfig.networks, () => {
    if (props.editorFocus) {
        loadNetworkList();
    }
}, { deep: true });

watch(selectedExternalList, () => {
    for (const networkName in selectedExternalList) {
        const enable = selectedExternalList[networkName];
        if (enable) {
            if (!externalList[networkName]) {
                externalList[networkName] = {};
            }
            externalList[networkName].external = true;
        } else {
            delete externalList[networkName];
        }
    }
    applyToYAML();
}, { deep: true });

watch(networkList, () => {
    applyToYAML();
}, { deep: true });

onMounted(() => {
    loadNetworkList();
    loadExternalNetworkList();
});

function loadNetworkList() {
    networkList.value = [];
    Object.keys(externalList).forEach(k => {
        delete externalList[k];
    });

    for (const key in props.jsonConfig.networks) {
        const value = props.jsonConfig.networks[key];
        const obj: NetworkRow = { key, value };
        if (value?.external) {
            externalList[key] = { ...value };
        } else {
            networkList.value.push(obj);
        }
    }

    Object.keys(selectedExternalList).forEach(k => {
        delete selectedExternalList[k];
    });
    for (const name in externalList) {
        selectedExternalList[name] = true;
    }
}

function loadExternalNetworkList() {
    socketStore.emitAgent(props.endpoint, "getDockerNetworkList", (res: SocketResponse & { dockerNetworkList?: string[] }) => {
        if (res.ok && res.dockerNetworkList) {
            externalNetworkList.value = res.dockerNetworkList.filter((n: string) => {
                if (n.startsWith(props.stack.name + "_")) {
                    return false;
                }
                if ([ "none", "host", "bridge" ].includes(n)) {
                    return false;
                }
                return true;
            });
        } else {
            toast.toastRes(res);
        }
    });
}

function addField() {
    networkList.value.push({ key: "", value: {} });
}

function remove(index: number) {
    networkList.value.splice(index, 1);
    applyToYAML();
}

function applyToYAML() {
    if (props.editorFocus) {
        return;
    }
    // eslint-disable-next-line vue/no-mutating-props
    props.jsonConfig.networks = {};
    for (const row of networkList.value) {
        // eslint-disable-next-line vue/no-mutating-props
        props.jsonConfig.networks[row.key] = row.value;
    }
    for (const name in externalList) {
        // eslint-disable-next-line vue/no-mutating-props
        props.jsonConfig.networks[name] = externalList[name];
    }
}
</script>

<style scoped>
.list-group { background-color: #070a10; }
.list-group li { display: flex; align-items: center; padding: 10px 0 10px 10px; }
</style>
