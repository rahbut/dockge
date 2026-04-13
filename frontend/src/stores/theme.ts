import { defineStore } from "pinia";
import { ref, computed, watch } from "vue";

export const useThemeStore = defineStore("theme", () => {
    const system = ref(window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
    const userTheme = ref<string>(localStorage.getItem("theme") ?? "dark");

    const theme = computed(() => {
        return userTheme.value === "auto" ? system.value : userTheme.value;
    });

    const isDark = computed(() => theme.value === "dark");

    function updateThemeColorMeta() {
        const meta = document.querySelector("#theme-color");
        if (meta) {
            meta.setAttribute("content", theme.value === "dark" ? "#161B22" : "#5cdd8b");
        }
    }

    function applyTheme() {
        document.body.classList.remove("light", "dark");
        document.body.classList.add(theme.value);
        updateThemeColorMeta();
    }

    function init() {
        applyTheme();
    }

    watch(userTheme, (val) => {
        localStorage.setItem("theme", val);
        applyTheme();
    });

    watch(system, () => {
        if (userTheme.value === "auto") {
            applyTheme();
        }
    });

    // Listen for OS theme changes
    window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", (e) => {
        system.value = e.matches ? "dark" : "light";
    });

    return { system, userTheme, theme, isDark, init, updateThemeColorMeta };
});
