<template>
    <TransitionRoot appear :show="isOpen" as="template">
        <HDialog as="div" class="relative z-50" @close="close">
            <TransitionChild as="template" enter="duration-200 ease-out" enter-from="opacity-0" enter-to="opacity-100" leave="duration-150 ease-in" leave-from="opacity-100" leave-to="opacity-0">
                <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" />
            </TransitionChild>
            <div class="fixed inset-0 flex items-center justify-center p-4">
                <TransitionChild as="template" enter="duration-200 ease-out" enter-from="opacity-0 scale-95" enter-to="opacity-100 scale-100" leave="duration-150 ease-in" leave-from="opacity-100 scale-100" leave-to="opacity-0 scale-95">
                    <HDialogPanel class="modal-content w-full max-w-md bg-white dark:bg-[#0d1117] rounded-2xl shadow-2xl p-6">
                        <div class="flex items-center justify-between mb-4">
                            <HDialogTitle class="text-lg font-semibold">{{ title || $t("Confirm") }}</HDialogTitle>
                            <button class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200" @click="close"><XIcon :size="18" /></button>
                        </div>
                        <div class="mb-6 text-sm text-gray-700 dark:text-[#b1b8c0]"><slot /></div>
                        <div class="flex justify-end gap-2">
                            <button class="btn btn-secondary" @click="no">{{ noText }}</button>
                            <button class="btn" :class="btnStyle" @click="yes">{{ yesText }}</button>
                        </div>
                    </HDialogPanel>
                </TransitionChild>
            </div>
        </HDialog>
    </TransitionRoot>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Dialog as HDialog, DialogPanel as HDialogPanel, DialogTitle as HDialogTitle, TransitionRoot, TransitionChild } from "@headlessui/vue";
import { XIcon } from "lucide-vue-next";

defineProps<{
    btnStyle?: string;
    yesText?: string;
    noText?: string;
    title?: string;
}>();

const emit = defineEmits<{ (e: "yes"): void; (e: "no"): void }>();

const isOpen = ref(false);

function show() {
    isOpen.value = true;
}

function close() {
    isOpen.value = false;
}

function yes() {
    isOpen.value = false;
    emit("yes");
}

function no() {
    isOpen.value = false;
    emit("no");
}

defineExpose({ show, close });
</script>
