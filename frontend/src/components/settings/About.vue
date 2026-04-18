<template>
    <div class="flex justify-center items-center">
        <div class="flex flex-col justify-center items-center my-16 mx-4">
            <object class="my-4" width="200" height="200" data="/icon.svg" />
            <div class="text-xl font-bold">Dockge</div>
            <div>{{ $t("Version") }}: {{ socketStore.info.version }}</div>
            <div class="text-sm text-[#cccccc] dark:text-[#333333]">{{ $t("Frontend Version") }}: {{ socketStore.frontendVersion }}</div>
            <div v-if="!socketStore.isFrontendBackendVersionMatched" class="alert alert-warning mt-4" role="alert">
                ⚠️ {{ $t("Frontend Version do not match backend version!") }}
            </div>

            <div v-if="selfStack" class="mt-6">
                <button
                    class="btn"
                    :class="selfStack.updateAvailable ? 'btn-warning' : 'btn-normal'"
                    :disabled="upgrading"
                    @click="upgradeDockge"
                >
                    <CloudDownloadIcon :size="14" class="mr-1" />
                    {{ upgrading ? $t("Upgrading…") : $t("Upgrade Dockge") }}
                </button>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { CloudDownloadIcon } from "lucide-vue-next";
import { useSocketStore } from "../../stores/socket";
import { useToastHelper } from "../../composables/useToastHelper";
import type { SocketResponse } from "../../types";

const socketStore = useSocketStore();
const toast = useToastHelper();

const upgrading = ref(false);

const selfStack = computed(() => {
    const name = socketStore.info.selfStackName as string | undefined;
    if (!name) {
        return null;
    }
    return socketStore.stackList[name] ?? null;
});

function upgradeDockge() {
    const name = socketStore.info.selfStackName as string | undefined;
    if (!name) {
        return;
    }
    upgrading.value = true;
    socketStore.emitAgent("", "updateStack", name, (res: SocketResponse) => {
        upgrading.value = false;
        toast.toastRes(res);
        // The container will restart shortly — socket.io reconnect handles the rest.
    });
}
</script>
