import { createI18n } from "vue-i18n";
import en from "./lang/en.json";

const languageList = {
    "bg-BG": "Български",
    "es": "Español",
    "de": "Deutsch",
    "fr": "Français",
    "pl-PL": "Polski",
    "pt": "Português",
    "pt-BR": "Português-Brasil",
    "sl": "Slovenščina",
    "tr": "Türkçe",
    "zh-CN": "简体中文",
    "zh-TW": "繁體中文(台灣)",
    "ur": "Urdu",
    "ko-KR": "한국어",
    "ru": "Русский",
    "cs-CZ": "Čeština",
    "ar": "العربية",
    "th": "ไทย",
    "it-IT": "Italiano",
    "sv-SE": "Svenska",
    "uk-UA": "Українська",
    "da": "Dansk",
    "ja": "日本語",
    "nl": "Nederlands",
    "ro": "Română",
    "id": "Bahasa Indonesia (Indonesian)",
    "vi": "Tiếng Việt",
    "hu": "Magyar",
    "ca": "Català",
    "ga": "Gaeilge",
    "de-CH": "Schwiizerdütsch",
    "mag": "मगही",
    "mai": "मैथिली",
};

// vue-i18n's createI18n types require 'any' for the messages map when using
// dynamic locale loading with mixed message shapes.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const messages: Record<string, any> = { en };

for (const lang in languageList) {
    messages[lang] = {
        languageName: (languageList as Record<string, string>)[lang],
    };
}

const rtlLangs = [ "fa", "ar-SY", "ur", "ar" ];

export const currentLocale = () => {
    const ll = languageList as Record<string, string>;
    return localStorage.getItem("locale")
        || (ll[navigator.language] && navigator.language)
        || (ll[navigator.language.substring(0, 2)] && navigator.language.substring(0, 2))
        || "en";
};

export const localeDirection = () => {
    return rtlLangs.includes(currentLocale()) ? "rtl" : "ltr";
};

export const i18n = createI18n({
    legacy: false,
    locale: currentLocale(),
    fallbackLocale: "en",
    silentFallbackWarn: true,
    silentTranslationWarn: true,
    messages: messages,
});
