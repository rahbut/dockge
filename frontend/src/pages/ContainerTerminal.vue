<template>
    <transition name="slide-fade" appear>
        <div>
            <h1 class="mb-3">{{ $t("terminal") }} - {{ serviceName }} ({{ stackName }})</h1>
            <div class="mb-3">
                <router-link :to="shRoute" class="btn btn-normal mr-2">{{ $t("Switch to sh") }}</router-link>
            </div>
            <Terminal class="terminal" :rows="20" mode="interactive" :name="terminalName" :stack-name="stackName" :service-name="serviceName" :shell="shell" :endpoint="endpoint" />
        </div>
    </transition>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { getContainerExecTerminalName } from "../utils";
import Terminal from "../components/Terminal.vue";

const route = useRoute();

const stackName = computed(() => route.params.stackName as string);
const endpoint = computed(() => (route.params.endpoint as string) || "");
const shell = computed(() => route.params.type as string);
const serviceName = computed(() => route.params.serviceName as string);
const terminalName = computed(() => getContainerExecTerminalName(endpoint.value, stackName.value, serviceName.value, 0));

const shRoute = computed(() => {
    const ep = route.params.endpoint as string;
    if (ep) {
        return { name: "containerTerminalEndpoint", params: { endpoint: ep, stackName: stackName.value, serviceName: serviceName.value, type: "sh" } };
    }
    return { name: "containerTerminal", params: { stackName: stackName.value, serviceName: serviceName.value, type: "sh" } };
});
</script>

<style scoped>
.terminal { height: 410px; }
</style>
