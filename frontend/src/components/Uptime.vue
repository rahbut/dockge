<template>
    <span :class="className">{{ statusName }}</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { statusColor, statusNameShort } from "../utils";

import type { Stack } from "../types";

const props = defineProps<{
    stack: Stack;
    fixedWidth?: boolean;
}>();

const { t } = useI18n();

const colorClassMap: Record<string, string> = {
    primary: "badge rounded-pill bg-primary",
    secondary: "badge rounded-pill bg-secondary",
    danger: "badge rounded-pill bg-danger",
    warning: "badge rounded-pill bg-warning",
    success: "badge rounded-pill bg-success",
};

const color = computed(() => statusColor(props.stack?.status ?? 0));
const statusName = computed(() => {
    const key = statusNameShort(props.stack?.status ?? 0);
    return key && key !== "?" ? t(key) : "—";
});
const className = computed(() => {
    const base = colorClassMap[color.value] ?? "badge rounded-pill bg-secondary";
    return props.fixedWidth ? base + " fixed-width" : base;
});
</script>

<style scoped>
.badge { min-width: 62px; text-align: center; justify-content: center; }
.fixed-width { width: 62px; overflow: hidden; text-overflow: ellipsis; }
</style>
