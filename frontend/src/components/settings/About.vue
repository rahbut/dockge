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
        </div>
    </div>
</template>

<script setup lang="ts">
import { useSocketStore } from "../../stores/socket";
const socketStore = useSocketStore();
</script>
