<template>
    <div>
        <h1 v-show="show" class="mb-3">
            {{ $t("Settings") }}
        </h1>

        <div class="shadow-box shadow-box-settings">
            <div class="flex flex-wrap">
                <div v-if="showSubMenu" class="settings-menu w-full lg:w-1/4 md:w-5/12">
                    <router-link
                        v-for="(item, key) in subMenus"
                        :key="key"
                        :to="`/settings/${key}`"
                    >
                        <div class="menu-item">
                            {{ item.title }}
                        </div>
                    </router-link>

                    <a v-if="$root.isMobile && $root.loggedIn && $root.socket?.token !== 'autoLogin'" class="logout cursor-pointer" @click.prevent="$root.logout">
                        <div class="menu-item flex items-center gap-2">
                            <LogOutIcon :size="14" /> {{ $t("Logout") }}
                        </div>
                    </a>
                </div>
                <div class="settings-content flex-1 min-w-0 lg:w-3/4 md:w-7/12">
                    <div v-if="currentPage" class="settings-content-header">
                        {{ subMenus[currentPage].title }}
                    </div>
                    <div class="mx-3">
                        <router-view v-slot="{ Component }">
                            <transition name="slide-fade" appear>
                                <component :is="Component" />
                            </transition>
                        </router-view>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import { useRoute } from "vue-router";
import { LogOutIcon } from "lucide-vue-next";

export default {
    components: { LogOutIcon },
    data() {
        return {
            show: true,
            settings: {},
            settingsLoaded: false,
        };
    },

    computed: {
        currentPage() {
            let pathSplit = useRoute().path.split("/");
            let pathEnd = pathSplit[pathSplit.length - 1];
            if (!pathEnd || pathEnd === "settings") {
                return null;
            }
            return pathEnd;
        },

        showSubMenu() {
            if (this.$root.isMobile) {
                return !this.currentPage;
            } else {
                return true;
            }
        },

        subMenus() {
            return {
                general: {
                    title: this.$t("general"),
                },
                appearance: {
                    title: this.$t("Appearance"),
                },
                security: {
                    title: this.$t("Security"),
                },
                globalEnv: {
                    title: this.$t("GlobalEnv"),
                },
                about: {
                    title: this.$t("About"),
                },
            };
        },
    },

    watch: {
        "$root.isMobile"() {
            this.loadGeneralPage();
        }
    },

    mounted() {
        this.loadSettings();
        this.loadGeneralPage();
    },

    methods: {

        /**
         * Load the general settings page
         * For desktop only, on mobile do nothing
         */
        loadGeneralPage() {
            if (!this.currentPage && !this.$root.isMobile) {
                this.$router.push("/settings/appearance");
            }
        },

        /** Load settings from server */
        loadSettings() {
            this.$root.getSocket().emit("getSettings", (res) => {
                this.settings = res.data;
                this.settingsLoaded = true;
            });
        },

        /**
         * Callback for saving settings
         * @callback saveSettingsCB
         * @param {Object} res Result of operation
         */

        /**
         * Save Settings
         * @param {saveSettingsCB} [callback]
         * @param {string} [currentPassword] Only need for disableAuth to true
         */
        saveSettings(callback, currentPassword) {
            this.$root.getSocket().emit("setSettings", this.settings, currentPassword, (res) => {
                this.$root.toastRes(res);
                this.loadSettings();

                if (callback) {
                    callback();
                }
            });
        },
    }
};
</script>

<style scoped>

.shadow-box-settings {
    padding: 20px;
    min-height: calc(100vh - 155px);
}

footer {
    color: #aaa;
    font-size: 13px;
    margin-top: 20px;
    padding-bottom: 30px;
    text-align: center;
}

.settings-menu {
    a {
        text-decoration: none !important;
    }

    .menu-item {
        border-radius: 10px;
        margin: 0.5em;
        padding: 0.7em 1em;
        cursor: pointer;
        border-left-width: 0;
        transition: all ease-in-out 0.1s;
    }

    .menu-item:hover { background: #e7faec; }
    .dark .menu-item:hover { background: #161b22; }
    .active .menu-item {
        background: #e7faec;
        border-left: 4px solid #74c2ff;
        border-top-left-radius: 0;
        border-bottom-left-radius: 0;
    }
    .dark .active .menu-item { background: #161b22; }
}

.settings-content-header {
    width: calc(100% + 20px);
    border-bottom: 1px solid #dee2e6;
    border-radius: 0 10px 0 0;
    margin-top: -20px;
    margin-right: -20px;
    padding: 12.5px 1em;
    font-size: 26px;
}
.dark .settings-content-header { background: #161b22; border-bottom: 0; }
.mobile .settings-content-header { padding: 15px 0 0 0; }
.dark .mobile .settings-content-header { background-color: transparent; }

.logout { color: #dc3545 !important; }
</style>
