<template>
    <div>
        <div v-if="valid">
            <ul v-if="isArrayInited" class="list-group">
                <li v-for="(value, index) in array" :key="index" class="list-group-item">
                    <select v-model="array[index]" class="no-bg domain-input">
                        <option value="">{{ $t("Select a network...") }}</option>
                        <option v-for="option in options" :key="option" :value="option">{{ option }}</option>
                    </select>
                    <XIcon :size="14" class="action remove ml-2 mr-3 text-red-500" @click="remove(index as number)" />
                </li>
            </ul>
            <button class="btn btn-normal btn-sm mt-3" @click="addField">{{ $t("addListItem", [ displayName ]) }}</button>
        </div>
        <div v-else>Long syntax is not supported here. Please use the YAML editor.</div>
    </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { XIcon } from "lucide-vue-next";
import type { Service } from "../types";

const props = defineProps<{
    name: string;
    displayName: string;
    placeholder?: string;
    options: string[];
    /** The service object that owns the array */
    target: Service | Record<string, unknown>;
}>();

const array = computed(() => {
    const val = (props.target as Record<string, unknown>)[props.name];
    if (!val) {
        return [];
    }
    return val as string[];
});

const isArrayInited = computed(() => (props.target as Record<string, unknown>)[props.name] !== undefined);

const valid = computed(() => {
    if (!Array.isArray(array.value)) {
        return false;
    }
    for (const item of array.value) {
        if (typeof item === "object") {
            return false;
        }
    }
    return true;
});

function addField() {
    const target = props.target as Record<string, unknown>;
    if (!target[props.name]) {
        target[props.name] = [];
    }
    (target[props.name] as string[]).push("");
}

function remove(index: number) {
    array.value.splice(index, 1);
}
</script>

<style scoped>
.list-group { background-color: #070a10; }
.list-group li { display: flex; align-items: center; padding: 10px 0 10px 10px; }
</style>
