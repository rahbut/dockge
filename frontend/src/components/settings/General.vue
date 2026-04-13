<template>
    <div>
        <form class="my-4" autocomplete="off" @submit.prevent="saveGeneral">
            <div class="mb-4">
                <label for="language" class="form-label">{{ $t("Language") }}</label>
                <select id="language" v-model="langStore.language" class="form-select">
                    <option v-for="(lang, i) in availableLocales" :key="`Lang${i}`" :value="lang">
                        {{ localeLabel(lang) }}
                    </option>
                </select>
            </div>

            <div class="mb-4">
                <label class="form-label" for="primaryBaseURL">{{ $t("primaryHostname") }}</label>
                <div class="input-group mb-3">
                    <input v-model="settingsStore.settings.primaryHostname" class="form-control" :placeholder="$t('CurrentHostname')" />
                    <button class="btn btn-outline-primary" type="button" @click="autoGetHostname">
                        {{ $t("autoGet") }}
                    </button>
                </div>
            </div>

            <div>
                <button class="btn btn-primary" type="submit">{{ $t("Save") }}</button>
            </div>
        </form>
    </div>
</template>

<script setup lang="ts">
import { useSettingsStore } from "../../stores/settings";
import { useLangStore } from "../../stores/lang";
import { useLocales } from "../../composables/useLocales";

const settingsStore = useSettingsStore();
const langStore = useLangStore();
const { availableLocales, localeLabel } = useLocales();

function saveGeneral() {
    settingsStore.saveSettings();
}

function autoGetHostname() {
    settingsStore.settings.primaryHostname = location.hostname;
}
</script>
