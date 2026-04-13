<template>
    <transition name="slide-fade" appear>
        <div v-if="!processing">
            <h1 class="mb-3">{{ $t("console") }}</h1>
            <Terminal v-if="enableConsole" class="terminal" :rows="20" mode="mainTerminal" name="console" :endpoint="endpoint" />
            <div v-else class="alert alert-warning shadow-box" role="alert">
                <h4 class="alert-heading font-semibold mb-2">{{ $t("Console is not enabled") }}</h4>
                <!-- eslint-disable vue/no-v-html -->
                <p v-html="$t('ConsoleNotEnabledMSG1')"></p>
                <p v-html="$t('ConsoleNotEnabledMSG2')"></p>
                <p v-html="$t('ConsoleNotEnabledMSG3')"></p>
                <!-- eslint-enable vue/no-v-html -->
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRoute } from "vue-router";
import Terminal from "../components/Terminal.vue";
import { useSocketStore } from "../stores/socket";

const route = useRoute();
const socketStore = useSocketStore();

const processing = ref(true);
const enableConsole = ref(false);
const endpoint = computed(() => (route.params.endpoint as string) || "");

onMounted(() => {
    socketStore.emitAgent(endpoint.value, "checkMainTerminal", (res: import("../types").SocketResponse) => {
        enableConsole.value = res.ok;
        processing.value = false;
    });
});
</script>

<style scoped>
.terminal { height: 410px; }
</style>
