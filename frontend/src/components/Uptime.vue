<template>
    <span :class="className">{{ statusName }}</span>
</template>

<script>
import { statusColor, statusNameShort } from "../../../common/util-common";

// Explicit class map — Tailwind needs complete class names at build time
const colorClassMap = {
    primary: "badge rounded-pill bg-primary",
    secondary: "badge rounded-pill bg-secondary",
    danger: "badge rounded-pill bg-danger",
    warning: "badge rounded-pill bg-warning",
    success: "badge rounded-pill bg-success",
};

export default {
    props: {
        stack: {
            type: Object,
            default: null,
        },
        fixedWidth: {
            type: Boolean,
            default: false,
        },
    },

    computed: {
        color() {
            return statusColor(this.stack?.status);
        },

        statusName() {
            return this.$t(statusNameShort(this.stack?.status));
        },

        className() {
            const base = colorClassMap[this.color] ?? "badge rounded-pill bg-secondary";
            return this.fixedWidth ? base + " fixed-width" : base;
        },
    },
};
</script>

<style scoped>
.badge { min-width: 62px; }
.fixed-width { width: 62px; overflow: hidden; text-overflow: ellipsis; }
</style>
