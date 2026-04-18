<!-- eslint-disable vue/no-mutating-props -->
<template>
    <div>
        <h4 class="mb-3">{{ stack.composeFileName }}</h4>

        <div class="shadow-box mb-3 editor-box" :class="{ 'edit-mode': isEditMode }">
            <code-mirror
                ref="yamlEditor"
                v-model="stack.composeYAML"
                :extensions="extensions"
                minimal
                :wrap="true"
                :dark="themeStore.isDark"
                :tab="true"
                :disabled="!isEditMode"
                :hasFocus="editorFocus"
                @change="emit('yaml-change')"
            />
        </div>
        <div v-if="isEditMode" class="mb-3">{{ yamlError }}</div>

        <div v-if="isEditMode">
            <h4 class="mb-3">.env</h4>
            <div class="shadow-box mb-3 editor-box" :class="{ 'edit-mode': isEditMode }">
                <code-mirror
                    ref="envEditor"
                    v-model="stack.composeENV"
                    :extensions="extensionsEnv"
                    minimal
                    :wrap="true"
                    :dark="themeStore.isDark"
                    :tab="true"
                    :disabled="!isEditMode"
                    :hasFocus="editorFocus"
                    @change="emit('yaml-change')"
                />
            </div>
        </div>

        <div v-if="isEditMode">
            <h4 class="mb-3">{{ $t("network", 2) }}</h4>
            <div class="shadow-box big-padding mb-3">
                <NetworkInput
                    :json-config="jsonConfig"
                    :stack="stack"
                    :endpoint="endpoint"
                    :editor-focus="editorFocus"
                />
            </div>
        </div>
    </div>
</template>

<!-- eslint-disable vue/no-mutating-props -->
<script setup lang="ts">
import { computed } from "vue";
import CodeMirror from "vue-codemirror6";
import { yaml } from "@codemirror/lang-yaml";
import { python } from "@codemirror/lang-python";
import { dracula, solarizedLight } from "thememirror";
import { lineNumbers, EditorView } from "@codemirror/view";
import { useThemeStore } from "../../stores/theme";
import NetworkInput from "../NetworkInput.vue";
import type { Stack, ComposeConfig } from "../../types";

const props = defineProps<{
    stack: Stack;
    jsonConfig: ComposeConfig;
    isEditMode: boolean;
    endpoint: string;
    editorFocus: boolean;
    yamlError: string;
    focusEffectHandler: (state: unknown, focusing: boolean) => null;
}>();

const emit = defineEmits<{ (e: "yaml-change"): void }>();

const themeStore = useThemeStore();

const editorTheme = computed(() => themeStore.isDark ? dracula : solarizedLight);

const extensions = computed(() => [
    editorTheme.value,
    yaml(),
    lineNumbers(),
    EditorView.focusChangeEffect.of(props.focusEffectHandler),
]);

const extensionsEnv = computed(() => [
    editorTheme.value,
    python(),
    lineNumbers(),
    EditorView.focusChangeEffect.of(props.focusEffectHandler),
]);
</script>

<style scoped>
.editor-box { font-family: var(--font-mono); font-size: 14px; }
</style>
