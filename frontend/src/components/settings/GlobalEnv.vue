<template>
    <div>
        <div v-if="settingsStore.settingsLoaded" class="my-4">
            <form class="my-4" autocomplete="off" @submit.prevent="settingsStore.saveSettings()">
                <div class="shadow-box mb-3 editor-box edit-mode">
                    <code-mirror
                        ref="editor"
                        v-model="(settingsStore.settings.globalENV as string)"
                        :extensions="extensionsEnv"
                        minimal
                        :wrap="true"
                        :dark="themeStore.isDark"
                        :tab="true"
                        :hasFocus="editorFocus"
                    />
                </div>
                <div class="my-4">
                    <button class="btn btn-primary" type="submit">{{ $t("Save") }}</button>
                </div>
            </form>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import CodeMirror from "vue-codemirror6";
import { python } from "@codemirror/lang-python";
import { dracula, solarizedLight } from "thememirror";
import { lineNumbers, EditorView } from "@codemirror/view";
import { useSettingsStore } from "../../stores/settings";
import { useThemeStore } from "../../stores/theme";

const settingsStore = useSettingsStore();
const themeStore = useThemeStore();

const editorFocus = ref(false);

const focusEffectHandler = (_state: unknown, focusing: boolean) => {
    editorFocus.value = focusing;
    return null;
};

const editorTheme = computed(() => themeStore.isDark ? dracula : solarizedLight);

const extensionsEnv = computed(() => [
    editorTheme.value,
    python(),
    lineNumbers(),
    EditorView.focusChangeEffect.of(focusEffectHandler),
]);
</script>

<style scoped>
.editor-box { font-family: 'JetBrains Mono', monospace; font-size: 14px; }
.dark .editor-box.edit-mode { background-color: #2c2f38 !important; }
</style>
