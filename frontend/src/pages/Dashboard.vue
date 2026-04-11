<template>
    <div class="px-2 w-[98%] mx-auto">
        <div class="flex flex-wrap gap-4">
            <div v-if="!$root.isMobile" class="w-full md:w-64 xl:w-72 shrink-0">
                <div class="mb-3">
                    <router-link to="/compose" class="btn btn-primary">
                        <PlusIcon :size="14" class="mr-1" /> {{ $t("compose") }}
                    </router-link>
                </div>
                <StackList :scrollbar="true" />
            </div>

            <div ref="container" class="flex-1 min-w-0 mb-3">
                <router-view :key="$route.fullPath" :calculatedHeight="height" />
            </div>
        </div>
    </div>
</template>

<script>
import StackList from "../components/StackList.vue";
import { PlusIcon } from "lucide-vue-next";

export default {
    components: {
        StackList,
        PlusIcon,
    },
    data() {
        return {
            height: 0
        };
    },
    mounted() {
        this.height = this.$refs.container.offsetHeight;
    },
};
</script>
