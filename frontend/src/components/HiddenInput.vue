<template>
    <div class="input-group mb-3">
        <input
            ref="input"
            v-model="model"
            :type="visibility"
            class="form-control"
            :placeholder="placeholder"
            :maxlength="maxlength"
            :autocomplete="autocomplete"
            :required="required"
            :readonly="readonly"
        >
        <a v-if="visibility === 'password'" class="btn btn-outline-primary" @click="visibility = 'text'">
            <EyeIcon :size="16" />
        </a>
        <a v-if="visibility === 'text'" class="btn btn-outline-primary" @click="visibility = 'password'">
            <EyeOffIcon :size="16" />
        </a>
    </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { EyeIcon, EyeOffIcon } from "lucide-vue-next";

const props = defineProps<{
    modelValue?: string;
    placeholder?: string;
    maxlength?: number;
    autocomplete?: string;
    required?: boolean;
    readonly?: boolean;
}>();

const emit = defineEmits<{ (e: "update:modelValue", v: string): void }>();

const visibility = ref<"password" | "text">("password");

const model = computed({
    get: () => props.modelValue ?? "",
    set: (v) => emit("update:modelValue", v),
});
</script>
