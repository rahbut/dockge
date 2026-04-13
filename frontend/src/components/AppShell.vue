<template>
    <div :class="rootClasses">
        <!-- Lost connection banner -->
        <div v-if="!socketStore.socketIO.connected && !socketStore.socketIO.firstConnect" class="lost-connection">
            <div class="px-4">
                {{ socketStore.socketIO.connectionErrorMsg }}
                <div v-if="socketStore.socketIO.showReverseProxyGuide">
                    {{ $t("reverseProxyMsg1") }} <a href="https://github.com/louislam/uptime-kuma/wiki/Reverse-Proxy" target="_blank">{{ $t("reverseProxyMsg2") }}</a>
                </div>
            </div>
        </div>

        <!-- Desktop header (≥1024px) -->
        <header v-if="!layout.isNarrow" class="flex flex-wrap items-center justify-center py-3 mb-3 border-b border-gray-200 dark:border-[#1d2634] dark:bg-[#161b22]">
            <div class="flex items-center mb-3 md:mb-0 md:mr-auto ms-4 gap-3">
                <button v-if="auth.loggedIn" class="mobile-header-btn" :title="layout.sidebarPinned ? 'Hide sidebar' : 'Show sidebar'" @click="layout.sidebarPinned = !layout.sidebarPinned">
                    <PanelLeftCloseIcon v-if="layout.sidebarPinned" :size="18" />
                    <PanelLeftOpenIcon v-else :size="18" />
                </button>
                <router-link to="/" class="flex items-center gap-2 text-dark no-underline">
                    <object class="me-2" width="40" height="40" data="/icon.svg" />
                    <span class="text-2xl font-bold text-gray-900 dark:text-[#f0f6fc]">Dockge</span>
                </router-link>
            </div>

            <ul class="nav nav-pills me-6">
                <li v-if="auth.loggedIn" class="nav-item me-2">
                    <router-link to="/" class="nav-link"><HomeIcon :size="16" /> {{ $t("home") }}</router-link>
                </li>
                <li v-if="auth.loggedIn" class="nav-item me-2">
                    <router-link to="/console" class="nav-link"><TerminalIcon :size="16" /> {{ $t("console") }}</router-link>
                </li>
                <li v-if="auth.loggedIn" class="nav-item me-2">
                    <button class="nav-link" :title="themeLabel" @click="cycleTheme">
                        <SunIcon v-if="themeStore.userTheme === 'light'" :size="16" />
                        <MoonIcon v-else-if="themeStore.userTheme === 'dark'" :size="16" />
                        <SunMoonIcon v-else :size="16" />
                    </button>
                </li>
                <li v-if="auth.loggedIn" class="nav-item">
                    <HMenu as="div" class="relative">
                        <HMenuButton class="nav-link cursor-pointer flex gap-2 items-center rounded-md bg-black/10 dark:bg-white/10 px-3 py-2">
                            <div class="flex items-center justify-center text-white font-bold text-xs rounded-full w-6 h-6" style="background: linear-gradient(135deg, var(--color-primary), var(--color-primary-end))">
                                {{ auth.usernameFirstChar }}
                            </div>
                            <ChevronDownIcon :size="14" />
                        </HMenuButton>
                        <transition enter-active-class="transition duration-100 ease-out" enter-from-class="transform scale-95 opacity-0" enter-to-class="transform scale-100 opacity-1" leave-active-class="transition duration-75 ease-in" leave-from-class="transform scale-100 opacity-1" leave-to-class="transform scale-95 opacity-0">
                            <HMenuItems class="absolute right-0 mt-2 w-48 origin-top-right rounded-2xl overflow-hidden shadow-xl bg-white dark:bg-[#0d1117] border border-gray-100 dark:border-[#1d2634] focus:outline-none z-50">
                                <div class="px-4 py-3 text-sm border-b border-gray-100 dark:border-[#1d2634]">
                                    <i18n-t v-if="auth.username != null" tag="span" keypath="signedInDisp"><strong>{{ auth.username }}</strong></i18n-t>
                                    <span v-if="auth.username == null">{{ $t("signedInDispDisabled") }}</span>
                                </div>
                                <HMenuItem v-slot="{ active }">
                                    <button class="w-full text-left flex items-center gap-2 px-4 py-3 text-sm" :class="active ? 'bg-gray-50 dark:bg-[#070a10]' : ''" @click="scanFolder">
                                        <RefreshCwIcon :size="14" /> {{ $t("scanFolder") }}
                                    </button>
                                </HMenuItem>
                                <HMenuItem v-slot="{ active }">
                                    <router-link to="/settings/general" class="flex items-center gap-2 px-4 py-3 text-sm" :class="active ? 'bg-gray-50 dark:bg-[#070a10]' : ''">
                                        <SettingsIcon :size="14" /> {{ $t("Settings") }}
                                    </router-link>
                                </HMenuItem>
                                <HMenuItem v-slot="{ active }">
                                    <button class="w-full text-left flex items-center gap-2 px-4 py-3 text-sm border-t border-gray-100 dark:border-[#1d2634]" :class="active ? 'bg-gray-50 dark:bg-[#070a10]' : ''" @click="auth.logout">
                                        <LogOutIcon :size="14" /> {{ $t("Logout") }}
                                    </button>
                                </HMenuItem>
                            </HMenuItems>
                        </transition>
                    </HMenu>
                </li>
            </ul>
        </header>

        <!-- Mobile / narrow header (<1024px) -->
        <header v-if="layout.isNarrow" class="mobile-header">
            <button v-if="auth.loggedIn" class="mobile-header-btn" @click="layout.sidebarOpen = !layout.sidebarOpen">
                <XIcon v-if="layout.sidebarOpen" :size="20" />
                <MenuIcon v-else :size="20" />
            </button>
            <router-link to="/" class="flex items-center gap-2 no-underline">
                <object width="28" height="28" data="/icon.svg" style="pointer-events:none" />
                <span class="text-lg font-bold text-gray-900 dark:text-[#f0f6fc]">Dockge</span>
            </router-link>
            <button v-if="auth.loggedIn" class="mobile-header-btn" :title="themeLabel" @click="cycleTheme">
                <SunIcon v-if="themeStore.userTheme === 'light'" :size="18" />
                <MoonIcon v-else-if="themeStore.userTheme === 'dark'" :size="18" />
                <SunMoonIcon v-else :size="18" />
            </button>
        </header>

        <!-- Main content slot -->
        <main>
            <div v-if="socketStore.socketIO.connecting" class="container mt-5">
                <h4>{{ $t("connecting...") }}</h4>
            </div>
            <slot v-if="auth.loggedIn" />
            <Login v-if="!auth.loggedIn && auth.allowLoginDialog" />
        </main>
    </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import Login from "./Login.vue";
