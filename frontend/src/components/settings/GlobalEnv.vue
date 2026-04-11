<template>
    <div>
        <div v-if="settingsLoaded" class="my-4">
            <form class="my-4" autocomplete="off" @submit.prevent="saveGeneral">
                <div class="shadow-box mb-3 editor-box edit-mode">
                    <code-mirror
                        ref="editor"
                        v-model="settings.globalENV"
                        :extensions="extensionsEnv"
                        minimal
                        wrap="true"
                        :dark="$root.isDark"
                        tab="true"
                        :hasFocus="editorFocus"
                        @change="onChange"
                    />
                </div>

                <div class="my-4">
                    <!-- Save Button -->
                    <div>
                        <button class="btn btn-primary" type="submit">
                            {{ $t("Save") }}
                        </button>
                    </div>
                </div>
            </form>
        </div>
    </div>
</template>

<script>
import CodeMirror from "vue-codemirror6";
import { python } from "@codemirror/lang-python"; // good enough for .env key=value highlighting
import { dracula, solarizedLight } from "thememirror";
import { lineNumbers, EditorView } from "@codemirror/view";
import { ref } from "vue";

export default {
    name: "GlobalEnv",
    components: {
        CodeMirror,
    },

    setup() {
        const editorFocus = ref(false);

        const focusEffectHandler = (state, focusing) => {
            editorFocus.value = focusing;
            return null;
        };

        return { editorFocus,
            focusEffectHandler };
    },

    computed: {
        editorTheme() {
            return this.$root.isDark ? dracula : solarizedLight;
        },

        extensionsEnv() {
            return [
                this.editorTheme,
                python(),
                lineNumbers(),
                EditorView.focusChangeEffect.of(this.focusEffectHandler),
            ];
        },

        settings() {
            return this.$parent.$parent.$parent.settings;
        },
        saveSettings() {
            return this.$parent.$parent.$parent.saveSettings;
        },
        settingsLoaded() {
            return this.$parent.$parent.$parent.settingsLoaded;
        },
    },

    methods: {
        /** Save the settings */
        saveGeneral() {
            this.saveSettings();
        },

        onChange() {
            // hook for future live validation if desired
        },
    },
};
</script>

<style scoped>
.editor-box { font-family: 'JetBrains Mono', monospace; font-size: 14px; }
.dark .editor-box.edit-mode { background-color: #2c2f38 !important; }
</style>
