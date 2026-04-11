<template>
    <TransitionRoot appear :show="isOpen" as="template">
        <HDialog as="div" class="relative z-50" @close="close">
            <TransitionChild
                as="template"
                enter="duration-200 ease-out"
                enter-from="opacity-0"
                enter-to="opacity-100"
                leave="duration-150 ease-in"
                leave-from="opacity-100"
                leave-to="opacity-0"
            >
                <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" />
            </TransitionChild>

            <div class="fixed inset-0 flex items-center justify-center p-4">
                <TransitionChild
                    as="template"
                    enter="duration-200 ease-out"
                    enter-from="opacity-0 scale-95"
                    enter-to="opacity-100 scale-100"
                    leave="duration-150 ease-in"
                    leave-from="opacity-100 scale-100"
                    leave-to="opacity-0 scale-95"
                >
                    <HDialogPanel class="modal-content w-full max-w-md bg-white dark:bg-[#0d1117] rounded-2xl shadow-2xl p-6">
                        <div class="flex items-center justify-between mb-4">
                            <HDialogTitle class="text-lg font-semibold">
                                {{ title || $t("Confirm") }}
                            </HDialogTitle>
                            <button class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200" @click="close">
                                <XIcon :size="18" />
                            </button>
                        </div>

                        <div class="mb-6 text-sm text-gray-700 dark:text-[#b1b8c0]">
                            <slot />
                        </div>

                        <div class="flex justify-end gap-2">
                            <button class="btn btn-secondary" @click="no">
                                {{ noText }}
                            </button>
                            <button class="btn" :class="btnStyle" @click="yes">
                                {{ yesText }}
                            </button>
                        </div>
                    </HDialogPanel>
                </TransitionChild>
            </div>
        </HDialog>
    </TransitionRoot>
</template>

<script>
import {
    Dialog as HDialog,
    DialogPanel as HDialogPanel,
    DialogTitle as HDialogTitle,
    TransitionRoot,
    TransitionChild,
} from "@headlessui/vue";
import { XIcon } from "lucide-vue-next";

export default {
    components: {
        HDialog,
        HDialogPanel,
        HDialogTitle,
        TransitionRoot,
        TransitionChild,
        XIcon,
    },
    props: {
        btnStyle: {
            type: String,
            default: "btn-primary",
        },
        yesText: {
            type: String,
            default: "Yes",
        },
        noText: {
            type: String,
            default: "No",
        },
        title: {
            type: String,
            default: null,
        },
    },
    emits: [ "yes", "no" ],
    data: () => ({
        isOpen: false,
    }),
    methods: {
        show() {
            this.isOpen = true;
        },
        close() {
            this.isOpen = false;
        },
        yes() {
            this.isOpen = false;
            this.$emit("yes");
        },
        no() {
            this.isOpen = false;
            this.$emit("no");
        },
    },
};
</script>
