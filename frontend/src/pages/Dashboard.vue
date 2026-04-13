<template>
    <div class="px-2 w-[98%] mx-auto" :class="layout.isNarrow ? 'pt-3' : 'pt-0'">
        <div class="flex gap-4">
            <!-- Collapsible sidebar: wide screens only -->
            <div v-if="!layout.isNarrow" class="sidebar-panel" :class="{ 'sidebar-collapsed': !layout.sidebarPinned }">
                <div class="sidebar-inner">
                    <div class="mb-3">
                        <router-link to="/compose" class="btn btn-primary w-full justify-center">
                            <PlusIcon :size="14" class="mr-1" /> {{ $t("compose") }}
                        </router-link>
                    </div>
                    <StackList :scrollbar="true" />
                </div>
            </div>

            <!-- Detail pane -->
            <div class="flex-1 min-w-0 mb-3">
                <router-view :key="$route.fullPath" />
            </div>
        </div>
    </div>

    <!-- Slide-over drawer: narrow screens (<1024px) -->
    <Teleport to="body">
        <template v-if="layout.isNarrow">
            <Transition name="drawer-fade">
                <div v-if="layout.sidebarOpen" class="fixed inset-0 bg-black/50 z-40 backdrop-blur-sm" @click="layout.sidebarOpen = false" />
            </Transition>
            <Transition name="drawer-slide">
                <div v-if="layout.sidebarOpen" class="fixed top-0 left-0 h-full w-72 z-50 flex flex-col bg-white dark:bg-[#0d1117] border-r border-gray-200 dark:border-[#1d2634] shadow-2xl">
                    <!-- User info -->
                    <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-[#1d2634]">
                        <div class="flex items-center gap-2">
                            <div class="flex items-center justify-center text-white font-bold text-xs rounded-full w-7 h-7 shrink-0" style="background: linear-gradient(135deg, var(--color-primary), var(--color-primary-end))">
                                {{ auth.usernameFirstChar }}
                            </div>
                            <span class="text-sm font-medium text-gray-800 dark:text-[#b1b8c0]">{{ auth.username }}</span>
                        </div>
                        <button class="mobile-header-btn" @click="layout.sidebarOpen = false">
                            <XIcon :size="18" />
                        </button>
                    </div>
                    <!-- Nav links -->
                    <nav class="px-3 pt-3 pb-2 border-b border-gray-200 dark:border-[#1d2634]">
                        <router-link to="/" class="drawer-nav-link" @click="layout.sidebarOpen = false">
                            <HomeIcon :size="16" /> {{ $t("home") }}
                        </router-link>
                        <router-link to="/console" class="drawer-nav-link" @click="layout.sidebarOpen = false">
                            <TerminalIcon :size="16" /> {{ $t("console") }}
                        </router-link>
                        <router-link to="/settings/general" class="drawer-nav-link" @click="layout.sidebarOpen = false">
                            <SettingsIcon :size="16" /> {{ $t("Settings") }}
                        </router-link>
                        <button class="drawer-nav-link w-full text-left" @click="scanFolder">
                            <RefreshCwIcon :size="16" /> {{ $t("scanFolder") }}
                        </button>
                    </nav>
                    <!-- Stack list -->
                    <div class="flex-1 overflow-hidden px-3 pt-3">
                        <div class="mb-3">
                            <router-link to="/compose" class="btn btn-primary w-full justify-center" @click="layout.sidebarOpen = false">
                                <PlusIcon :size="14" class="mr-1" /> {{ $t("compose") }}
                            </router-link>
                        </div>
                        <StackList :scrollbar="true" />
                    </div>
                    <!-- Logout -->
                    <div class="px-4 py-3 border-t border-gray-200 dark:border-[#1d2634]">
                        <button class="flex items-center gap-2 text-sm text-red-500 hover:text-red-600" @click="auth.logout">
                            <LogOutIcon :size="14" /> {{ $t("Logout") }}
                        </button>
                    </div>
                </div>
            </Transition>
        </template>
    </Teleport>
</template>

<script setup lang="ts">
import { watch } from "vue";
import { useRoute } from "vue-router";
import StackList from "../components/StackList.vue";
import { PlusIcon, XIcon, HomeIcon, TerminalIcon, SettingsIcon, RefreshCwIcon, LogOutIcon } from "lucide-vue-next";
import { ALL_ENDPOINTS } from "../utils";
import { useSocketStore } from "../stores/socket";
import { useAuthStore } from "../stores/auth";
import { useLayoutStore } from "../stores/layout";

const socketStore = useSocketStore();
const auth = useAuthStore();
const layout = useLayoutStore();
const route = useRoute();

watch(() => route.fullPath, () => {
    layout.sidebarOpen = false;
});

function scanFolder() {
    layout.sidebarOpen = false;
    socketStore.emitAgent(ALL_ENDPOINTS, "requestStackList", () => { });
}
</script>
