import { defineStore } from "pinia";
import { ref, watch } from "vue";
import { currentLocale, localeDirection, i18n } from "../i18n";

const langModules = import.meta.glob("../lang/*.json");

function setPageLocale() {
    const html = document.documentElement;
    html.setAttribute("lang", currentLocale());
    html.setAttribute("dir", localeDirection());
}

export const useLangStore = defineStore("lang", () => {
    const language = ref(currentLocale());

    async function changeLang(lang: string) {
        const mod = langModules["../lang/" + lang + ".json"];
        if (mod) {
            // vue-i18n message modules don't ship TS types for their JSON default
            const loaded = await mod() as { default: Record<string, unknown> };
            i18n.global.setLocaleMessage(lang, loaded.default);
        }
        // Composition API locale is a Ref — cast required as vue-i18n types don't expose it directly
        (i18n.global.locale as unknown as { value: string }).value = lang;
        localStorage.setItem("locale", lang);
        setPageLocale();
    }

    async function init() {
        if (language.value !== "en") {
            await changeLang(language.value);
        }
    }

    watch(language, async (lang) => {
        await changeLang(lang);
    });

    return { language, changeLang, init };
});
