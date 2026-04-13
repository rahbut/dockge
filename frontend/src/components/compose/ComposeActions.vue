<template>
    <div v-if="stack.isManagedByDockge" class="mb-3 flex flex-wrap gap-2">
        <div class="btn-group">
            <button v-if="isEditMode" class="btn btn-primary" :disabled="processing" @click="emit('deploy')">
                <RocketIcon :size="14" class="mr-1" /> {{ $t("deployStack") }}
            </button>
            <button v-if="isEditMode" class="btn btn-normal" :disabled="processing" @click="emit('save')">
                <SaveIcon :size="14" class="mr-1" /> {{ $t("saveStackDraft") }}
            </button>
            <button v-if="!isEditMode" class="btn btn-secondary" :disabled="processing" @click="emit('edit')">
                <PenIcon :size="14" class="mr-1" /> {{ $t("editStack") }}
            </button>
            <button v-if="!isEditMode && !active" class="btn btn-primary" :disabled="processing" @click="emit('start')">
                <PlayIcon :size="14" class="mr-1" /> {{ $t("startStack") }}
            </button>
            <button v-if="!isEditMode && active" class="btn btn-normal" :disabled="processing" @click="emit('restart')">
                <RotateCwIcon :size="14" class="mr-1" /> {{ $t("restartStack") }}
            </button>
            <button v-if="!isEditMode" class="btn" :class="stack.updateAvailable || updateAvailable ? 'btn-warning' : 'btn-normal'" :disabled="processing" @click="emit('update')">
                <CloudDownloadIcon :size="14" class="mr-1" /> {{ $t("updateStack") }}
            </button>
            <button v-if="!isEditMode && active" class="btn btn-normal" :disabled="processing" @click="emit('stop')">
                <SquareIcon :size="14" class="mr-1" /> {{ $t("stopStack") }}
            </button>

            <HMenu as="div" class="relative flex items-stretch">
                <HMenuButton class="btn btn-normal rounded-l-none px-3"><ChevronDownIcon :size="14" /></HMenuButton>
                <transition enter-active-class="transition duration-100 ease-out" enter-from-class="transform scale-95 opacity-0" enter-to-class="transform scale-100 opacity-1" leave-active-class="transition duration-75 ease-in" leave-from-class="transform scale-100 opacity-1" leave-to-class="transform scale-95 opacity-0">
                    <HMenuItems class="absolute right-0 mt-1 w-44 origin-top-right rounded-xl overflow-hidden shadow-xl bg-white dark:bg-[#0d1117] border border-gray-100 dark:border-[#1d2634] z-50 focus:outline-none">
                        <HMenuItem v-slot="{ active: itemActive }">
                            <button class="w-full text-left flex items-center gap-2 px-4 py-2 text-sm" :class="itemActive ? 'bg-gray-50 dark:bg-[#070a10]' : ''" @click="emit('down')">
                                <SquareIcon :size="13" /> {{ $t("downStack") }}
                            </button>
                        </HMenuItem>
                    </HMenuItems>
                </transition>
            </HMenu>
        </div>

        <button v-if="isEditMode && !isAdd" class="btn btn-normal" :disabled="processing" @click="emit('discard')">{{ $t("discardStack") }}</button>
        <button v-if="!isEditMode" class="btn btn-danger" :disabled="processing" @click="emit('delete')">
            <Trash2Icon :size="14" class="mr-1" /> {{ $t("deleteStack") }}
        </button>
    </div>
</template>

<script setup lang="ts">
import { Menu as HMenu, MenuButton as HMenuButton, MenuItems as HMenuItems, MenuItem as HMenuItem } from "@headlessui/vue";
import { RocketIcon, SaveIcon, PenIcon, PlayIcon, RotateCwIcon, CloudDownloadIcon, SquareIcon, Trash2Icon, ChevronDownIcon } from "lucide-vue-next";

import type { Stack } from "../../types";

defineProps<{
    stack: Stack;
    isEditMode: boolean;
    isAdd: boolean;
    active: boolean;
    processing: boolean;
    updateAvailable?: boolean;
}>();

const emit = defineEmits<{
    (e: "deploy"): void;
    (e: "save"): void;
    (e: "edit"): void;
    (e: "start"): void;
    (e: "stop"): void;
    (e: "restart"): void;
    (e: "update"): void;
    (e: "down"): void;
    (e: "discard"): void;
    (e: "delete"): void;
}>();
</script>
