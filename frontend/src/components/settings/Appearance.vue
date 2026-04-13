<template>
    <div>
        <div class="my-4">
            <label for="language" class="form-label">{{ $t("Language") }}</label>
            <select id="language" v-model="langStore.language" class="form-select">
                <option v-for="(lang, i) in availableLocales" :key="`Lang${i}`" :value="lang">
                    {{ localeLabel(lang) }}
                </option>
            </select>
        </div>

        <div class="my-4">
            <label class="form-label">{{ $t("Theme") }}</label>
            <div class="flex gap-2">
                <button
                    v-for="opt in themeOptions"
                    :key="opt.value"
                    class="btn btn-outline-primary"
                    :class="{ active: themeStore.userTheme === opt.value }"
                    @click="themeStore.userTheme = opt.value"
                >
                    {{ $t(opt.label) }}
                </button>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useThemeStore } from "../../stores/theme";
import { useLangStore } from "../../stores/lang";
import { useLocales } from "../../composables/useLocales";

const themeStore = useThemeStore();
const langStore = useLangStore();
const { availableLocales, localeLabel } = useLocales();

const themeOptions = [
    { value: "light", label: "Light" },
    { value: "dark", label: "Dark" },
    { value: "auto", label: "Auto" },
];
</script>
