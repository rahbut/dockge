<template>
    <transition name="slide-fade" appear>
        <div v-if="!processing">
            <h1 class="mb-3">{{ $t("console") }}</h1>

            <Terminal v-if="enableConsole" class="terminal" :rows="20" mode="mainTerminal" name="console" :endpoint="endpoint"></Terminal>

            <!-- eslint-disable vue/no-v-html -->
            <div v-else class="alert alert-warning shadow-box" role="alert">
                <h4 class="alert-heading font-semibold mb-2">{{ $t("Console is not enabled") }}</h4>
                <p v-html="$t('ConsoleNotEnabledMSG1')"></p>
                <p v-html="$t('ConsoleNotEnabledMSG2')"></p>
                <p v-html="$t('ConsoleNotEnabledMSG3')"></p>
            </div>
            <!-- eslint-enable vue/no-v-html -->
        </div>
    </transition>
</template>

<script>
import Terminal from "../components/Terminal.vue";

export default {
    components: {
        Terminal,
    },
    data() {
        return {
            processing: true,
            enableConsole: false,
        };
    },
    computed: {
        endpoint() {
            return this.$route.params.endpoint || "";
        },
    },
    mounted() {
        this.$root.emitAgent(this.endpoint, "checkMainTerminal", (res) => {
            this.enableConsole = res.ok;
            this.processing = false;
        });
    },
    methods: {

    }
};
</script>

<style scoped>
.terminal { height: 410px; }
</style>
