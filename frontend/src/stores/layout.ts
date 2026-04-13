import { defineStore } from "pinia";
import { ref, watch } from "vue";

export const useLayoutStore = defineStore("layout", () => {
    const isMobile = ref(false);
    const isNarrow = ref(false);
    const sidebarOpen = ref(false);
    const sidebarPinned = ref(localStorage.getItem("sidebarPinned") !== "0");

    function onResize() {
        isMobile.value = window.innerWidth < 768;
        isNarrow.value = window.innerWidth < 1024;
        if (!isNarrow.value) {
            sidebarOpen.value = false;
        }
    }

    function init() {
        onResize();
        window.addEventListener("resize", onResize);
    }

    function destroy() {
        window.removeEventListener("resize", onResize);
    }

    watch(sidebarPinned, (val) => {
        localStorage.setItem("sidebarPinned", val ? "1" : "0");
    });

    return { isMobile, isNarrow, sidebarOpen, sidebarPinned, init, destroy };
});