import { Menu as HMenu, MenuButton as HMenuButton, MenuItems as HMenuItems, MenuItem as HMenuItem } from "@headlessui/vue";
import { HomeIcon, TerminalIcon, ChevronDownIcon, RefreshCwIcon, SettingsIcon, LogOutIcon, SunIcon, MoonIcon, SunMoonIcon, MenuIcon, XIcon, PanelLeftCloseIcon, PanelLeftOpenIcon } from "lucide-vue-next";
import { ALL_ENDPOINTS } from "../utils";
import { useSocketStore } from "../stores/socket";
import { useAuthStore } from "../stores/auth";
import { useThemeStore } from "../stores/theme";
import { useLayoutStore } from "../stores/layout";

const socketStore = useSocketStore();
const auth = useAuthStore();
const themeStore = useThemeStore();
const layout = useLayoutStore();

const rootClasses = computed(() => ({
    [themeStore.theme]: true,
    mobile: layout.isMobile,
}));

const themeLabel = computed(() => {
    const labels: Record<string, string> = { light: "Light mode", dark: "Dark mode", auto: "Auto (system)" };
    return labels[themeStore.userTheme] ?? "Auto (system)";
});

function cycleTheme() {
    const order = [ "light", "dark", "auto" ];
    themeStore.userTheme = order[(order.indexOf(themeStore.userTheme ?? "dark") + 1) % order.length];
}

function scanFolder() {
    socketStore.emitAgent(ALL_ENDPOINTS, "requestStackList", () => {});
}
</script>

<style scoped>
main {
    min-height: calc(100vh - 160px);
}
.mobile main {
    min-height: calc(100vh - 50px);
    padding-top: 0.75rem;
}
</style>
