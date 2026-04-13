<template>
    <div class="shadow-box mb-3" :style="boxStyle">
        <div class="list-header">
            <div class="header-top">
                <div class="placeholder"></div>
                <div class="search-wrapper">
                    <a v-if="searchText === ''" class="search-icon"><SearchIcon :size="16" /></a>
                    <a v-else class="search-icon cursor-pointer" @click="searchText = ''"><XIcon :size="16" /></a>
                    <form><input v-model="searchText" class="form-control search-input" autocomplete="off" /></form>
                </div>
            </div>
        </div>
        <div ref="stackListEl" class="stack-list" :class="{ scrollbar }" :style="stackListStyle">
            <template v-if="socketStore.stackListLoading">
                <div v-for="n in 4" :key="n" class="stack-skeleton" />
            </template>
            <div v-else-if="Object.keys(sortedStackList).length === 0" class="text-center mt-3">
                <router-link to="/compose">{{ $t("addFirstStackMsg") }}</router-link>
            </div>
            <template v-else>
                <StackListItem v-for="(item, index) in sortedStackList" :key="index" :stack="item" />
            </template>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { SearchIcon, XIcon } from "lucide-vue-next";
import { CREATED_FILE, CREATED_STACK, EXITED, RUNNING, UNKNOWN } from "../utils";
import { useSocketStore } from "../stores/socket";
import StackListItem from "./StackListItem.vue";

const props = defineProps<{
    scrollbar?: boolean;
    inline?: boolean;
}>();

const socketStore = useSocketStore();
const searchText = ref("");
const windowTop = ref(0);
const stackListEl = ref<HTMLElement | null>(null);

const boxStyle = computed(() => {
    if (props.inline) {
        return {};
    }
    return window.innerWidth > 550
        ? { height: `calc(100vh - 160px + ${windowTop.value}px)` }
        : { height: "calc(100vh - 160px)" };
});

const stackListStyle = computed(() => ({ height: "calc(100% - 60px)" }));

const sortedStackList = computed(() => {
    const result = Object.values(socketStore.completeStackList).filter((stack) => {
        return !searchText.value || stack.name.toLowerCase().includes(searchText.value.toLowerCase());
    });

    result.sort((m1, m2) => {
        if (m1.status !== m2.status) {
            for (const s of [ RUNNING, EXITED, CREATED_STACK, CREATED_FILE, UNKNOWN ]) {
                if (m2.status === s) {
                    return 1;
                }
                if (m1.status === s) {
                    return -1;
                }
            }
        }
        return m1.name.localeCompare(m2.name);
    });
    return result;
});

function onScroll() {
    windowTop.value = Math.min(window.scrollY, 133);
}

// Mount/unmount scroll listener — use onMounted/onUnmounted
import { onMounted, onUnmounted } from "vue";
onMounted(() => window.addEventListener("scroll", onScroll));
onUnmounted(() => window.removeEventListener("scroll", onScroll));
</script>

<style scoped>
.shadow-box {
    height: calc(100vh - 150px);
    position: sticky;
    top: 10px;
}
.list-header {
    border-bottom: 1px solid #dee2e6;
    border-radius: 10px 10px 0 0;
    margin: -10px;
    margin-bottom: 10px;
    padding: 10px;
}
.dark .list-header { background-color: #161b22; border-bottom: 0; }
.header-top { display: flex; justify-content: space-between; align-items: center; }
.header-filter { display: flex; align-items: center; }
@media (max-width: 770px) {
    .list-header { margin: -20px; margin-bottom: 10px; padding: 5px; }
}
.search-wrapper { display: flex; align-items: center; }
.search-icon { padding: 10px; color: #c0c0c0; }
.search-input { max-width: 15em; }
.stack-item { width: 100%; }
</style>
