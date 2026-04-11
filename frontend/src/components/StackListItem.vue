<template>
    <router-link :to="url" :class="{ 'dim' : !stack.isManagedByDockge }" class="item">
        <Uptime :stack="stack" :fixed-width="true" class="me-2" />
        <div class="title">
            <span>{{ stackName }}</span>
            <ArrowUpCircleIcon v-if="stack.updateAvailable === true" :size="12" class="update-badge ml-1 inline" :title="$t('updateAvailable')" />
            <div v-if="$root.agentCount > 1" class="endpoint">{{ endpointDisplay }}</div>
        </div>
    </router-link>
</template>

<script>
import Uptime from "./Uptime.vue";
import { ArrowUpCircleIcon } from "lucide-vue-next";

export default {
    components: {
        Uptime,
        ArrowUpCircleIcon,
    },
    props: {
        /** Stack this represents */
        stack: {
            type: Object,
            default: null,
        },
    },
    computed: {
        endpointDisplay() {
            return this.$root.endpointDisplayFunction(this.stack.endpoint);
        },
        url() {
            if (this.stack.endpoint) {
                return `/compose/${this.stack.name}/${this.stack.endpoint}`;
            } else {
                return `/compose/${this.stack.name}`;
            }
        },
        stackName() {
            return this.stack.name;
        }
    },
};
</script>

<style scoped>

.item {
    text-decoration: none;
    display: flex;
    align-items: center;
    min-height: 52px;
    border-radius: 10px;
    transition: all ease-in-out 0.15s;
    width: 100%;
    padding: 5px 8px;
    &.disabled {
        opacity: 0.3;
    }
    &:hover {
        background-color: #e7faec;
    }
    &.active {
        background-color: #cdf8f4;
    }
    .title {
        margin-top: -4px;
    }
    .endpoint {
        font-size: 12px;
        color: #575c62;
    }
}

.dim {
    opacity: 0.5;
}

.update-badge {
    color: #f0ad4e;
    font-size: 0.75em;
}

</style>
