import { currentLocale, localeDirection } from "../i18n";
import { defineComponent } from "vue";

function setPageLocale() {
    const html = document.documentElement;
    html.setAttribute("lang", currentLocale());
    html.setAttribute("dir", localeDirection());
}
const langModules = import.meta.glob("../lang/*.json");

export default defineComponent({
    data() {
        return {
            language: currentLocale(),
        };
    },

    watch: {
        async language(lang) {
            await this.changeLang(lang);
        },
    },

    async created() {
        if (this.language !== "en") {
            await this.changeLang(this.language);
        }
    },

    methods: {
        /**
         * Change the application language
         * @param {string} lang Language code to switch to
         * @returns {Promise<void>}
         */
        async changeLang(lang : string) {
            const message = (await langModules["../lang/" + lang + ".json"]()).default;
            this.$i18n.setLocaleMessage(lang, message);
            this.$i18n.locale = lang;
            localStorage.locale = lang;
            setPageLocale();
        }
    }
});
