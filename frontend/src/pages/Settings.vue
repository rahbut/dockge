<template>
    <div>
        <h1 v-show="show" class="mb-3">{{ $t("Settings") }}</h1>

        <div class="shadow-box shadow-box-settings">
            <div class="flex flex-wrap">
                <div v-if="showSubMenu" class="settings-menu w-full lg:w-1/4 md:w-5/12">
                    <router-link v-for="(item, key) in subMenus" :key="key" :to="`/settings/${key}`">
                        <div class="menu-item">{{ item.title }}</div>
                    </router-link>
                    <a v-if="layout.isMobile && auth.loggedIn" class="logout cursor-pointer" @click.prevent="auth.logout">
                        <div class="menu-item flex items-center gap-2">
                            <LogOutIcon :size="14" /> {{ $t("Logout") }}
                        </div>
                    </a>
                </div>
                <div class="settings-content flex-1 min-w-0 lg:w-3/4 md:w-7/12">
                    <div v-if="currentPage" class="settings-content-header">
                        {{ subMenus[currentPage]?.title }}
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

<script setup lang="ts">
import { computed, watch, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { LogOutIcon } from "lucide-vue-next";
import { useLayoutStore } from "../stores/layout";
import { useAuthStore } from "../stores/auth";
import { useSettingsStore } from "../stores/settings";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const layout = useLayoutStore();
const auth = useAuthStore();
const settingsStore = useSettingsStore();

const show = computed(() => true);

const currentPage = computed(() => {
    const parts = route.path.split("/");
    const end = parts[parts.length - 1];
    return (!end || end === "settings") ? null : end;
});

const showSubMenu = computed(() =>
    layout.isMobile ? !currentPage.value : true
);

const subMenus = computed((): Record<string, { title: string }> => ({
    general: { title: t("general") },
    security: { title: t("Security") },
    globalEnv: { title: t("GlobalEnv") },
    maintenance: { title: t("Maintenance") },
    about: { title: t("About") },
}));

function loadGeneralPage() {
    if (!currentPage.value && !layout.isMobile) {
        router.push("/settings/general");
    }
}

watch(() => layout.isMobile, () => loadGeneralPage());

onMounted(() => {
    settingsStore.loadSettings();
    loadGeneralPage();
});
</script>

<style scoped>
.shadow-box-settings { padding: 20px; min-height: calc(100vh - 155px); }
footer { color: #aaa; font-size: 13px; margin-top: 20px; padding-bottom: 30px; text-align: center; }
.settings-menu {
    a { text-decoration: none !important; }
    .menu-item { border-radius: 10px; margin: 0.5em; padding: 0.7em 1em; cursor: pointer; border-left-width: 0; transition: all ease-in-out 0.1s; }
    .menu-item:hover { background: linear-gradient(135deg, #e7faec, #d4f5e0); color: #0d1f1e; }
    .dark .menu-item:hover { background: var(--color-dark-bg2); color: var(--color-dark-font); }
    .active .menu-item { background: linear-gradient(135deg, #cdf8f4, #d4f5e0); color: #0d1f1e; }
    .dark .active .menu-item { background: linear-gradient(135deg, #1a3a38, #1a3028); color: #cdf8f4; }
}
.settings-content-header { width: calc(100% + 20px); border-bottom: 1px solid #dee2e6; border-radius: 0 10px 0 0; margin-top: -20px; margin-right: -20px; padding: 12.5px 1em; font-size: 26px; background: linear-gradient(135deg, #f0fdf8, #e7faec); color: #0d1f1e; }
.dark .settings-content-header { background: linear-gradient(135deg, #161b22, #0d1117); color: var(--color-dark-font); border-bottom: 0; }
.mobile .settings-content-header { padding: 15px 0 0 0; }
.dark .mobile .settings-content-header { background-color: transparent; }
.logout { color: #dc3545 !important; }
</style>
