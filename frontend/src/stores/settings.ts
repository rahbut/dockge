import { defineStore } from "pinia";
import { ref } from "vue";
import { useSocketStore } from "./socket";
import { useToastHelper } from "../composables/useToastHelper";
import type { SocketResponse } from "../types";

export const useSettingsStore = defineStore("settings", () => {
    const settings = ref<Record<string, unknown>>({});
    const settingsLoaded = ref(false);
    const settingsError = ref<string | null>(null);

    function loadSettings() {
        const socketStore = useSocketStore();
        socketStore.getSocket().emit("getSettings", (res: SocketResponse & { data?: Record<string, unknown> }) => {
            if (res.data) {
                settings.value = res.data;
                settingsLoaded.value = true;
                settingsError.value = null;
            } else {
                settingsError.value = "Failed to load settings";
            }
        });
    }

    function saveSettings(callback?: () => void, currentPassword?: string) {
        const socketStore = useSocketStore();
        const toast = useToastHelper();
        socketStore.getSocket().emit("setSettings", settings.value, currentPassword, (res: SocketResponse) => {
            toast.toastRes(res);
            if (res.ok) {
                // Only reload on success to avoid overwriting local changes on error
                loadSettings();
                if (callback) {
                    callback();
                }
            }
        });
    }

    return { settings, settingsLoaded, settingsError, loadSettings, saveSettings };
});
