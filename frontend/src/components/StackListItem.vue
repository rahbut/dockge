<template>
    <router-link :to="url" :class="{ dim: !stack.isManagedByDockge }" class="item">
        <Uptime :stack="stack" :fixed-width="true" class="me-2" />
        <div class="title">
            <span>{{ stack.name }}</span>
            <span v-if="stack.updateAvailable === true" :title="$t('updateAvailable')">
                <ArrowUpCircleIcon :size="12" class="update-badge ml-1 inline" />
            </span>
            <div v-if="socketStore.agentCount > 1" class="endpoint">{{ endpointDisplay }}</div>
        </div>
    </router-link>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ArrowUpCircleIcon } from "lucide-vue-next";
import { useSocketStore } from "../stores/socket";
import Uptime from "./Uptime.vue";

import type { Stack } from "../types";

const props = defineProps<{ stack: Stack }>();
const socketStore = useSocketStore();

const endpointDisplay = computed(() =>
    props.stack.endpoint || "current endpoint"
);

const url = computed(() =>
    props.stack.endpoint
        ? `/compose/${props.stack.name}/${props.stack.endpoint}`
        : `/compose/${props.stack.name}`
);
</script>

<style scoped>
.item {
    text-decoration: none;
    display: flex;
    align-items: center;
    min-height: 52px;
    border-radius: 10px;
    transition: all ease-in-out 0.15s;
    width: 100%;
    padding: 5px 8px;
    &.disabled { opacity: 0.3; }
    &:hover { background-color: #e7faec; }
    &.active { background: linear-gradient(135deg, #cdf8f4, #d4f5e0); color: #0d1f1e; }
    .title { margin-top: -4px; }
    .endpoint { font-size: 12px; color: #575c62; }
}
.dim { opacity: 0.5; }
.update-badge { color: #f0ad4e; font-size: 0.75em; }
</style>
