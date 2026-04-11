<template>
    <div :class="classes">
        <div v-if="! $root.socketIO.connected && ! $root.socketIO.firstConnect" class="lost-connection">
            <div class="px-4">
                {{ $root.socketIO.connectionErrorMsg }}
                <div v-if="$root.socketIO.showReverseProxyGuide">
                    {{ $t("reverseProxyMsg1") }} <a href="https://github.com/louislam/uptime-kuma/wiki/Reverse-Proxy" target="_blank">{{ $t("reverseProxyMsg2") }}</a>
                </div>
            </div>
        </div>

        <!-- Desktop header -->
        <header v-if="! $root.isMobile" class="flex flex-wrap items-center justify-center py-3 mb-3 border-b border-gray-200 dark:border-[#1d2634] dark:bg-[#161b22]">
            <router-link to="/" class="flex items-center mb-3 md:mb-0 md:mr-auto text-dark no-underline ms-4">
                <object class="me-2 ms-4" width="40" height="40" data="/icon.svg" />
                <span class="text-2xl font-bold text-gray-900 dark:text-[#f0f6fc]">Dockge</span>
            </router-link>

            <ul class="nav nav-pills me-6">
                <li v-if="$root.loggedIn" class="nav-item me-2">
                    <router-link to="/" class="nav-link">
                        <HomeIcon :size="16" /> {{ $t("home") }}
                    </router-link>
                </li>

                <li v-if="$root.loggedIn" class="nav-item me-2">
                    <router-link to="/console" class="nav-link">
                        <TerminalIcon :size="16" /> {{ $t("console") }}
                    </router-link>
                </li>

                <li v-if="$root.loggedIn" class="nav-item">
                    <HMenu as="div" class="relative">
                        <HMenuButton class="nav-link cursor-pointer flex gap-2 items-center rounded-md bg-black/10 dark:bg-white/10 px-3 py-2">
                            <div class="profile-pic flex items-center justify-center text-white font-bold text-xs rounded-full w-6 h-6" style="background: linear-gradient(135deg, var(--color-primary), var(--color-primary-end))">
                                {{ $root.usernameFirstChar }}
                            </div>
                            <ChevronDownIcon :size="14" />
                        </HMenuButton>

                        <transition
                            enter-active-class="transition duration-100 ease-out"
                            enter-from-class="transform scale-95 opacity-0"
                            enter-to-class="transform scale-100 opacity-1"
                            leave-active-class="transition duration-75 ease-in"
                            leave-from-class="transform scale-100 opacity-1"
                            leave-to-class="transform scale-95 opacity-0"
                        >
                            <HMenuItems class="absolute right-0 mt-2 w-48 origin-top-right rounded-2xl overflow-hidden shadow-xl bg-white dark:bg-[#0d1117] border border-gray-100 dark:border-[#1d2634] focus:outline-none z-50">
                                <div class="px-4 py-3 text-sm border-b border-gray-100 dark:border-[#1d2634]">
                                    <i18n-t v-if="$root.username != null" tag="span" keypath="signedInDisp">
                                        <strong>{{ $root.username }}</strong>
                                    </i18n-t>
                                    <span v-if="$root.username == null">{{ $t("signedInDispDisabled") }}</span>
                                </div>

                                <HMenuItem v-slot="{ active }">
                                    <button
                                        class="w-full text-left flex items-center gap-2 px-4 py-3 text-sm"
                                        :class="active ? 'bg-gray-50 dark:bg-[#070a10]' : ''"
                                        @click="scanFolder"
                                    >
                                        <RefreshCwIcon :size="14" /> {{ $t("scanFolder") }}
                                    </button>
                                </HMenuItem>

                                <HMenuItem v-slot="{ active }">
                                    <router-link
                                        to="/settings/general"
                                        class="flex items-center gap-2 px-4 py-3 text-sm"
                                        :class="active ? 'bg-gray-50 dark:bg-[#070a10]' : ''"
                                    >
                                        <SettingsIcon :size="14" /> {{ $t("Settings") }}
                                    </router-link>
                                </HMenuItem>

                                <HMenuItem v-slot="{ active }">
                                    <button
                                        class="w-full text-left flex items-center gap-2 px-4 py-3 text-sm border-t border-gray-100 dark:border-[#1d2634]"
                                        :class="active ? 'bg-gray-50 dark:bg-[#070a10]' : ''"
                                        @click="$root.logout"
                                    >
                                        <LogOutIcon :size="14" /> {{ $t("Logout") }}
                                    </button>
                                </HMenuItem>
                            </HMenuItems>
                        </transition>
                    </HMenu>
                </li>
            </ul>
        </header>

        <main>
            <div v-if="$root.socketIO.connecting" class="container mt-5">
                <h4>{{ $t("connecting...") }}</h4>
            </div>

            <router-view v-if="$root.loggedIn" />
            <Login v-if="! $root.loggedIn && $root.allowLoginDialog" />
        </main>
    </div>
</template>

<script>
import Login from "../components/Login.vue";
import { Menu as HMenu, MenuButton as HMenuButton, MenuItems as HMenuItems, MenuItem as HMenuItem } from "@headlessui/vue";
import { HomeIcon, TerminalIcon, ChevronDownIcon, RefreshCwIcon, SettingsIcon, LogOutIcon } from "lucide-vue-next";
import { ALL_ENDPOINTS } from "../../../common/util-common";

export default {
    components: {
        Login,
        HMenu,
        HMenuButton,
        HMenuItems,
        HMenuItem,
        HomeIcon,
        TerminalIcon,
        ChevronDownIcon,
        RefreshCwIcon,
        SettingsIcon,
        LogOutIcon,
    },

    computed: {
        classes() {
            const classes = {};
            classes[this.$root.theme] = true;
            classes["mobile"] = this.$root.isMobile;
            return classes;
        },
    },

    methods: {
        scanFolder() {
            this.$root.emitAgent(ALL_ENDPOINTS, "requestStackList", (res) => {
                this.$root.toastRes(res);
            });
        },
    },
};
</script>

<style scoped>
main {
    min-height: calc(100vh - 160px);
}
</style>
