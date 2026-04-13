<template>
    <div>
        <div v-if="settingsStore.settingsLoaded" class="my-4">
            <h5 class="settings-subheading mb-4">{{ $t("Automatic Update Checks") }}</h5>

            <div class="shadow-box big-padding mb-4">
                <div class="form-check form-switch mb-4">
                    <input
                        id="update-check-enabled"
                        v-model="enabled"
                        class="form-check-input"
                        type="checkbox"
                        @change="onEnabledChange"
                    />
                    <label class="form-check-label" for="update-check-enabled">
                        {{ $t("Check for updates daily") }}
                    </label>
                </div>

                <Transition name="slide-fade">
                    <div v-if="enabled" class="mb-2">
                        <label for="update-check-time" class="form-label">
                            {{ $t("Run at") }}
                        </label>
                        <div class="flex items-center gap-3">
                            <input
                                id="update-check-time"
                                v-model="checkTime"
                                type="time"
                                class="form-control"
                                style="max-width: 10rem"
                                @change="save"
                            />
                            <span class="text-sm text-gray-500 dark:text-gray-400">
                                {{ $t("serverLocalTime") }}
                            </span>
                        </div>
                    </div>
                </Transition>
            </div>

            <p class="text-sm text-gray-500 dark:text-gray-400">
                {{ $t("updateCheckDescription") }}
            </p>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from "vue";
import { useSettingsStore } from "../../stores/settings";

const settingsStore = useSettingsStore();

const DEFAULT_TIME = "02:00";

const enabled = ref(false);
const checkTime = ref(DEFAULT_TIME);

// Initialise from settings once loaded.
onMounted(() => {
    if (settingsStore.settingsLoaded) {
        loadFromSettings();
    }
});

watch(() => settingsStore.settingsLoaded, (loaded) => {
    if (loaded) {
        loadFromSettings();
    }
});

function loadFromSettings() {
    const stored = settingsStore.settings.updateCheckTime as string | undefined;
    if (stored) {
        enabled.value = true;
        checkTime.value = stored;
    } else {
        enabled.value = false;
        checkTime.value = DEFAULT_TIME;
    }
}

function onEnabledChange() {
    if (!enabled.value) {
        // Disabled — clear the setting.
        settingsStore.settings.updateCheckTime = "";
    } else {
        // Enabled — persist the current (or default) time immediately.
        settingsStore.settings.updateCheckTime = checkTime.value;
    }
    settingsStore.saveSettings();
}

function save() {
    if (enabled.value) {
        settingsStore.settings.updateCheckTime = checkTime.value;
        settingsStore.saveSettings();
    }
}
</script>
